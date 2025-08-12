package guilded

import (
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type guildedWebhookDataError struct{}

// Disable implements the lightning.ChannelDisabler interface for guildedWebhookDataError.
func (guildedWebhookDataError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (guildedWebhookDataError) Error() string {
	return "invalid webhook data for Guilded channel"
}

type guildedStatusError struct {
	msg          string
	code         int
	disableWrite bool
}

// Disable implements the lightning.ChannelDisabler interface for guildedStatusError.
func (e guildedStatusError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.disableWrite}
}

func (e guildedStatusError) Error() string {
	return strconv.Itoa(e.code) + ": " + e.msg
}

type guildedWebhookTokenNilError struct {
	channel string
}

func (e guildedWebhookTokenNilError) Error() string {
	return "guilded: " + e.channel + " has a nil webhook token, probably due to a Guilded bug"
}
