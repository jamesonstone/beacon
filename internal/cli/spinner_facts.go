package cli

type factDeck struct {
	facts    []string
	order    []int
	position int
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
