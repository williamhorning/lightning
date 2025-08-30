package guilded

import (
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type guildedWebhookDataError struct{}

func (guildedWebhookDataError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (guildedWebhookDataError) Error() string {
	return "invalid webhook data for Guilded channel"
}

type guildedStatusError struct {
	msg          string
	data         string
	code         int
	disableWrite bool
}

func (e guildedStatusError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.disableWrite}
}

func (e guildedStatusError) Error() string {
	return strconv.Itoa(e.code) + ":" + e.msg + "\n\tdata: " + e.data
}

type guildedWebhookTokenNilError struct {
	channel string
}

func (e guildedWebhookTokenNilError) Error() string {
	return "guilded: " + e.channel + " has a nil webhook token, probably due to a Guilded bug"
}
