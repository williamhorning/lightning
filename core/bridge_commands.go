package lightning

import (
	"slices"
	"strconv"

	"github.com/oklog/ulid/v2"
)

func bridgeCommand(db Database) Command {
	return Command{
		Name:        "bridge",
		Description: "manage bridges between channels",
		Executor: func(opts CommandOptions) (string, error) {
			return "take a look at the subcommands of this command", nil
		},
		Subcommands: []Command{
			{
				Name:        "create",
				Description: "create a new bridge",
				Arguments:   []CommandArgument{{"name", "the name to use for the bridge", true}},
				Executor: func(options CommandOptions) (string, error) {
					return createCommand(db, options)
				},
			},
			{
				Name:        "join",
				Description: "join an existing bridge",
				Arguments:   []CommandArgument{{"id", "the ID of the bridge to join", true}},
				Executor: func(options CommandOptions) (string, error) {
					return joinCommand(db, options, false)
				},
			},
			{
				Name:        "subscribe",
				Description: "subscribe to a bridge",
				Arguments:   []CommandArgument{{"id", "the ID of the bridge to subscribe to", true}},
				Executor: func(options CommandOptions) (string, error) {
					return joinCommand(db, options, true)
				},
			},
			{
				Name:        "leave",
				Description: "leave a bridge",
				Arguments:   []CommandArgument{{"id", "the ID of the bridge to leave", true}},
				Executor: func(options CommandOptions) (string, error) {
					return leaveCommand(db, options)
				},
			},
			{
				Name:        "toggle",
				Description: "toggle settings for a bridge",
				Arguments:   []CommandArgument{{"setting", "the bridge setting to toggle", true}},
				Executor: func(options CommandOptions) (string, error) {
					return toggleCommand(db, options)
				},
			},
			{
				Name:        "status",
				Description: "get the status of the bridge in this channel",
				Executor: func(options CommandOptions) (string, error) {
					return statusCommand(db, options)
				},
			},
		},
	}
}

func prepareChannelForBridge(db Database, opts CommandOptions) (BridgeChannel, string) {
	Log.Trace().Str("channel", opts.ChannelID).Str("plugin", opts.Plugin).Msg("Adding channel to bridge")

	if br, err := db.getBridgeByChannel(opts.ChannelID); br.ID != "" || err != nil {
		return BridgeChannel{}, "This channel is already part of a bridge. Please leave the bridge first."
	}

	plugin, ok := Plugins.Get(opts.Plugin)
	if !ok {
		return BridgeChannel{}, LogError(ErrPluginNotFound, "Failed to add channel to bridge using plugin",
			map[string]any{"plugin": opts.Plugin, "channel": opts.ChannelID}, ReadWriteDisabled{}).Error()
	}

	data, err := plugin.SetupChannel(opts.ChannelID)
	if err != nil {
		return BridgeChannel{}, LogError(err, "Failed to setup channel for bridge",
			map[string]any{"plugin": plugin.Name(), "channel": opts.ChannelID}, ReadWriteDisabled{}).Error()
	}

	return BridgeChannel{
		ID:       opts.ChannelID,
		Data:     data,
		Plugin:   plugin.Name(),
		Disabled: ReadWriteDisabled{false, false},
	}, ""
}

func createCommand(db Database, opts CommandOptions) (string, error) {
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
		return LogError(err, "Failed to create bridge in database",
			map[string]any{"bridge": bridge}, ReadWriteDisabled{}).Error(), nil
	}

	Log.Debug().Str("bridge_id", bridge.ID).Str("channel", opts.ChannelID).Msg("Bridge created successfully")
	return "Bridge created successfully! You can now join it using `" + opts.Prefix + "bridge join " + bridge.ID + "`.", nil
}

func joinCommand(db Database, opts CommandOptions, subscribe bool) (string, error) {
	id := opts.Arguments["id"]

	ch, errMsg := prepareChannelForBridge(db, opts)
	if errMsg != "" {
		return errMsg, nil
	}

	br, err := db.getBridge(id)
	if err != nil {
		return LogError(err, "Failed to get bridge from database",
			map[string]any{"bridge_id": id}, ReadWriteDisabled{}).Error(), nil
	} else if br.ID == "" {
		return "No bridge found with the provided ID.", nil
	}

	ch.Disabled = ReadWriteDisabled{subscribe, false}
	br.Channels = append(br.Channels, ch)

	if err := db.createBridge(br); err != nil {
		return LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": br}, ReadWriteDisabled{}).Error(), nil
	}

	Log.Debug().Str("bridge_id", br.ID).Str("channel", opts.ChannelID).Msg("Channel joined bridge successfully")
	return "Bridge joined successfully!", nil
}

func leaveCommand(db Database, opts CommandOptions) (string, error) {
	id := opts.Arguments["id"]

	br, err := db.getBridgeByChannel(opts.ChannelID)
	if err != nil {
		return LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, ReadWriteDisabled{}).Error(), nil
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
		return LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": br}, ReadWriteDisabled{}).Error(), nil
	}

	return "You have successfully left the bridge.", nil
}

func toggleCommand(db Database, opts CommandOptions) (string, error) {
	setting := opts.Arguments["setting"]

	br, err := db.getBridgeByChannel(opts.ChannelID)
	if err != nil {
		return LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, ReadWriteDisabled{}).Error(), nil
	} else if br.ID == "" {
		return "You are not in a bridge.", nil
	}

	if setting != "allow_everyone" {
		return "That setting does not exist. Available settings are: `allow_everyone`.", nil
	}

	br.Settings.AllowEveryone = !br.Settings.AllowEveryone

	if err := db.createBridge(br); err != nil {
		return LogError(err, "Failed to update bridge in database",
			map[string]any{"bridge": br}, ReadWriteDisabled{}).Error(), nil
	}

	return "Bridge settings updated successfully", nil
}

func statusCommand(db Database, opts CommandOptions) (string, error) {
	br, err := db.getBridgeByChannel(opts.ChannelID)
	if err != nil {
		return LogError(err, "Failed to get bridge from database",
			map[string]any{"channel": opts.ChannelID}, ReadWriteDisabled{}).Error(), nil
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
