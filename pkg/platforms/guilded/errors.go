package guilded

import (
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

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
	return strconv.FormatInt(int64(e.code), 10) + ": " + e.msg + "\n\tdata: " + e.data
}

type guildedShuttingDownError struct{}

func (*guildedShuttingDownError) Error() string {
	return "Guilded is shutting down on December 19th, so you'll no longer able to setup new channels with Guilded." +
		"Please look at moving your server elsewhere. See https://www.guilded.gg/blog/guilded-shut-down-12-19-25"
}
