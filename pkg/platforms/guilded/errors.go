package guilded

import (
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type guildedStatusError struct {
	code int
}

func (e guildedStatusError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.code >= 400 && e.code < 500}
}

func (e guildedStatusError) Error() string {
	return "failed to send Guilded message: " + strconv.FormatInt(int64(e.code), 10)
}

type guildedShuttingDownError struct{}

func (*guildedShuttingDownError) Error() string {
	return "Guilded is shutting down on December 19th, so you'll no longer able to setup new channels with Guilded." +
		"Please look at moving your server elsewhere. See https://www.guilded.gg/blog/guilded-shut-down-12-19-25"
}
