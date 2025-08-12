package lightning

// PluginConstructor makes a [Plugin] with the specified config.
type PluginConstructor func(config any) (Plugin, error)

// A Plugin provides methods used by [Bot] to allow bots to not worry
// about platform specifics, as each Plugin handles that.
type Plugin interface {
	SetupChannel(channel string) (any, error)
	SendCommandResponse(message Message, opts *SendOptions, user string) ([]string, error)
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
		return PluginRegisteredError{}
	}

	b.types[name] = constructor

	return nil
}

// UsePluginType takes in a plugin name and config to use a plugin with your bot.
// It only returns an error if a plugin already exists *or* if the plugin type is
// not found.
func (b *Bot) UsePluginType(typeName, instanceName string, config any) error {
	if instanceName == "" {
		instanceName = typeName
	}

	if _, exists := b.plugins[instanceName]; exists {
		return PluginRegisteredError{}
	}

	b.typesMutex.RLock()

	constructor, ok := b.types[typeName]

	b.typesMutex.RUnlock()

	if !ok {
		return MissingPluginError{}
	}

	instance, err := constructor(config)
	if err != nil {
		return err
	}

	b.pluginMutex.Lock()

	b.plugins[instanceName] = instance

	b.pluginMutex.Unlock()

	ensureHandlers(b)

	b.startPluginListeners(instanceName, instance)

	return nil
}

// startPluginListeners listens for events from a plugin and forwards them.
// do NOT rely on the ChannelID format, treat it as an opaque string.
func (b *Bot) startPluginListeners(name string, instance Plugin) {
	go func() {
		for msg := range instance.ListenMessages() {
			msg.ChannelID = name + "::" + msg.ChannelID
			b.messageChannel <- msg
		}
	}()
	go func() {
		for edit := range instance.ListenEdits() {
			edit.Message.ChannelID = name + "::" + edit.Message.ChannelID
			b.editChannel <- edit
		}
	}()
	go func() {
		for del := range instance.ListenDeletes() {
			del.ChannelID = name + "::" + del.ChannelID
			b.delChannel <- del
		}
	}()
	go func() {
		for cmd := range instance.ListenCommands() {
			cmd.ChannelID = name + "::" + cmd.ChannelID
			b.commandChannel <- cmd
		}
	}()
}
