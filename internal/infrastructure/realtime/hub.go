package realtime

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type subscriber struct {
	ch chan Event
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[*subscriber]struct{}
}

func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*subscriber]struct{}),
	}
}

func (h *Hub) Subscribe(householdID string) (<-chan Event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub := &subscriber{ch: make(chan Event, 16)}
	if h.subscribers[householdID] == nil {
		h.subscribers[householdID] = make(map[*subscriber]struct{})
	}
	h.subscribers[householdID][sub] = struct{}{}

	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if subs, ok := h.subscribers[householdID]; ok {
			delete(subs, sub)
			close(sub.ch)
			if len(subs) == 0 {
				delete(h.subscribers, householdID)
			}
		}
	}

	return sub.ch, unsubscribe
}

func (h *Hub) Publish(householdID, eventType string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	event := Event{Type: eventType, Payload: json.RawMessage(payload)}
	for sub := range h.subscribers[householdID] {
		select {
		case sub.ch <- event:
		default:
		}
	}
}
