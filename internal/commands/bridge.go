package commands

import (
	"fmt"
	"log"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// BridgeCommand provides `!bridge`.
func BridgeCommand(database data.Database) *lightning.Command {
	return &lightning.Command{
		Name:        "bridge",
		Description: "manage bridges between channels",
		Executor: func(opts *lightning.CommandOptions) {
			if err := opts.Reply(getMessage("the `bridge` command",
				"This command allows you to create and manage bridges between channels on different platforms. "+
					"Subcommands that are available are:\n"+
					"- `create`: Create a new bridge in this channel.\n"+
					"- `join <id>`: Join an existing bridge with the given ID.\n"+
					"- `subscribe <id>`: Subscribe to an existing bridge with the given ID (read-only).\n"+
					"- `leave <id>`: Leave the bridge that this channel is part of.\n"+
					"- `toggle <setting>`: Toggle a setting for the bridge that this channel is part of.\n"+
					"- `status`: Get the status of the bridge that this channel is part of.\n\n"+
					"Available settings are: `allow_everyone`."), false); err != nil {
				log.Printf("failed to reply to bridge command: %v\n", err)
			}
		},
		Subcommands: map[string]*lightning.Command{
			"create": bridgeCreate(database), "join": bridgeJoin(database, "join"),
			"subscribe": bridgeJoin(database, "subscribe"), "leave": bridgeLeave(database),
			"toggle": bridgeToggle(database), "status": bridgeStatus(database),
		},
	}
}

func prepareChannelForBridge(db data.Database, opts *lightning.CommandOptions) (*data.BridgeChannel, error) {
	if br, err := db.GetBridgeByChannel(opts.ChannelID); br.ID != "" || err != nil {
		return nil, alreadyInBridgeError{}
	}

	channelData, err := opts.Bot.SetupChannel(opts.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to setup channel for bridge: %w\n\tchannel: %s", err, opts.ChannelID)
	}

	return &data.BridgeChannel{Data: channelData, ID: opts.ChannelID, Disabled: lightning.ChannelDisabled{}}, nil
}

type alreadyInBridgeError struct{}

func (alreadyInBridgeError) Error() string {
	return "this channel is already part of a bridge. please leave the bridge first"
}

type bridgeNotFoundError struct{}

func (bridgeNotFoundError) Error() string {
	return "bridge not found"
}

type missingArgumentError struct {
	argument string
}

func (e missingArgumentError) Error() string {
	return "argument " + e.argument + " is required"
}
