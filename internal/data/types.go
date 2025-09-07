package data

import "github.com/williamhorning/lightning/pkg/lightning"

// EventType is the event type used by the bridge.
type EventType string

// These event types are supported by the bridge.
const (
	TypeCreate EventType = "create"
	TypeEdit   EventType = "edit"
	TypeDelete EventType = "delete"
)

// BridgeSettings are used to configure the bridge.
type BridgeSettings struct {
	AllowEveryone bool `json:"allow_everyone"`
}

// BridgeChannel represents a channel in a bridge.
type BridgeChannel struct {
	Data     any                       `json:"data,omitempty"`
	ID       string                    `json:"id"`
	Disabled lightning.ChannelDisabled `json:"disabled"`
}

// Bridge represents a collection of channels to send and receive messages between.
type Bridge struct {
	ID       string          `json:"id"`
	Channels []BridgeChannel `json:"channels"`
	Settings BridgeSettings  `json:"settings"`
}

// ChannelMessage represents a collection of message IDs for a specific channel.
type ChannelMessage struct {
	ChannelID  string   `json:"channel_id"`
	MessageIDs []string `json:"message_ids"`
}

// BridgeMessageCollection represents a collection of messages for a specific bridge.
type BridgeMessageCollection struct {
	ID       string           `json:"id"`
	BridgeID string           `json:"bridge_id"`
	Messages []ChannelMessage `json:"messages"`
}

// GetChannelMessageIDs returns the message IDs for a specific channel in the bridge message collection.
func (m *BridgeMessageCollection) GetChannelMessageIDs(channelID string) []string {
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

// GetChannelDisabled returns the disabled status for a specific channel in the bridge.
func (b *Bridge) GetChannelDisabled(channelID string) lightning.ChannelDisabled {
	for _, channel := range b.Channels {
		if channel.ID == channelID {
			return channel.Disabled
		}
	}

	return lightning.ChannelDisabled{Read: false, Write: false}
}
