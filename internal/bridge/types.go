package bridge

import "github.com/williamhorning/lightning/pkg/lightning"

type bridgeSettings struct {
	AllowEveryone bool `json:"allow_everyone"`
}

type bridgeChannel struct {
	ID       string `json:"id"`
	Data     any    `json:"data"`
	Disabled any    `json:"disabled"`
	Plugin   string `json:"plugin"`
}

type bridgeMessage struct {
	Channel string   `json:"channel"`
	Plugin  string   `json:"plugin"`
	ID      []string `json:"id"`
}

type bridge struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Channels []bridgeChannel `json:"channels"`
	Settings bridgeSettings  `json:"settings"`
}

type bridgeMessageCollection struct {
	BridgeID string          `json:"bridge_id"`
	Messages []bridgeMessage `json:"messages"`

	bridge //nolint:embeddedstructfieldcheck // memory alignment is better this way
}

func (b *bridgeMessageCollection) getChannelMessageIDs(channelID, plugin string) []string {
	if b == nil {
		return nil
	}

	for _, message := range b.Messages {
		if message.Channel == channelID && message.Plugin == plugin {
			return message.ID
		}
	}

	return nil
}

func (b *bridgeChannel) isDisabled() lightning.ChannelDisabled {
	switch value := b.Disabled.(type) {
	case bool:
		return lightning.ChannelDisabled{Read: value, Write: value}
	case map[string]any:
		read, ok := value["read"].(bool)
		if !ok {
			read = false
		}

		write, ok := value["write"].(bool)
		if !ok {
			write = false
		}

		return lightning.ChannelDisabled{Read: read, Write: write}
	case lightning.ChannelDisabled:
		return value
	default:
		return lightning.ChannelDisabled{Read: false, Write: false}
	}
}

func (b *bridge) getChannelDisabled(channelID, plugin string) lightning.ChannelDisabled {
	for _, channel := range b.Channels {
		if channel.ID == channelID && channel.Plugin == plugin {
			return channel.isDisabled()
		}
	}

	return lightning.ChannelDisabled{Read: false, Write: false}
}

type eventType string

const (
	typeCreate eventType = "create"
	typeEdit   eventType = "edit"
	typeDelete eventType = "delete"
)
