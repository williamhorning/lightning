package bridge

import (
	"slices"
	"strconv"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func bridgeCommand(db Database) lightning.Command {
	return lightning.Command{
		Name:        "bridge",
		Description: "manage bridges between channels",
		Executor: func(opts lightning.CommandOptions) (string, error) {
			return "take a look at the subcommands of this command", nil
		},
		Subcommands: []lightning.Command{
			{
				Name:        "create",
				Description: "create a new bridge",
				Arguments:   []lightning.CommandArgument{{Name: "name", Description: "the name to use for the bridge", Required: true}},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return createCommand(db, options)
				},
			},
			{
				Name:        "join",
				Description: "join an existing bridge",
				Arguments:   []lightning.CommandArgument{{Name: "id", Description: "the ID of the bridge to join", Required: true}},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return joinCommand(db, options, false)
				},
			},
			{
				Name:        "subscribe",
				Description: "subscribe to a bridge",
				Arguments:   []lightning.CommandArgument{{Name: "id", Description: "the ID of the bridge to subscribe to", Required: true}},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return joinCommand(db, options, true)
				},
			},
			{
				Name:        "leave",
				Description: "leave a bridge",
				Arguments:   []lightning.CommandArgument{{Name: "id", Description: "the ID of the bridge to leave", Required: true}},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return leaveCommand(db, options)
				},
			},
			{
				Name:        "toggle",
				Description: "toggle settings for a bridge",
				Arguments:   []lightning.CommandArgument{{Name: "setting", Description: "the bridge setting to toggle", Required: true}},
				Executor: func(options lightning.CommandOptions) (string, error) {
					return toggleCommand(db, options)
				},
			},
			{
				Name:        "status",
				Description: "get the status of the bridge in this channel",
				Executor: func(options lightning.CommandOptions) (string, error) {
					return statusCommand(db, options)
				},
			},
		},
	}
}

func prepareChannelForBridge(db Database, opts lightning.CommandOptions) (BridgeChannel, string) {
	lightning.Log.Trace().Str("channel", opts.ChannelID).Str("plugin", opts.Plugin).Msg("Adding channel to bridge")

	if br, err := db.getBridgeByChannel(opts.ChannelID); br.ID != "" || err != nil {
		return BridgeChannel{}, "This channel is already part of a bridge. Please leave the bridge first."
	}

	plugin, ok := lightning.Plugins.Get(opts.Plugin)
	if !ok {
		return BridgeChannel{}, lightning.LogError(lightning.ErrPluginNotFound, "Failed to add channel to bridge using plugin",
			map[string]any{"plugin": opts.Plugin, "channel": opts.ChannelID}, lightning.ChannelDisabled{}).Error()
	}

	data, err := plugin.SetupChannel(opts.ChannelID)
	if err != nil {
		return BridgeChannel{}, lightning.LogError(err, "Failed to setup channel for bridge",
			map[string]any{"plugin": plugin.Name(), "channel": opts.ChannelID}, lightning.ChannelDisabled{}).Error()
	}

	return BridgeChannel{
		ID:       opts.ChannelID,
		Data:     data,
		Plugin:   plugin.Name(),
		Disabled: lightning.ChannelDisabled{Read: false, Write: false},
	}, ""
}

func createCommand(db Database, opts lightning.CommandOptions) (string, error) {
	ch, errMsg := prepareChannelForBridge(db, opts)
	if errMsg != "" {
		return errMsg, nil
	}

	bridge := Bridge{
		ID:       ulid.Make().String(),
		Name:     opts.Arguments["name"],
		Channels: []BridgeChannel{ch},
		Settings: BridgeSettings{false},
	}

	if err := db.createBridge(bridge); err != nil {
		return lightning.LogError(err, "Failed to create bridge in database",
			map[string]any{"bridge": bridge}, lightning.ChannelDisabled{}).Error(), nil
	}

	lightning.Log.Debug().Str("bridge_id", bridge.ID).Str("channel", opts.ChannelID).Msg("Bridge created successfully")
	return "Bridge created successfully! You can now join it using ||`" + opts.Prefix + "bridge join " + bridge.ID + "`||. Keep this command secret!", nil
}

func joinCommand(db Database, opts lightning.CommandOptions, subscribe bool) (string, error) {
	id := opts.Arguments["id"]

	ch, errMsg := prepareChannelForBridge(db, opts)
	if errMsg != "" {
		return errMsg, nil
	}

	br, err := db.getBridge(id)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"bridge_id": id}, lightning.ChannelDisabled{}).Error(), nil
	} else if br.ID == "" {
		return "No bridge found with the provided ID.", nil
	}

	ch.Disabled = lightning.ChannelDisabled{Read: subscribe, Write: false}
	br.Channels = append(br.Channels, ch)

	if err := db.createBridge(br); err != nil {
		return lightning.LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": br}, lightning.ChannelDisabled{}).Error(), nil
	}

	lightning.Log.Debug().Str("bridge_id", br.ID).Str("channel", opts.ChannelID).Msg("Channel joined bridge successfully")
	return "Bridge joined successfully!", nil
}

func leaveCommand(db Database, opts lightning.CommandOptions) (string, error) {
	id := opts.Arguments["id"]

	br, err := db.getBridgeByChannel(opts.ChannelID)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, lightning.ChannelDisabled{}).Error(), nil
	} else if br.ID == "" {
		return "You are not in a bridge.", nil
	}

	if br.ID != id {
		return "This channel is not part of the specified bridge.", nil
	}

	for i, channel := range br.Channels {
		if channel.ID == opts.ChannelID {
			br.Channels = slices.Delete(br.Channels, i, i+1)
			break
		}
	}

	if err := db.createBridge(br); err != nil {
		return lightning.LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": br}, lightning.ChannelDisabled{}).Error(), nil
	}

	return "You have successfully left the bridge.", nil
}

func toggleCommand(db Database, opts lightning.CommandOptions) (string, error) {
	setting := opts.Arguments["setting"]

	br, err := db.getBridgeByChannel(opts.ChannelID)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, lightning.ChannelDisabled{}).Error(), nil
	} else if br.ID == "" {
		return "You are not in a bridge.", nil
	}

	if setting != "allow_everyone" {
		return "That setting does not exist. Available settings are: `allow_everyone`.", nil
	}

	br.Settings.AllowEveryone = !br.Settings.AllowEveryone

	if err := db.createBridge(br); err != nil {
		return lightning.LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": br}, lightning.ChannelDisabled{}).Error(), nil
	}

	return "Bridge settings updated successfully", nil
}

func statusCommand(db Database, opts lightning.CommandOptions) (string, error) {
	br, err := db.getBridgeByChannel(opts.ChannelID)
	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, lightning.ChannelDisabled{}).Error(), nil
	} else if br.ID == "" {
		return "You are not in a bridge.", nil
	}

	status := "Name: `" + br.Name + "`\n\nChannels:\n"

	for i, channel := range br.Channels {
		status += strconv.Itoa(i) + ". `" + channel.ID + "` on `" + channel.Plugin + "`"
		if channel.IsDisabled().Read {
			status += " (subscribed)"
		}
		if channel.IsDisabled().Write {
			status += " (write disabled)"
		}
		status += "\n"
	}

	status += "\nSettings:\n"
	status += "- Allow Everyone: `" + (map[bool]string{true: "✔", false: "❌"})[br.Settings.AllowEveryone] + "`\n"

	return status, nil
}
