package commands

import (
	"slices"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func bridgeLeave(database data.Database) *lightning.Command {
	return &lightning.Command{
		Name:        "leave",
		Description: "leave the bridge that this channel is part of",
		Arguments:   []*lightning.CommandArgument{{Name: "id", Description: "the bridge id to use", Required: true}},
		Executor: func(opts *lightning.CommandOptions) {
			if opts.Arguments["id"] == "" {
				sendErr(missingArgumentError{argument: "id"}, "missing argument", opts)

				return
			}

			bridge, err := database.GetBridgeByChannel(opts.ChannelID)
			if err != nil {
				sendErr(err, "failed to get bridge from channel", opts)

				return
			} else if bridge.ID == "" {
				sendErr(bridgeNotFoundError{}, "this channel is not part of a bridge", opts)

				return
			}

			if opts.Arguments["id"] != bridge.ID {
				sendErr(bridgeNotFoundError{}, "this channel is not part of a bridge with that ID", opts)

				return
			}

			for idx, channel := range bridge.Channels {
				if channel.ID == opts.ChannelID {
					bridge.Channels = slices.Delete(bridge.Channels, idx, idx+1)

					break
				}
			}

			if err = database.CreateBridge(bridge); err != nil {
				sendErr(err, "failed to update database", opts)

				return
			}

			if err = opts.Reply(
				getMessage("left bridge!", "you successfully left the bridge `"+bridge.ID+"`!"), true,
			); err != nil {
				sendErr(err, "failed to respond to command", opts)
			}
		},
	}
}
