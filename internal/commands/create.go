package commands

import (
	"log/slog"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func bridgeCreate(database data.Database) *lightning.Command {
	return &lightning.Command{
		Name:        "create",
		Description: "create a new bridge in this channel",
		Executor: func(opts *lightning.CommandOptions) {
			channel, err := prepareChannelForBridge(database, opts)
			if err != nil {
				sendErr(err, "failed to setup channel", opts)

				return
			}

			bridge := data.Bridge{
				ID:       ulid.Make().String(),
				Channels: []data.BridgeChannel{*channel},
				Settings: data.BridgeSettings{},
			}

			if err = database.CreateBridge(bridge); err != nil {
				sendErr(err, "failed to save to database", opts)

				return
			}

			if err = opts.Reply(getMessage("created bridge!",
				"you can now join the bridge you made in other channels by using ||`"+opts.Prefix+"bridge join "+
					bridge.ID+"`||. Keep that command secret!"), true); err != nil {
				sendErr(err, "failed to respond to create command", opts)

				bridge.Channels = []data.BridgeChannel{}

				if err = database.CreateBridge(bridge); err != nil {
					slog.Error("failed to remove bridge after failed response", "err", err) // at least log the error
				}
			}
		},
	}
}
