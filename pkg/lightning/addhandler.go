package lightning

import "sync/atomic"

// AddHandler allows you to register a listener for a given event type.
// Each handler must take in a *Bot and a pointer to a struct that corresponds
// with the event you want to listen to.
func (b *Bot) AddHandler(listener any) {
	switch listener := listener.(type) {
	case func(*Bot, *EditedMessage):
		go processEventHandlers(&listener, b.editChannel, &b.editHandlers, &b.editProcessorActive, b)
	case func(*Bot, *Message):
		go processEventHandlers(&listener, b.messageChannel, &b.messageHandlers, &b.messageProcessorActive, b)
	case func(*Bot, *BaseMessage):
		go processEventHandlers(&listener, b.delChannel, &b.delHandlers, &b.delProcessorActive, b)
	case func(*Bot, *CommandEvent):
		go processEventHandlers(&listener, b.commandChannel, &b.commandHandlers, &b.commandProcessorActive, b)
	}
}

func processEventHandlers[C any](
	listener *func(*Bot, C),
	incoming <-chan C,
	handlers *atomic.Pointer[[]func(*Bot, C)],
	store *atomic.Bool,
	bot *Bot,
) {
	if listener != nil {
		newHandlers := append(*handlers.Load(), *listener)
		handlers.Store(&newHandlers)
	}

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
