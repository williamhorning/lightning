package lightning

import "strings"

// SetupChannel allows you to create the platform-specific equivalent of
// a webhook and allows you to send messages with a different author, when
// then return value is passed as ChannelData in [*SendOptions].
func (b *Bot) SetupChannel(userID, channelID string) (map[string]string, error) {
	plugin, channel, err := b.getPluginFromChannel(channelID)
	if err != nil {
		return nil, err
	}

	result, err := plugin.SetupChannel(userID, channel)
	if err == nil {
		return result, nil
	}

	return nil, &PluginMethodError{channelID, "SetupChannel", err}
}

// SendMessage allows you to send a message to the channel and plugin specified
// on the provided [Message]. You may additionally provide [*SendOptions]. It
// returns the IDs of the messages sent, which may be nil if an error occurs.
func (b *Bot) SendMessage(message *Message, opts *SendOptions) ([]string, error) {
	plugin, channel, err := b.getPluginFromChannel(message.ChannelID)
	if err != nil {
		return nil, err
	}

	msg := *message
	msg.ChannelID = channel

	result, err := plugin.SendMessage(&msg, opts)
	if err == nil {
		return result, nil
	}

	return nil, &PluginMethodError{message.ChannelID, "SendMessage", err}
}

// EditMessage allows you to edit a message in the channel and plugin specified.
// The 'ids' parameter should contain the IDs of the messages to be edited, as
// returned by SendMessage. It will return the message IDs left after editing,
// which may differ from the original message IDs in content and length.
func (b *Bot) EditMessage(message *Message, ids []string, opts *SendOptions) ([]string, error) {
	plugin, channel, err := b.getPluginFromChannel(message.ChannelID)
	if err != nil {
		return nil, err
	}

	msg := *message
	msg.ChannelID = channel

	result, err := plugin.EditMessage(&msg, ids, opts)
	if err != nil {
		return result, &PluginMethodError{message.ChannelID, "EditMessage", err}
	}

	return result, nil
}

// DeleteMessages allows you to delete messages in the channel and plugin specified.
// The 'ids' parameter should contain the IDs of the messages to be edited, as
// returned by SendMessage.
func (b *Bot) DeleteMessages(channelID string, ids []string) error {
	plugin, channel, err := b.getPluginFromChannel(channelID)
	if err != nil {
		return err
	}

	err = plugin.DeleteMessage(channel, ids)
	if err != nil {
		return &PluginMethodError{channelID, "DeleteMessages", err}
	}

	return nil
}

func (b *Bot) getPluginFromChannel(channel string) (Plugin, string, error) {
	pluginName, channelName, _ := strings.Cut(channel, "::")

	b.mutex.RLock()
	plugin, found := b.plugins[pluginName]
	b.mutex.RUnlock()

	if !found {
		return nil, "", MissingPluginInstanceError{pluginName}
	}

	return plugin, channelName, nil
}
