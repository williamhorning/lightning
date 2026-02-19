package lightning

import (
	"log"
	"sync"
)

// AddHandler allows you to register a listener for a given event type.
// Each handler must take in a *Bot and a pointer to a struct that corresponds
// with the event you want to listen to. If you provide a listener which does
// not match a known event signature, it will be ignored.
func (b *Bot) AddHandler(listener any) {
	switch listener := listener.(type) {
	case func(*Bot, *EditedMessage):
		b.editEvents.add(listener)
	case func(*Bot, *Message):
		b.messageEvents.add(listener)
	case func(*Bot, *BaseMessage):
		b.deleteEvents.add(listener)
	case func(*Bot, *CommandEvent):
		b.commandEvents.add(listener)
	default:
		log.Printf("lightning: can't add unknown listener type: %T\n", listener)
	}
}

type handler[T any] struct {
	mu       sync.Mutex
	handlers []func(*Bot, T)
}

func (h *handler[T]) add(fn func(*Bot, T)) {
	h.mu.Lock()
	h.handlers = append(h.handlers, fn)
	h.mu.Unlock()
}

func (h *handler[T]) dispatch(bot *Bot, evt T) {
	handlers := h.handlers
	for _, fn := range handlers {
		go fn(bot, evt)
	}
}
