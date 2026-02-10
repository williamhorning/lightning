package main

type bridgeChannel struct {
	ID            string            `db:"channel_id"`
	Data          map[string]string `db:"data"`
	DisabledRead  bool              `db:"disabled_read"`
	DisabledWrite bool              `db:"disabled_write"`
}

type bridge struct {
	ID            string
	AllowEveryone bool
	Channels      []bridgeChannel
}

type channelMessage struct {
	ChannelID  string   `db:"channel_id"`
	MessageIDs []string `db:"ids"`
}

type channelMessageSet []channelMessage

func (m channelMessageSet) getChannel(id string) []string {
	for _, msg := range m {
		if msg.ChannelID == id {
			return msg.MessageIDs
		}
	}

	return nil
}

func (b *bridge) getChannel(channelID string) bridgeChannel {
	for _, channel := range b.Channels {
		if channel.ID == channelID {
			return channel
		}
	}

	return bridgeChannel{}
}

type unsupportedDatabaseVersionError struct{}

func (unsupportedDatabaseVersionError) Error() string {
	return "database version must be d0.8.3 or higher. try running v0.8.6 to migrate your data."
}
