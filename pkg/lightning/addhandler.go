package lightning

import "sync/atomic"

// AddHandler allows you to register a listener for a given event type.
// Each handler must take in a *Bot and a pointer to a struct that corresponds
// with the event you want to listen to.
func (b *Bot) AddHandler(listener any) {
	switch listener := listener.(type) {
	case func(*Bot, *EditedMessage):
		newHandlers := append(*b.editHandlers.Load(), listener)
		b.editHandlers.Store(&newHandlers)

		go processEventHandlers(b.editChannel, &b.editHandlers, &b.editProcessorActive, b)
	case func(*Bot, *Message):
		newHandlers := append(*b.messageHandlers.Load(), listener)
		b.messageHandlers.Store(&newHandlers)

		go processEventHandlers(b.messageChannel, &b.messageHandlers, &b.messageProcessorActive, b)
	case func(*Bot, *BaseMessage):
		newHandlers := append(*b.delHandlers.Load(), listener)
		b.delHandlers.Store(&newHandlers)

		go processEventHandlers(b.delChannel, &b.delHandlers, &b.delProcessorActive, b)
	case func(*Bot, *CommandEvent):
		newHandlers := append(*b.commandHandlers.Load(), listener)
		b.commandHandlers.Store(&newHandlers)

		go processEventHandlers(b.commandChannel, &b.commandHandlers, &b.commandProcessorActive, b)
	}
}

func processEventHandlers[C any](
	incoming <-chan C,
	handlers *atomic.Pointer[[]func(*Bot, C)],
	store *atomic.Bool,
	bot *Bot,
) {
	if store.Swap(true) {
		return
	}

	for msg := range incoming {
		for _, handler := range *handlers.Load() {
			localMsg := msg
			go handler(bot, localMsg)
		}
	}

	store.Store(false)
}
