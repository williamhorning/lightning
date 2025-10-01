package commands

import (
	"strconv"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func bridgeStatus(database data.Database) *lightning.Command {
	return &lightning.Command{
		Name:        "status",
		Description: "get the status of the bridge that this channel is part of",
		Executor: func(opts *lightning.CommandOptions) {
			bridge, err := database.GetBridgeByChannel(opts.ChannelID)
			if err != nil {
				sendErr(err, "failed to get channel for bridge", opts)

				return
			} else if bridge.ID == "" {
				sendErr(bridgeNotFoundError{}, "this channel is not part of a bridge", opts)

				return
			}

			status := "Channels:\n\n"

			for i, channel := range bridge.Channels {
				status += strconv.FormatInt(int64(i+1), 10) + ". `" + channel.ID + "`"

				if channel.Disabled.Read {
					status += " (subscribed)"
				}

				if channel.Disabled.Write {
					status += " (write disabled)"
				}

				status += "\n"
			}

			status += "\n\n" + getSettingsString(&bridge)

			if err = opts.Reply(getMessage("bridge status", status), false); err != nil {
				sendErr(err, "failed to respond to command", opts)
			}
		},
	}
}

func getSettingsString(bridge *data.Bridge) string {
	emoji := map[bool]string{true: "✔", false: "❌"}

	return "Settings: \n\n- AllowEveryone: `" + emoji[bridge.Settings.AllowEveryone] + "`\n"
}
