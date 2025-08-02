package bridge

import (
	"github.com/williamhorning/lightning/pkg/lightning"
)

type eventType string

const (
	typeCreate eventType = "create"
	typeEdit   eventType = "edit"
	typeDelete eventType = "delete"
)

type bridgeSettings struct {
	AllowEveryone bool `json:"allow_everyone"`
}

type bridgeChannel struct {
	Data     any                       `json:"data,omitempty"`
	ID       string                    `json:"id"`
	Disabled lightning.ChannelDisabled `json:"disabled"`
}

type bridge struct {
	ID       string          `json:"id"`
	Channels []bridgeChannel `json:"channels"`
	Settings bridgeSettings  `json:"settings"`
}

type channelMessage struct {
	ChannelID  string   `json:"channel_id"`
	MessageIDs []string `json:"message_ids"`
}

type channelMessageArray []channelMessage

type bridgeMessageCollection struct {
	ID       string              `json:"id"`
	BridgeID string              `json:"bridge_id"`
	Messages channelMessageArray `json:"messages"`
}

func (m *bridgeMessageCollection) getChannelMessageIDs(channelID string) []string {
	if m == nil {
		return nil
	}

	for _, msg := range m.Messages {
		if msg.ChannelID == channelID {
			return msg.MessageIDs
		}
	}

	return nil
}

func (b *bridge) getChannelDisabled(channelID string) lightning.ChannelDisabled {
	for _, channel := range b.Channels {
		if channel.ID == channelID {
			return channel.Disabled
		}
	}

	return lightning.ChannelDisabled{Read: false, Write: false}
}
