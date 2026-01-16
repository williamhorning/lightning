// Package lightning provides a framework for creating a cross-platform chatbot
package lightning

import "sync"

// Bot represents the collection of commands, plugins, and events that are
// used to make a bot using Lightning.
type Bot struct {
	messageEvents handler[*Message]
	editEvents    handler[*EditedMessage]
	deleteEvents  handler[*BaseMessage]
	commandEvents handler[*CommandEvent]

	commands map[string]Command
	plugins  map[string]Plugin
	types    map[string]PluginConstructor
	mutex    sync.RWMutex

	prefix string
}

// NewBot creates a new *Bot based on the [BotOptions] provided to it.
func NewBot(prefix string) *Bot {
	bot := &Bot{
		commands: make(map[string]Command),
		plugins:  make(map[string]Plugin),
		types:    make(map[string]PluginConstructor),
		prefix:   prefix,
	}

	bot.commandEvents.add(handleCommandEvent)
	bot.messageEvents.add(handleTextCommand)

	return bot
}
