package cli

import (
	"fmt"
	"io"
	"sync"
	"time"
)

const (
	scanLoaderInterval = 100 * time.Millisecond
	hideCursor         = "\x1b[?25l"
	showCursor         = "\x1b[?25h"
	clearLine          = "\r\x1b[2K"
	resetStyle         = "\x1b[0m"
)

var (
	scanLoaderFrames = []string{"◜", "◠", "◝", "◞", "◡", "◟"}
	scanLoaderColors = []string{"\x1b[96m", "\x1b[94m", "\x1b[95m", "\x1b[35m", "\x1b[93m", "\x1b[36m"}
)

type scanLoader struct {
	writer   io.Writer
	enabled  bool
	color    bool
	interval time.Duration
	done     chan bool
	stopped  chan struct{}
	stopOnce sync.Once
}

func startScanLoader(writer io.Writer, enabled, color bool) *scanLoader {
	return startScanLoaderWithInterval(writer, enabled, color, scanLoaderInterval)
}

func startScanLoaderWithInterval(writer io.Writer, enabled, color bool, interval time.Duration) *scanLoader {
	loader := &scanLoader{
		writer: writer, enabled: enabled && writer != nil, color: color, interval: interval,
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

	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()
	frame := 0
	l.render(frame)

	for {
		select {
		case ready := <-l.done:
			if ready {
				_, _ = fmt.Fprintf(l.writer, "%s%s\n", clearLine, l.readyText())
			} else {
				_, _ = fmt.Fprint(l.writer, clearLine)
			}
			return
		case <-ticker.C:
			frame++
			l.render(frame)
		}
	}
}

func (l *scanLoader) render(frame int) {
	arc := scanLoaderFrames[frame%len(scanLoaderFrames)]
	if l.color {
		arc = scanLoaderColors[frame%len(scanLoaderColors)] + arc + resetStyle
	}
	_, _ = fmt.Fprintf(l.writer, "%s%s beacon scanning the horizon…", clearLine, arc)
}

func (l *scanLoader) readyText() string {
	if !l.color {
		return "✓ beacon ready"
	}
	return "\x1b[92m✓\x1b[0m beacon ready"
}
