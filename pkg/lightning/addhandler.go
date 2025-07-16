package lightning

import (
	"log/slog"
	"sync"
	"sync/atomic"
)

// AddHandler allows you to register a listener for a given event type.
// Each handler must take in a *Bot and a pointer to a struct that corresponds
// with the event you want to listen to.
func (b *Bot) AddHandler(listener any) {
	b.handlersMutex.Lock()
	defer b.handlersMutex.Unlock()

	switch listener := listener.(type) {
	case func(*Bot, *EditedMessage):
		b.editHandlers = append(b.editHandlers, listener)
	case func(*Bot, *Message):
		b.messageHandlers = append(b.messageHandlers, listener)
	case func(*Bot, *BaseMessage):
		b.delHandlers = append(b.delHandlers, listener)
	case func(*Bot, *CommandEvent):
		b.commandHandlers = append(b.commandHandlers, listener)
	default:
		slog.Warn("invalid listener registered, this won't ever be called", "listener", listener)
	}

	ensureHandlers(b)
}

func ensureHandlers(bot *Bot) {
	if !bot.editProcessorActive.Swap(true) {
		go processEventHandlers(bot.editChannel, &bot.editHandlers, &bot.handlersMutex, &bot.editProcessorActive, bot)
	}

	if !bot.messageProcessorActive.Swap(true) {
		go processEventHandlers(bot.messageChannel, &bot.messageHandlers, &bot.handlersMutex,
			&bot.messageProcessorActive, bot)
	}

	if !bot.delProcessorActive.Swap(true) {
		go processEventHandlers(bot.delChannel, &bot.delHandlers, &bot.handlersMutex, &bot.delProcessorActive, bot)
	}

	if !bot.commandProcessorActive.Swap(true) {
		go processEventHandlers(bot.commandChannel, &bot.commandHandlers, &bot.handlersMutex,
			&bot.commandProcessorActive, bot)
	}
}

func processEventHandlers[C any](
	incoming chan C,
	handlersPtr *[]func(*Bot, *C),
	mutex *sync.RWMutex,
	store *atomic.Bool,
	lightning *Bot,
) {
	for msg := range incoming {
		mutex.RLock()

		handlersCopy := make([]func(*Bot, *C), len(*handlersPtr))
		copy(handlersCopy, *handlersPtr)
		mutex.RUnlock()

		for _, handler := range handlersCopy {
			localMsg := msg

			go func(handle func(*Bot, *C)) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("lightning: handler panic", "error", r)
					}
				}()

				handle(lightning, &localMsg)
			}(handler)
		}
	}

	store.Store(false)
}
