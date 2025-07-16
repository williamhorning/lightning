package lightning

// SetupChannel allows you to create the platform-specific equivalent of
// a webhook and allows you to send messages with a different author, when
// then return value is passed as ChannelData in [*SendOptions].
func (b *Bot) SetupChannel(pluginName, channel string) (any, error) {
	plugin, ok := b.plugins[pluginName]
	if !ok {
		return nil, MissingPluginError{}
	}

	result, err := plugin.SetupChannel(channel)
	if err == nil {
		return result, nil
	}

	return nil, LogError(err, "failed to setup channel", map[string]any{"channel": channel}, nil)
}

// SendMessage allows you to send a message to the channel and plugin specified
// on the provided [Message]. You may additionally provide [*SendOptions].
func (b *Bot) SendMessage(message Message, opts *SendOptions) ([]string, error) {
	plugin, ok := b.plugins[message.Plugin]
	if !ok {
		return nil, MissingPluginError{}
	}

	result, err := plugin.SendMessage(message, opts)
	if err == nil {
		return result, nil
	}

	return nil, LogError(err, "failed to send message", map[string]any{"message": message, "opts": opts}, nil)
}

// EditMessage allows you to edit a message in the channel and plugin specified.
func (b *Bot) EditMessage(message Message, ids []string, opts *SendOptions) error {
	plugin, ok := b.plugins[message.Plugin]
	if !ok {
		return MissingPluginError{}
	}

	if err := plugin.EditMessage(message, ids, opts); err != nil {
		return LogError(err, "failed to edit message", map[string]any{"message": message, "opts": opts}, nil)
	}

	return nil
}

// DeleteMessages allows you to delete messages in the channel and plugin specified.
func (b *Bot) DeleteMessages(pluginName, channel string, ids []string) error {
	plugin, ok := b.plugins[pluginName]
	if !ok {
		return MissingPluginError{}
	}

	if err := plugin.DeleteMessage(channel, ids); err != nil {
		return LogError(err, "failed to delete message", map[string]any{"channel": channel, "ids": ids}, nil)
	}

	return nil
}
