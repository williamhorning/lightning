package lightning

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrPluginNotFound      = errors.New("plugin not found internally: this is a bug or misconfiguration")
	ErrPluginConfigInvalid = errors.New("plugin config is invalid")

	pluginConstructors = make(map[string]PluginConstructor)
	pluginRegistry     = make(map[string]Plugin)
	handledEvents      = make(map[string]struct{})

	constructorsLock   sync.RWMutex
	pluginRegistryLock sync.RWMutex

	messages []chan Message
	edits    []chan Message
	deletes  []chan BaseMessage
	commands []chan CommandEvent
	mutex    sync.RWMutex
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

func RegisterPluginType(name string, constructor PluginConstructor) {
	constructorsLock.Lock()
	defer constructorsLock.Unlock()

	Log.Debug().Str("plugin", name).Msg("Registering plugin type")

	if _, exists := pluginConstructors[name]; exists {
		Log.Panic().Str("plugin", name).Msg("Plugin type already registered")
	}

	pluginConstructors[name] = constructor
}

func GetPlugin(name string) (Plugin, bool) {
	pluginRegistryLock.RLock()
	defer pluginRegistryLock.RUnlock()
	plugin, exists := pluginRegistry[name]
	return plugin, exists
}

func distributeEvents[T any](ev string, plugin Plugin, source <-chan T, destinations *[]chan T) {
	for event := range source {
		key := getEventKey(event) + "-" + ev

		time.Sleep(150 * time.Millisecond)

		if _, exists := handledEvents[key]; exists {
			Log.Trace().Str("plugin", plugin.Name()).Str("event", key).Msg("Event already handled, skipping")
			continue
		}

		mutex.RLock()
		for _, ch := range *destinations {
			select {
			case ch <- event:
				Log.Trace().Str("plugin", plugin.Name()).Str("event", key).Msg("Event distributed")
			default:
				Log.Warn().Str("plugin", plugin.Name()).Msg("Skipped event - channel full or closed")
			}
		}
		mutex.RUnlock()
	}
}

func getEventKey(event any) string {
	switch e := event.(type) {
	case Message:
		return e.Plugin + "-" + e.EventID
	case BaseMessage:
		return e.Plugin + "-" + e.EventID
	case CommandEvent:
		return e.Plugin + "-" + e.EventID
	default:
		return "-"
	}
}

func registerPlugin(plugin string, config any) {
	pluginRegistryLock.Lock()
	defer pluginRegistryLock.Unlock()

	Log.Debug().Str("plugin", plugin).Msg("Registering plugin")

	if _, exists := pluginRegistry[plugin]; exists {
		Log.Panic().Str("plugin", plugin).Msg("Plugin already registered")
	}

	constructorsLock.RLock()
	constructor, exists := pluginConstructors[plugin]
	constructorsLock.RUnlock()

	if !exists {
		Log.Panic().Str("plugin", plugin).Msg("Plugin type not found")
	}

	instance, err := constructor(config)
	if err != nil {
		Log.Panic().Str("plugin", plugin).Err(err).Msg("Failed to setup plugin")
	}

	commands_list := make([]Command, 0, len(commandRegistry))
	for _, cmd := range commandRegistry {
		commands_list = append(commands_list, cmd)
	}

	if err := instance.SetupCommands(commands_list); err != nil {
		Log.Warn().Str("plugin", plugin).Err(err).Msg("Failed to setup commands for plugin")
	}

	pluginRegistry[plugin] = instance
	go distributeEvents("create", instance, instance.ListenMessages(), &messages)
	go distributeEvents("edit", instance, instance.ListenEdits(), &edits)
	go distributeEvents("delete", instance, instance.ListenDeletes(), &deletes)
	go distributeEvents("command", instance, instance.ListenCommands(), &commands)

	Log.Debug().Str("plugin", plugin).Msg("Plugin registered and listening!")
}

func setHandled(plugin string, event string, ev string) {
	Log.Trace().Str("plugin", plugin).Str("event", event).Str("ev", ev).Msg("Setting handled event")
	handledEvents[plugin+"-"+event+"-"+ev] = struct{}{}
}

func createEventChannel[T any](bufferSize int, channelList *[]chan T) <-chan T {
	ch := make(chan T, bufferSize)
	mutex.Lock()
	*channelList = append(*channelList, ch)
	mutex.Unlock()
	return ch
}

func ListenMessages() <-chan Message {
	Log.Trace().Msg("Creating message event channel")
	return createEventChannel(100, &messages)
}

func ListenEdits() <-chan Message {
	Log.Trace().Msg("Creating edit event channel")
	return createEventChannel(100, &edits)
}

func ListenDeletes() <-chan BaseMessage {
	Log.Trace().Msg("Creating delete event channel")
	return createEventChannel(100, &deletes)
}

func ListenCommands() <-chan CommandEvent {
	Log.Trace().Msg("Creating command event channel")
	return createEventChannel(100, &commands)
}
