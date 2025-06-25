package lightning

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrPluginNotFound      = errors.New("plugin not found internally: this is a bug or misconfiguration")
	ErrPluginConfigInvalid = errors.New("plugin config is invalid")
	Plugins                = &PluginRegistry{
		make(map[string]Plugin),
		sync.RWMutex{},
		make(map[string]PluginConstructor),
		sync.RWMutex{},
		make(map[string]struct{}),
		[]chan Message{},
		[]chan Message{},
		[]chan BaseMessage{},
		[]chan CommandEvent{},
		sync.RWMutex{},
		200 * time.Millisecond,
	}
)

type PluginConstructor func(config any) (Plugin, error)

type Plugin interface {
	Name() string
	SetupChannel(channel string) (any, error)
	SendMessage(message Message, opts *BridgeMessageOptions) ([]string, error)
	EditMessage(message Message, ids []string, opts *BridgeMessageOptions) error
	DeleteMessage(ids []string, opts *BridgeMessageOptions) error
	SetupCommands(command []Command) error
	ListenMessages() <-chan Message
	ListenEdits() <-chan Message
	ListenDeletes() <-chan BaseMessage
	ListenCommands() <-chan CommandEvent
}

type PluginRegistry struct {
	Plugins         map[string]Plugin
	pluginsLock     sync.RWMutex
	pluginTypes     map[string]PluginConstructor
	pluginTypesLock sync.RWMutex
	handledEvents   map[string]struct{}
	messages        []chan Message
	edits           []chan Message
	deletes         []chan BaseMessage
	commands        []chan CommandEvent
	eventMutex      sync.RWMutex
	eventDelay      time.Duration
}

func (pr *PluginRegistry) RegisterType(name string, constructor PluginConstructor) {
	pr.pluginTypesLock.Lock()
	defer pr.pluginTypesLock.Unlock()

	Log.Debug().Str("plugin", name).Msg("Registering plugin type")

	if _, exists := pr.pluginTypes[name]; exists {
		Log.Panic().Str("plugin", name).Msg("Plugin type already registered")
	}

	pr.pluginTypes[name] = constructor
}

func (pr *PluginRegistry) Get(name string) (Plugin, bool) {
	pr.pluginsLock.RLock()
	defer pr.pluginsLock.RUnlock()
	plugin, exists := pr.Plugins[name]
	return plugin, exists
}

func (pr *PluginRegistry) registerPlugin(name string, config any) error {
	pr.pluginTypesLock.RLock()
	pr.pluginsLock.Lock()
	defer pr.pluginTypesLock.RUnlock()
	defer pr.pluginsLock.Unlock()

	Log.Debug().Str("plugin", name).Msg("Registering plugin")

	if _, exists := pr.Plugins[name]; exists {
		Log.Panic().Str("plugin", name).Msg("Plugin already registered")
	}

	constructor, exists := pr.pluginTypes[name]
	if !exists {
		return ErrPluginNotFound
	}

	instance, err := constructor(config)
	if err != nil {
		return err
	}

	pr.Plugins[instance.Name()] = instance

	go distributeEvents(pr, "create", instance, instance.ListenMessages(), &pr.messages)
	go distributeEvents(pr, "edit", instance, instance.ListenEdits(), &pr.edits)
	go distributeEvents(pr, "delete", instance, instance.ListenDeletes(), &pr.deletes)
	go distributeEvents(pr, "command", instance, instance.ListenCommands(), &pr.commands)

	Log.Debug().Str("plugin", instance.Name()).Msg("Plugin registered and listening!")
	return nil
}

func (pr *PluginRegistry) setHandled(plugin string, event string, ev string) {
	pr.eventMutex.Lock()
	defer pr.eventMutex.Unlock()
	Log.Trace().Str("plugin", plugin).Str("event", event).Str("ev", ev).Msg("Setting handled event")
	pr.handledEvents[ev+"-"+plugin+"-"+event] = struct{}{}
}

func (pr *PluginRegistry) ListenMessages() <-chan Message {
	return createEventChannel(pr, 100, &pr.messages)
}

func (pr *PluginRegistry) ListenEdits() <-chan Message {
	return createEventChannel(pr, 100, &pr.edits)
}

func (pr *PluginRegistry) ListenDeletes() <-chan BaseMessage {
	return createEventChannel(pr, 100, &pr.deletes)
}

func (pr *PluginRegistry) ListenCommands() <-chan CommandEvent {
	return createEventChannel(pr, 100, &pr.commands)
}

func distributeEvents[T any](pr *PluginRegistry, ev string, plugin Plugin, source <-chan T, destinations *[]chan T) {
	for event := range source {
		key := ev + "-" + plugin.Name() + "-"

		switch v := any(event).(type) {
		case Message:
			key += v.EventID
		case BaseMessage:
			key += v.EventID
		case CommandEvent:
			key += v.EventID
		}

		time.Sleep(pr.eventDelay)

		if _, exists := pr.handledEvents[key]; exists {
			Log.Trace().Str("plugin", plugin.Name()).Str("event", key).Msg("Event already handled, skipping")
			continue
		}

		pr.setHandled(plugin.Name(), ev, key)

		pr.eventMutex.RLock()
		for _, ch := range *destinations {
			select {
			case ch <- event:
				Log.Trace().Str("plugin", plugin.Name()).Str("event", key).Msg("Event distributed")
			default:
				Log.Warn().Str("plugin", plugin.Name()).Msg("Skipped event - channel full or closed")
			}
		}
		pr.eventMutex.RUnlock()
	}
}

func createEventChannel[T any](pr *PluginRegistry, bufferSize int, channelList *[]chan T) <-chan T {
	Log.Trace().Msg("Creating event channel")
	ch := make(chan T, bufferSize)
	pr.eventMutex.Lock()
	defer pr.eventMutex.Unlock()
	*channelList = append(*channelList, ch)
	return ch
}
