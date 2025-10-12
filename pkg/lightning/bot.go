// Package lightning provides a framework for creating a cross-platform chatbot
package lightning

import (
	"sync"
	"sync/atomic"
)

// VERSION is the version of the lightning bot framework.
const VERSION = "0.8.0-rc.5"

// BotOptions allows you to configure the prefix used by the bot for registered
// commands, in addition to any platform specifics (like slash commands). If a
// zero value is provided for the Prefix, it will default to "!".
type BotOptions struct {
	Prefix string
}

// Bot represents the collection of commands, plugins, and events that are
// used to make a bot using Lightning.
type Bot struct {
	messageHandlers atomic.Pointer[[]func(*Bot, *Message)]
	editHandlers    atomic.Pointer[[]func(*Bot, *EditedMessage)]
	delHandlers     atomic.Pointer[[]func(*Bot, *BaseMessage)]
	commandHandlers atomic.Pointer[[]func(*Bot, *CommandEvent)]

	messageChannel chan *Message
	editChannel    chan *EditedMessage
	delChannel     chan *BaseMessage
	commandChannel chan *CommandEvent

	commands map[string]*Command
	plugins  map[string]Plugin
	types    map[string]PluginConstructor

	prefix string

	pluginMutex sync.RWMutex
	typesMutex  sync.RWMutex

	messageProcessorActive atomic.Bool
	editProcessorActive    atomic.Bool
	delProcessorActive     atomic.Bool
	commandProcessorActive atomic.Bool
}

// NewBot creates a new *Bot based on the [BotOptions] provided to it.
func NewBot(opts BotOptions) *Bot {
	if opts.Prefix == "" {
		opts.Prefix = "!"
	}

	bot := &Bot{
		prefix: opts.Prefix,

		commands: make(map[string]*Command),
		plugins:  make(map[string]Plugin),
		types:    make(map[string]PluginConstructor),

		messageChannel: make(chan *Message, 1000),
		editChannel:    make(chan *EditedMessage, 1000),
		delChannel:     make(chan *BaseMessage, 1000),
		commandChannel: make(chan *CommandEvent, 1000),
	}

	bot.messageHandlers.Store(&[]func(*Bot, *Message){})
	bot.editHandlers.Store(&[]func(*Bot, *EditedMessage){})
	bot.delHandlers.Store(&[]func(*Bot, *BaseMessage){})
	bot.commandHandlers.Store(&[]func(*Bot, *CommandEvent){})

	bot.AddHandler(handleCommandEvent)
	bot.AddHandler(handleMessageCommand)

	return bot
}
