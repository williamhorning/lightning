// Package lightning provides a framework for creating a cross-platform chatbot
package lightning

import (
	"sync"
	"sync/atomic"
)

// VERSION is the version of the lightning bot framework.
const VERSION = "0.8.0-beta.3"

// BotOptions allows you to configure the default author used by commands
// and the prefix used by the bot for registered commands, in addition to
// any platform specifics (like slash commands).
type BotOptions struct {
	Author MessageAuthor
	Prefix string
}

// Bot represents the collection of commands, plugins, and events that are
// used to make a bot using Lightning.
type Bot struct {
	commands map[string]Command

	plugins map[string]Plugin

	types map[string]PluginConstructor

	messageChannel chan Message
	editChannel    chan EditedMessage
	delChannel     chan BaseMessage
	commandChannel chan CommandEvent

	author          MessageAuthor
	messageHandlers []func(*Bot, *Message)
	editHandlers    []func(*Bot, *EditedMessage)
	delHandlers     []func(*Bot, *BaseMessage)
	commandHandlers []func(*Bot, *CommandEvent)

	pluginMutex sync.RWMutex

	typesMutex sync.RWMutex

	handlersMutex sync.RWMutex

	messageProcessorActive atomic.Bool
	editProcessorActive    atomic.Bool
	delProcessorActive     atomic.Bool
	commandProcessorActive atomic.Bool
}

// NewBot creates a new *Bot based on the [BotOptions] provided to it.
func NewBot(opts BotOptions) *Bot {
	bot := &Bot{
		author: opts.Author,

		commands: make(map[string]Command),

		plugins:     make(map[string]Plugin),
		pluginMutex: sync.RWMutex{},

		types:      make(map[string]PluginConstructor),
		typesMutex: sync.RWMutex{},

		messageChannel: make(chan Message, 1000),
		editChannel:    make(chan EditedMessage, 1000),
		delChannel:     make(chan BaseMessage, 1000),
		commandChannel: make(chan CommandEvent, 1000),

		messageHandlers: make([]func(*Bot, *Message), 0),
		editHandlers:    make([]func(*Bot, *EditedMessage), 0),
		delHandlers:     make([]func(*Bot, *BaseMessage), 0),
		commandHandlers: make([]func(*Bot, *CommandEvent), 0),
		handlersMutex:   sync.RWMutex{},

		messageProcessorActive: atomic.Bool{},
		editProcessorActive:    atomic.Bool{},
		delProcessorActive:     atomic.Bool{},
		commandProcessorActive: atomic.Bool{},
	}

	bot.AddHandler(handleCommandEvent)
	bot.AddHandler(handleMessageCommand(opts.Prefix))

	return bot
}
