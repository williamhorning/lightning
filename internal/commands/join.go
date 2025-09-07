package commands

import (
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func bridgeJoin(database data.Database, name string) *lightning.Command {
	cmd := &lightning.Command{
		Name:        name,
		Description: "join an existing bridge with the given ID",
		Arguments:   []*lightning.CommandArgument{{Name: "id", Description: "the bridge id to use", Required: true}},
		Executor: func(opts *lightning.CommandOptions) {
			if opts.Arguments["id"] == "" {
				sendErr(missingArgumentError{argument: "id"}, "missing argument", opts)

				return
			}

			bridge, err := database.GetBridge(opts.Arguments["id"])
			if err != nil {
				sendErr(err, "failed to get bridge by id", opts)

				return
			} else if bridge.ID == "" {
				sendErr(bridgeNotFoundError{}, "no bridge with that ID exists", opts)

				return
			}

			channel, err := prepareChannelForBridge(database, opts)
			if err != nil {
				sendErr(err, "failed to setup channel for bridge", opts)

				return
			}

			channel.Disabled = lightning.ChannelDisabled{Read: name == "subscribe", Write: false}
			bridge.Channels = append(bridge.Channels, *channel)

			if err = database.CreateBridge(bridge); err != nil {
				sendErr(err, "failed to update bridge in the database", opts)

				return
			}

			if err = opts.Reply(
				getMessage("joined bridge!", "you successfully joined the bridge `"+bridge.ID+"`!"), true,
			); err != nil {
				sendErr(err, "failed to respond to command", opts)
			}
		},
	}

	if name == "subscribe" {
		cmd.Description = "subscribe to an existing bridge with the given ID (read-only)"
	}

	return cmd
}
