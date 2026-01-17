package lightning

// PluginConstructor makes a [Plugin] with the specified config.
type PluginConstructor func(config map[string]string) (Plugin, error)

// A Plugin provides methods used by [Bot] to allow bots to not worry
// about platform specifics, as each Plugin handles that.
type Plugin interface {
	SetupChannel(channel string) (map[string]string, error)
	SendMessage(message *Message, opts *SendOptions) ([]string, error)
	EditMessage(message *Message, ids []string, opts *SendOptions) ([]string, error)
	DeleteMessage(channel string, ids []string) error
	SetupCommands(command map[string]Command)
	ListenMessages() <-chan *Message
	ListenEdits() <-chan *EditedMessage
	ListenDeletes() <-chan *BaseMessage
	ListenCommands() <-chan *CommandEvent
}

// AddPluginType takes in a [PluginConstructor] and registers it so you can later
// use it. It overwrites existing plugin types if the name is a duplicate.
func (b *Bot) AddPluginType(name string, constructor PluginConstructor) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.types[name] = constructor
}

// UsePluginType takes in a plugin name and config to use a plugin with your bot.
// It only returns an error if a plugin already exists *or* if the plugin type is
// not found. If you pass an empty string to instanceName, it will default to
// typeName, but that value must be unique.
func (b *Bot) UsePluginType(typeName, instanceName string, config map[string]string) error {
	if instanceName == "" {
		instanceName = typeName
	}

	b.mutex.RLock()
	_, exists := b.plugins[instanceName]
	b.mutex.RUnlock()

	if exists {
		return PluginRegisteredError{instanceName}
	}

	b.mutex.RLock()
	constructor, ok := b.types[typeName]
	b.mutex.RUnlock()

	if !ok {
		return MissingPluginTypeError{typeName}
	}

	instance, err := constructor(config)
	if err != nil {
		return PluginMethodError{instanceName, "constructor", err}
	}

	b.mutex.Lock()
	b.plugins[instanceName] = instance
	b.mutex.Unlock()

	startPluginListeners(b, instanceName, &b.messageEvents, instance.ListenMessages())
	startPluginListeners(b, instanceName, &b.editEvents, instance.ListenEdits())
	startPluginListeners(b, instanceName, &b.deleteEvents, instance.ListenDeletes())
	startPluginListeners(b, instanceName, &b.commandEvents, instance.ListenCommands())

	return nil
}

func startPluginListeners[evt interface{ setChannelID(name string) }](
	b *Bot, name string, handler *handler[evt], events <-chan evt,
) {
	go func() {
		for evt := range events {
			evt.setChannelID(name)
			handler.dispatch(b, evt)
		}
	}()
}
