package bridge

import (
	"slices"
	"strconv"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/pkg/lightning"
)

const notInBridge = "You're not in a bridge"

func bridgeCommand(database Database) lightning.Command {
	return lightning.Command{
		Name:        "bridge",
		Description: "manage bridges between channels",
		Executor: func(_ lightning.CommandOptions) (string, error) {
			return "take a look at the subcommands of this command", nil
		},
		Subcommands: []lightning.Command{
			{
				Name:        "create",
				Description: "create a new bridge",
				Arguments: []lightning.CommandArgument{
					{Name: "name", Description: "the name to use for the bridge", Required: true},
				},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return createCommand(database, options)
				},
			},
			{
				Name:        "join",
				Description: "join an existing bridge",
				Arguments:   arguments("join"),
				Executor: func(options lightning.CommandOptions) (string, error) {
					return joinCommand(database, options, false)
				},
			},
			{
				Name:        "subscribe",
				Description: "subscribe to a bridge",
				Arguments:   arguments("subscribe to"),
				Executor: func(options lightning.CommandOptions) (string, error) {
					return joinCommand(database, options, true)
				},
			},
			{
				Name:        "leave",
				Description: "leave a bridge",
				Arguments:   arguments("leave"),
				Executor: func(options lightning.CommandOptions) (string, error) {
					return leaveCommand(database, options)
				},
			},
			{
				Name:        "toggle",
				Description: "toggle settings for a bridge",
				Arguments: []lightning.CommandArgument{
					{Name: "setting", Description: "the bridge setting to toggle", Required: true},
				},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return toggleCommand(database, options)
				},
			},
			{
				Name:        "status",
				Description: "get the status of the bridge in this channel",
				Executor: func(options lightning.CommandOptions) (string, error) {
					return statusCommand(database, options)
				},
			},
		},
	}
}

func arguments(to string) []lightning.CommandArgument {
	return []lightning.CommandArgument{
		{Name: "id", Description: "the ID of the bridge to " + to, Required: true},
	}
}

func prepareChannelForBridge(db Database, opts lightning.CommandOptions) (bridgeChannel, string) {
	if br, err := getBridgeByChannel(db, opts.ChannelID); br.ID != "" || err != nil {
		return bridgeChannel{}, "This channel is already part of a bridge. Please leave the bridge first."
	}

	data, err := opts.Bot.SetupChannel(opts.ChannelID)
	if err != nil {
		return bridgeChannel{}, lightning.LogError(err, "Failed to setup channel for bridge",
			map[string]any{"channel": opts.ChannelID}, nil).Error()
	}

	return bridgeChannel{
		ID:       opts.ChannelID,
		Data:     data,
		Disabled: lightning.ChannelDisabled{},
	}, ""
}

func createCommand(database Database, opts lightning.CommandOptions) (string, error) {
	channel, errMsg := prepareChannelForBridge(database, opts)
	if errMsg != "" {
		return errMsg, nil
	}

	bridgeData := bridge{
		ID:       ulid.Make().String(),
		Name:     opts.Arguments["name"],
		Channels: []bridgeChannel{channel},
		Settings: bridgeSettings{false},
	}

	if err := database.createBridge(bridgeData); err != nil {
		return lightning.LogError(err, "Failed to create bridge in database",
			map[string]any{"bridge": bridgeData}, nil).Error(), nil
	}

	join := opts.Prefix + "bridge join " + bridgeData.ID

	return "Bridge created successfully! You can now join it using ||`" + join + "`||. Keep this command secret!", nil
}

func joinCommand(database Database, opts lightning.CommandOptions, subscribe bool) (string, error) {
	channel, errMsg := prepareChannelForBridge(database, opts)
	if errMsg != "" {
		return errMsg, nil
	}

	bridgeData, err := database.getBridge(opts.Arguments["id"])
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"bridge_id": opts.Arguments["id"]}, nil).Error(), nil
	} else if bridgeData.ID == "" {
		return "No bridge found with the provided ID.", nil
	}

	channel.Disabled = lightning.ChannelDisabled{Read: subscribe, Write: false}
	bridgeData.Channels = append(bridgeData.Channels, channel)

	if err := database.createBridge(bridgeData); err != nil {
		return lightning.LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": bridgeData}, nil).Error(), nil
	}

	return "Bridge joined successfully!", nil
}

func leaveCommand(database Database, opts lightning.CommandOptions) (string, error) {
	bridgeData, err := getBridgeByChannel(database, opts.ChannelID)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, nil).Error(), nil
	} else if bridgeData.ID == "" {
		return notInBridge, nil
	}

	if bridgeData.ID != opts.Arguments["id"] {
		return "This channel is not part of the specified bridge.", nil
	}

	for idx, channel := range bridgeData.Channels {
		if compareChannelIDs(channel, opts.ChannelID) {
			bridgeData.Channels = slices.Delete(bridgeData.Channels, idx, idx+1)

			break
		}
	}

	if err := database.createBridge(bridgeData); err != nil {
		return lightning.LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": bridgeData}, nil).Error(), nil
	}

	return "You have successfully left the bridge.", nil
}

func toggleCommand(database Database, opts lightning.CommandOptions) (string, error) {
	setting := opts.Arguments["setting"]

	bridgeData, err := getBridgeByChannel(database, opts.ChannelID)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, nil).Error(), nil
	} else if bridgeData.ID == "" {
		return notInBridge, nil
	}

	if setting != "allow_everyone" {
		return "That setting does not exist. Available settings are: `allow_everyone`.", nil
	}

	bridgeData.Settings.AllowEveryone = !bridgeData.Settings.AllowEveryone

	if err := database.createBridge(bridgeData); err != nil {
		return lightning.LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": bridgeData}, nil).Error(), nil
	}

	return "Bridge settings updated successfully", nil
}

func statusCommand(db Database, opts lightning.CommandOptions) (string, error) {
	bridgeData, err := getBridgeByChannel(db, opts.ChannelID)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, nil).Error(), nil
	} else if bridgeData.ID == "" {
		return notInBridge, nil
	}

	status := "Name: `" + bridgeData.Name + "`\n\nChannels:\n"

	for i, channel := range bridgeData.Channels {
		status += strconv.Itoa(i+1) + ". `"

		if channel.DeprecatedPlugin == "" {
			status += channel.ID + "`"
		} else {
			status += channel.DeprecatedPlugin + "::" + channel.ID + "`"
		}

		if channel.isDisabled().Read {
			status += " (subscribed)"
		}

		if channel.isDisabled().Write {
			status += " (write disabled)"
		}

		status += "\n"
	}

	status += "\nSettings:\n"
	status += "- AllowEveryone: `" + (map[bool]string{true: "✔", false: "❌"})[bridgeData.Settings.AllowEveryone] + "`\n"

	return status, nil
}
