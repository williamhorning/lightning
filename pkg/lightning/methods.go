package lightning

import "strings"

// SetupChannel allows you to create the platform-specific equivalent of
// a webhook and allows you to send messages with a different author, when
// then return value is passed as ChannelData in [*SendOptions].
func (b *Bot) SetupChannel(channelID string) (map[string]string, error) {
	plugin, channel, ok := b.getPluginFromChannel(channelID)
	if !ok {
		return nil, MissingPluginError{}
	}

	result, err := plugin.SetupChannel(channel)
	if err == nil {
		return result, nil
	}

	return nil, &PluginMethodError{channelID, "SetupChannel", "failed to setup channel", []error{err}}
}

// SendMessage allows you to send a message to the channel and plugin specified
// on the provided [Message]. You may additionally provide [*SendOptions]. It
// returns the IDs of the messages sent, which may be nil if an error occurs.
func (b *Bot) SendMessage(message *Message, opts *SendOptions) ([]string, error) {
	plugin, channel, ok := b.getPluginFromChannel(message.ChannelID)
	if !ok {
		return nil, MissingPluginError{}
	}

	oldID := message.ChannelID
	message.ChannelID = channel

	result, err := plugin.SendMessage(message, opts)
	if err == nil {
		return result, nil
	}

	return nil, &PluginMethodError{oldID, "SendMessage", "failed to send message", []error{err}}
}

// EditMessage allows you to edit a message in the channel and plugin specified.
// The 'ids' parameter should contain the IDs of the messages to be edited, as
// returned by SendMessage.
func (b *Bot) EditMessage(message *Message, ids []string, opts *SendOptions) error {
	plugin, channel, ok := b.getPluginFromChannel(message.ChannelID)
	if !ok {
		return MissingPluginError{}
	}

	oldID := message.ChannelID
	message.ChannelID = channel

	err := plugin.EditMessage(message, ids, opts)
	if err != nil {
		return &PluginMethodError{oldID, "EditMessage", "failed to edit message", []error{err}}
	}

	return nil
}

// DeleteMessages allows you to delete messages in the channel and plugin specified.
// The 'ids' parameter should contain the IDs of the messages to be edited, as
// returned by SendMessage.
func (b *Bot) DeleteMessages(channelID string, ids []string) error {
	plugin, channel, ok := b.getPluginFromChannel(channelID)
	if !ok {
		return MissingPluginError{}
	}

	err := plugin.DeleteMessage(channel, ids)
	if err != nil {
		return &PluginMethodError{channelID, "DeleteMessages", "failed to delete messages", []error{err}}
	}

	return nil
}

func (b *Bot) getPluginFromChannel(channel string) (Plugin, string, bool) {
	pluginName, channelName, found := strings.Cut(channel, "::")
	if !found {
		return nil, "", false
	}

	b.pluginMutex.RLock()
	plugin, found := b.plugins[pluginName]
	b.pluginMutex.RUnlock()

	return plugin, channelName, found
}
