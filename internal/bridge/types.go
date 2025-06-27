package bridge

import "github.com/williamhorning/lightning/pkg/lightning"

type BridgeSettings struct {
	AllowEveryone bool `json:"allow_everyone"`
}

type BridgeChannel struct {
	ID       string `json:"id"`
	Data     any    `json:"data"`
	Disabled any    `json:"disabled"`
	Plugin   string `json:"plugin"`
}

type BridgeMessage struct {
	ID      []string `json:"id"`
	Channel string   `json:"channel"`
	Plugin  string   `json:"plugin"`
}

type Bridge struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Channels []BridgeChannel `json:"channels"`
	Settings BridgeSettings  `json:"settings"`
}

type BridgeMessageCollection struct {
	Bridge
	BridgeID string          `json:"bridge_id"`
	Messages []BridgeMessage `json:"messages"`
}

func (b *BridgeChannel) IsDisabled() lightning.ChannelDisabled {
	switch v := b.Disabled.(type) {
	case bool:
		return lightning.ChannelDisabled{Read: v, Write: v}
	case map[string]any:
		read, okRead := v["read"].(bool)
		write, okWrite := v["write"].(bool)
		if okRead && okWrite {
			return lightning.ChannelDisabled{Read: read, Write: write}
		} else if okRead {
			return lightning.ChannelDisabled{Read: read, Write: false}
		} else if okWrite {
			return lightning.ChannelDisabled{Read: false, Write: write}
		} else {
			return lightning.ChannelDisabled{Read: false, Write: false}
		}
	case lightning.ChannelDisabled:
		return v
	}
	return lightning.ChannelDisabled{Read: false, Write: false}
}
