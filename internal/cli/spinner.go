package cli

import (
	"fmt"
	"io"
	"math/rand/v2"
	"sync"
	"time"
)

const (
	scanLoaderInterval = 100 * time.Millisecond
	minFactInterval    = time.Second
	factIntervalCount  = 5
	hideCursor         = "\x1b[?25l"
	showCursor         = "\x1b[?25h"
	clearLine          = "\r\x1b[2K"
	moveUpLine         = "\x1b[1A"
	resetStyle         = "\x1b[0m"
)

var (
	scanLoaderFrames = []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	scanLoaderColors = []string{"\x1b[96m", "\x1b[94m", "\x1b[95m", "\x1b[35m", "\x1b[93m", "\x1b[36m"}
	scanBeamColors   = []string{"\x1b[38;5;51m", "\x1b[38;5;159m", "\x1b[38;5;183m", "\x1b[38;5;219m", "\x1b[38;5;228m"}
	scanFactColors   = []string{"\x1b[38;5;183m", "\x1b[38;5;159m", "\x1b[38;5;219m", "\x1b[38;5;228m"}
)

type scanLoader struct {
	writer        io.Writer
	enabled       bool
	color         bool
	width         int
	frameInterval time.Duration
	minFactDelay  time.Duration
	deck          factDeck
	nextFactDelay func() time.Duration
	done          chan bool
	stopped       chan struct{}
	stopOnce      sync.Once
	rendered      bool
}

type scanLoaderOptions struct {
	frameInterval time.Duration
	minFactDelay  time.Duration
	width         int
	facts         []string
	factOrder     []int
	nextFactDelay func() time.Duration
}

type factDeck struct {
	facts    []string
	order    []int
	position int
}

func startScanLoader(writer io.Writer, enabled, color bool, width int) *scanLoader {
	return startScanLoaderWithOptions(writer, enabled, color, scanLoaderOptions{
		frameInterval: scanLoaderInterval,
		minFactDelay:  minFactInterval,
		width:         width,
		facts:         scanFacts,
		factOrder:     rand.Perm(len(scanFacts)),
		nextFactDelay: randomFactDelay,
	})
}

func startScanLoaderWithOptions(writer io.Writer, enabled, color bool, options scanLoaderOptions) *scanLoader {
	if options.frameInterval <= 0 {
		options.frameInterval = scanLoaderInterval
	}
	if options.nextFactDelay == nil {
		options.nextFactDelay = randomFactDelay
	}
	if options.minFactDelay <= 0 {
		options.minFactDelay = minFactInterval
	}
	loader := &scanLoader{
		writer: writer, enabled: enabled && writer != nil, color: color,
		width: options.width, frameInterval: options.frameInterval, minFactDelay: options.minFactDelay,
		deck: newFactDeck(options.facts, options.factOrder), nextFactDelay: options.nextFactDelay,
		done: make(chan bool, 1), stopped: make(chan struct{}),
	}
	if loader.enabled {
		go loader.run()
	}
	return loader
}

func (l *scanLoader) Stop(ready bool) {
	if !l.enabled {
		return
	}
	l.stopOnce.Do(func() {
		l.done <- ready
		<-l.stopped
	})
}

func (l *scanLoader) run() {
	defer close(l.stopped)
	_, _ = fmt.Fprint(l.writer, hideCursor)
	defer func() { _, _ = fmt.Fprint(l.writer, showCursor) }()

	ticker := time.NewTicker(l.frameInterval)
	defer ticker.Stop()
	factTimer, factTicks := l.factTimer()
	if factTimer != nil {
		defer factTimer.Stop()
	}
	frame := 0
	l.render(frame)

	for {
		select {
		case ready := <-l.done:
			l.clearRendered()
			if ready {
				_, _ = fmt.Fprintf(l.writer, "%s\n", l.readyText())
			}
			return
		case <-ticker.C:
			frame++
			l.render(frame)
		case <-factTicks:
			if l.deck.advance() {
				l.render(frame)
			}
			if l.deck.hasNext() {
				factTimer.Reset(l.factDelay())
			} else {
				factTicks = nil
			}
		}
	}
}

func (l *scanLoader) render(frame int) {
	beam := scanBeamLine(frame, l.width)
	fact := fitLoaderFact(l.deck.current(), l.width)
	if l.color {
		beam = colorizeScanBeam(beam, frame)
		fact = scanFactColors[l.deck.position%len(scanFactColors)] + fact + resetStyle
	}
	l.clearRendered()
	_, _ = fmt.Fprintf(l.writer, "%s%s\r\n%s  %s", clearLine, beam, clearLine, fact)
	l.rendered = true
}

func (l *scanLoader) clearRendered() {
	if !l.rendered {
		return
	}
	_, _ = fmt.Fprintf(l.writer, "%s%s%s", clearLine, moveUpLine, clearLine)
	l.rendered = false
}

func (l *scanLoader) readyText() string {
	if !l.color {
		return "✓ beacon ready"
	}
	return "\x1b[92m✓\x1b[0m beacon ready"
}

func (l *scanLoader) factTimer() (*time.Timer, <-chan time.Time) {
	if !l.deck.hasNext() {
		return nil, nil
	}
	timer := time.NewTimer(l.factDelay())
	return timer, timer.C
}

func (l *scanLoader) factDelay() time.Duration {
	delay := l.nextFactDelay()
	if delay < l.minFactDelay {
		return l.minFactDelay
	}
	return delay
}

func randomFactDelay() time.Duration {
	return time.Duration(rand.IntN(factIntervalCount)+1) * time.Second
}

func scanBeamLine(frame, width int) string {
	if width <= 0 {
		width = 80
	}
	arc := scanLoaderFrames[frame%len(scanLoaderFrames)]
	if width == 1 {
		return arc
	}
	trackWidth := width - 2
	track := make([]rune, trackWidth)
	for index := range track {
		track[index] = '·'
	}
	if trackWidth == 0 {
		return arc + " "
	}
	position, forward := scanBeamPosition(frame, trackWidth)
	track[position] = '✦'
	trailLength := min(8, trackWidth-1)
	if forward {
		for distance := 1; distance <= trailLength && position-distance >= 0; distance++ {
			track[position-distance] = '━'
		}
		if edge := position - trailLength - 1; edge >= 0 {
			track[edge] = '╺'
		}
	} else {
		for distance := 1; distance <= trailLength && position+distance < trackWidth; distance++ {
			track[position+distance] = '━'
		}
		if edge := position + trailLength + 1; edge < trackWidth {
			track[edge] = '╸'
		}
	}
	return arc + " " + string(track)
}

func scanBeamPosition(frame, width int) (int, bool) {
	if width <= 1 {
		return 0, true
	}
	cycle := 2 * (width - 1)
	step := frame % cycle
	if step <= width-1 {
		return step, true
	}
	return cycle - step, false
}

func colorizeScanBeam(beam string, frame int) string {
	runes := []rune(beam)
	if len(runes) == 0 {
		return beam
	}
	arc := scanLoaderColors[frame%len(scanLoaderColors)] + string(runes[0]) + resetStyle
	if len(runes) == 1 {
		return arc
	}
	track := scanBeamColors[frame%len(scanBeamColors)] + string(runes[1:]) + resetStyle
	return arc + track
}

func newFactDeck(facts []string, order []int) factDeck {
	deck := factDeck{facts: append([]string(nil), facts...)}
	seen := make([]bool, len(facts))
	for _, index := range order {
		if index >= 0 && index < len(facts) && !seen[index] {
			deck.order = append(deck.order, index)
			seen[index] = true
		}
	}
	for index := range facts {
		if !seen[index] {
			deck.order = append(deck.order, index)
		}
	}
	return deck
}

func (d factDeck) current() string {
	if len(d.order) == 0 {
		return "beacon scanning the horizon…"
	}
	return d.facts[d.order[d.position]]
}

func (d factDeck) hasNext() bool {
	return d.position+1 < len(d.order)
}

func (d *factDeck) advance() bool {
	if !d.hasNext() {
		return false
	}
	d.position++
	return true
}

func fitLoaderFact(fact string, width int) string {
	if width <= 0 {
		return fact
	}
	available := width - 2
	if available <= 0 {
		return ""
	}
	runes := []rune(fact)
	if len(runes) <= available {
		return fact
	}
	if available == 1 {
		return "…"
	}
	return string(runes[:available-1]) + "…"
}
