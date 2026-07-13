package agent

import "sync"

type eventHub struct {
	mutex       sync.Mutex
	next        int
	subscribers map[int]chan Event
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[int]chan Event)}
}

func (h *eventHub) Subscribe() (<-chan Event, func()) {
	h.mutex.Lock()
	id := h.next
	h.next++
	channel := make(chan Event, 64)
	h.subscribers[id] = channel
	h.mutex.Unlock()
	return channel, func() {
		h.mutex.Lock()
		if existing, ok := h.subscribers[id]; ok {
			delete(h.subscribers, id)
			close(existing)
		}
		h.mutex.Unlock()
	}
}

func (h *eventHub) Publish(event Event) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for _, channel := range h.subscribers {
		select {
		case channel <- event:
		default:
			// Agent work must never block on presentation. Snapshot-bearing and
			// terminal events replace the oldest buffered event so a slow client
			// can still recover current state and observe scan completion.
			if event.Snapshot == nil && event.Type != EventScanCompleted {
				continue
			}
			select {
			case <-channel:
			default:
			}
			select {
			case channel <- event:
			default:
			}
		}
	}
}
