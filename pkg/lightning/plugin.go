package lightning

// PluginConstructor makes a [Plugin] with the specified config.
type PluginConstructor func(config any) (Plugin, error)

// A Plugin provides methods used by [Bot] to allow bots to not worry
// about platform specifics, as each Plugin handles that.
type Plugin interface {
	Name() string
	SetupChannel(channel string) (any, error)
	SendMessage(message Message, opts *SendOptions) ([]string, error)
	EditMessage(message Message, ids []string, opts *SendOptions) error
	DeleteMessage(channel string, ids []string) error
	SetupCommands(command map[string]Command) error
	ListenMessages() <-chan Message
	ListenEdits() <-chan EditedMessage
	ListenDeletes() <-chan BaseMessage
	ListenCommands() <-chan CommandEvent
}

// AddPluginType takes in a [PluginConstructor] and registers it so you can later
// use it. It only returns an error if the plugin type is already registered.
func (b *Bot) AddPluginType(name string, constructor PluginConstructor) error {
	b.typesMutex.Lock()
	defer b.typesMutex.Unlock()

	if _, exists := b.types[name]; exists {
		return LogError(PluginRegisteredError{}, "Plugin type already registered", map[string]any{"name": name}, nil)
	}

	b.types[name] = constructor

	return nil
}

// UsePluginType takes in a plugin name and config to use a plugin with your bot.
// It only returns an error if a plugin already exists *or* if the plugin type is
// not found.
func (b *Bot) UsePluginType(name string, config any) error {
	b.typesMutex.RLock()
	defer b.typesMutex.RUnlock()

	b.pluginMutex.Lock()
	defer b.pluginMutex.Unlock()

	if _, exists := b.plugins[name]; exists {
		return PluginRegisteredError{}
	}

	constructor, exists := b.types[name]
	if !exists {
		return MissingPluginError{}
	}

	instance, err := constructor(config)
	if err != nil {
		return err
	}

	b.plugins[instance.Name()] = instance

	ensureHandlers(b)

	go func() {
		msgChan := instance.ListenMessages()
		for msg := range msgChan {
			b.messageChannel <- msg
		}
	}()

	go func() {
		editChan := instance.ListenEdits()
		for msg := range editChan {
			b.editChannel <- msg
		}
	}()

	go func() {
		delChan := instance.ListenDeletes()
		for msg := range delChan {
			b.delChannel <- msg
		}
	}()

	go func() {
		cmdChan := instance.ListenCommands()
		for cmd := range cmdChan {
			b.commandChannel <- cmd
		}
	}()

	return nil
}
