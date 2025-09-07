package commands

import (
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func bridgeToggle(database data.Database) *lightning.Command {
	return &lightning.Command{
		Name:        "toggle",
		Description: "toggle a setting for the bridge that this channel is part of",
		Executor: func(opts *lightning.CommandOptions) {
			if opts.Arguments["setting"] == "" {
				sendErr(missingArgumentError{argument: "setting"}, "missing argument", opts)

				return
			}

			bridge, err := database.GetBridgeByChannel(opts.ChannelID)
			if err != nil {
				sendErr(err, "failed to get bridge for channel", opts)

				return
			} else if bridge.ID == "" {
				sendErr(bridgeNotFoundError{}, "this channel is not part of a bridge", opts)

				return
			}

			if opts.Arguments["setting"] != "allow_everyone" {
				sendErr(missingArgumentError{argument: "setting"}, "invalid argument", opts)

				return
			}

			bridge.Settings.AllowEveryone = !bridge.Settings.AllowEveryone

			if err = database.CreateBridge(bridge); err != nil {
				sendErr(err, "failed to set settings", opts)

				return
			}

			if err = opts.Reply(getMessage("toggled setting!", getSettingsString(&bridge)), false); err != nil {
				sendErr(err, "failed to respond to command", opts)
			}
		},
	}
}
