package lightning

type BridgeSettings struct {
	AllowEveryone bool `json:"allow_everyone"`
}

type ReadWriteDisabled struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

type BridgeChannel struct {
	ID       string `json:"id"`
	Data     any    `json:"data"`
	Disabled any    `json:"disabled"`
	Plugin   string `json:"plugin"`
}

type BridgeMessageOptions struct {
	Channel  BridgeChannel  `json:"channel"`
	Settings BridgeSettings `json:"settings"`
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

func (b *BridgeChannel) IsDisabled() ReadWriteDisabled {
	switch v := b.Disabled.(type) {
	case bool:
		return ReadWriteDisabled{v, v}
	case map[string]any:
		read, okRead := v["read"].(bool)
		write, okWrite := v["write"].(bool)
		if okRead && okWrite {
			return ReadWriteDisabled{read, write}
		} else if okRead {
			return ReadWriteDisabled{read, false}
		} else if okWrite {
			return ReadWriteDisabled{false, write}
		} else {
			return ReadWriteDisabled{false, false}
		}
	case ReadWriteDisabled:
		return v
	}
	return ReadWriteDisabled{false, false}
}
