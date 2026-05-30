package bot

import (
	"sync"

	"github.com/phibot/phibot/internal/logger"
)

type EventHandler func(Event)

type EventBus struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("event handler panic", "event", event.Type, "panic", r)
				}
			}()
			h(event)
		}(handler)
	}
}
