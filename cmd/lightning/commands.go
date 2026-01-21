package main

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/oklog/ulid/v2"
)

func registerCommands(bot *lightning.Bot, database *database, username string) {
	bot.AddCommand(lightning.Command{
		Name:        "bridge",
		Description: "manage bridges between channels",
		Executor: getExecutor(
			"the `bridge` command", "", false,
			"This command allows you to create and manage bridges between channels on different platforms. \n\n"+
				"Subcommands that are available are:\n"+
				"- `create`: Create a new bridge containing this channel.\n"+
				"- `join <id>`: Join an existing bridge with the given ID.\n"+
				"- `subscribe <id>`: Subscribe to an existing bridge with the given ID (read-only).\n"+
				"- `leave <id>`: Leave the bridge that this channel is part of.\n"+
				"- `reset`: Tries to reset the state of channels in a bridge\n"+
				"- `toggle <setting>`: Toggle a setting for the bridge that this channel is part of.\n"+
				"- `status`: Get the status of the bridge that this channel is part of.\n\n"+
				"Available settings are: `allow_everyone`.",
		),
		Subcommands: map[string]lightning.Command{
			"create": getCreate(database), "join": getJoin(database, "join"),
			"subscribe": getJoin(database, "subscribe"), "leave": getLeave(database), "reset": getReset(database),
			"status": getStatus(database), "toggle": getToggle(database),
		},
	}, lightning.Command{
		Name:        "help",
		Description: "get help with the bot",
		Executor: getExecutor(
			"help for "+username, "https://williamhorning.dev/lightning", false,
			"Hi, I'm "+username+" v0.8.1!\n\n"+
				"Available commands are: \n"+
				"- `bridge`: manage bridges between channels\n"+
				"- `help`: returns this help message\n"+
				"- `ping`: checks the one way ping of the bot\n\n"+
				"See the [docs](https://williamhorning.dev/lightning) for more help"),
	}, lightning.Command{
		Name:        "ping",
		Description: "check the bot's one way ping",
		Executor: getExecutor("Pong! 🏓", "", false, func(opts *lightning.CommandOptions) string {
			return strconv.FormatInt(time.Since(opts.Time).Milliseconds(), 10) + "ms (one-way)"
		}),
	})
}

func getExecutor[T string | func(*lightning.CommandOptions) string](
	title, url string, secret bool, description T,
) func(*lightning.CommandOptions) {
	return func(opts *lightning.CommandOptions) {
		opts.Reply(&lightning.Message{Embeds: []lightning.Embed{{
			Title: title, URL: url, Description: (func() string {
				switch v := any(description).(type) {
				case string:
					return v
				case func(*lightning.CommandOptions) string:
					return v(opts)
				default:
					return ""
				}
			})(), Color: 0x487c7e,
			Footer: &lightning.EmbedFooter{
				Text:    "powered by lightning",
				IconURL: "https://williamhorning.dev/assets/lightning.png",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}}}, secret)
	}
}

func getErr(prefix, action string, err error) string {
	return "Something went wrong while running that command when it tried to " + action + ". Try `" +
		prefix + "bridge help` or see the [docs](https://williamhorning.dev/lightning) for help. \n\n```\n" +
		err.Error() + "\n```"
}

func prepareChannelForBridge(db *database, opts *lightning.CommandOptions) (*bridgeChannel, string) {
	if br, err := db.getBridgeByChannel(opts.ChannelID); br.ID != "" || err != nil {
		return nil, "You are already in a bridge, so you can't be in another one. If you didn't expect this, try `" +
			opts.Prefix + "bridge status` or `" + opts.Prefix + "bridge help`."
	}

	channelData, err := opts.Bot.SetupChannel(opts.ChannelID)
	if err != nil {
		return nil, getErr(opts.Prefix, "setup the channel `"+opts.ChannelID+"`", err)
	}

	return &bridgeChannel{Data: channelData, ID: opts.ChannelID}, ""
}

func getCreate(database *database) lightning.Command {
	return lightning.Command{
		Name:        "create",
		Description: "Create a new bridge containing this channel",
		Executor: getExecutor("bridge create", "", true, func(opts *lightning.CommandOptions) string {
			channel, msg := prepareChannelForBridge(database, opts)
			if msg != "" {
				return msg
			}

			bridge := bridge{
				ID:       ulid.Make().String(),
				Channels: []bridgeChannel{*channel},
				Settings: bridgeSettings{},
			}

			if err := database.createBridge(bridge); err != nil {
				return getErr(opts.Prefix, "update the database", err)
			}

			return "Bridge created successfully! You can now join it with the following command: ||`" +
				opts.Prefix + "bridge join " + bridge.ID + "`||. Keep that command secret, or else anyone could join!"
		}),
	}
}

func getJoin(database *database, name string) lightning.Command {
	cmd := lightning.Command{
		Name:        name,
		Description: "Join an existing bridge with the given ID",
		Arguments:   []lightning.CommandArgument{{Name: "id", Description: "bridge ID"}},
		Executor: getExecutor("bridge join", "", true, func(opts *lightning.CommandOptions) string {
			bridge, err := database.getBridge(opts.Arguments["id"])
			if err != nil || bridge.ID == "" {
				return "You can't join a bridge that doesn't exist. Check if you made one, " +
					"or if you provided the wrong ID. Try `" + opts.Prefix + "bridge create` or `" +
					opts.Prefix + "bridge help`."
			}

			channel, msg := prepareChannelForBridge(database, opts)
			if msg != "" {
				return msg
			}

			if name == "subscribe" {
				channel.Disabled.Read = true
			}

			bridge.Channels = append(bridge.Channels, *channel)

			if err := database.createBridge(bridge); err != nil {
				return getErr(opts.Prefix, "update the database", err)
			}

			return "You successfully joined the bridge."
		}),
	}

	if name == "subscribe" {
		cmd.Description = "Subscribe to an existing bridge with the given ID (read-only)"
	}

	return cmd
}

func getLeave(database *database) lightning.Command {
	return lightning.Command{
		Name:        "leave",
		Description: "Leave the bridge that this channel is part of",
		Arguments:   []lightning.CommandArgument{{Name: "id", Description: "bridge ID"}},
		Executor: getExecutor("bridge leave", "", true, func(opts *lightning.CommandOptions) string {
			bridge, err := database.getBridge(opts.Arguments["id"])
			if err != nil || bridge.ID == "" || bridge.ID != opts.Arguments["id"] {
				return "You can't leave a bridge if your aren't in a bridge, or provided the wrong ID. " +
					"Try `" + opts.Prefix + "bridge join` or `" + opts.Prefix + "bridge help`."
			}

			for idx, channel := range bridge.Channels {
				if channel.ID == opts.ChannelID {
					bridge.Channels = slices.Delete(bridge.Channels, idx, idx+1)

					break
				}
			}

			if err := database.createBridge(bridge); err != nil {
				return getErr(opts.Prefix, "update the database", err)
			}

			return "You successfully left the bridge."
		}),
	}
}

func getReset(database *database) lightning.Command {
	return lightning.Command{
		Name:        "reset",
		Description: "Tries to reset the state of channels in a bridge",
		Executor: getExecutor("bridge reset", "", false, func(opts *lightning.CommandOptions) string {
			bridge, err := database.getBridgeByChannel(opts.ChannelID)
			if err != nil || bridge.ID == "" {
				return "You can't reset channels in a bridge without being in a bridge. Try `" +
					opts.Prefix + "bridge join` or `" + opts.Prefix + "bridge help`."
			}

			errors := make([]string, 0, len(bridge.Channels))

			for idx, channel := range bridge.Channels {
				if !channel.Disabled.Read && !channel.Disabled.Write {
					continue
				}

				data, err := opts.Bot.SetupChannel(channel.ID)
				if err != nil {
					errors = append(errors, getErr(channel.ID, "setup the channel `"+channel.ID+"`", err))

					continue
				}

				bridge.Channels[idx].Data = data
				bridge.Channels[idx].Disabled.Read = false
				bridge.Channels[idx].Disabled.Write = false
			}

			if err := database.createBridge(bridge); err != nil {
				return getErr(opts.Prefix, "update the database", err)
			}

			errStr := ""
			if len(errors) != 0 {
				errStr = " The following errors occurred: \n\n" + strings.Join(errors, "\n\n")
			}

			return "Finished resetting channels in the bridge: " +
				strconv.FormatInt(int64(len(bridge.Channels)-len(errors)), 10) + " channels were reset." + errStr
		}),
	}
}

func getStatus(database *database) lightning.Command {
	return lightning.Command{
		Name:        "status",
		Description: "view channels and settings in this bridge",
		Executor: getExecutor("bridge status", "", false, func(opts *lightning.CommandOptions) string {
			bridge, err := database.getBridgeByChannel(opts.ChannelID)
			if err != nil || bridge.ID == "" {
				return "You are not in a bridge right now. If you didn't expect this, try `" +
					opts.Prefix + "bridge join` or `" + opts.Prefix + "bridge help`."
			}

			status := "**Channels:**\n"

			for _, channel := range bridge.Channels {
				status += "- `" + channel.ID + "`"

				switch {
				case channel.Disabled.Read && channel.Disabled.Write:
					status += " (disabled - try `" + opts.Prefix + "bridge reset` to fix this)"
				case channel.Disabled.Read:
					status += " (subscribed - to enable this channel, try `" + opts.Prefix + "bridge reset`)"
				case channel.Disabled.Write:
					status += " (read-only - try `" + opts.Prefix + "bridge reset` to fix this)"
				default:
				}

				status += "\n"
			}

			return status + "\n**Settings:**\n- AllowEveryone: " + strconv.FormatBool(bridge.Settings.AllowEveryone)
		}),
	}
}

func getToggle(database *database) lightning.Command {
	return lightning.Command{
		Name:        "toggle",
		Description: "toggle a bridge setting",
		Arguments:   []lightning.CommandArgument{{Name: "setting", Description: "setting name"}},
		Executor: getExecutor("bridge toggle", "", false, func(opts *lightning.CommandOptions) string {
			bridge, err := database.getBridgeByChannel(opts.ChannelID)
			if err != nil || bridge.ID == "" {
				return "You can't toggle settings without being in a bridge. Try `" +
					opts.Prefix + "bridge join` or `" + opts.Prefix + "bridge help`."
			}

			switch strings.ToLower(opts.Arguments["setting"]) {
			case "alloweveryone", "allow_everyone":
				bridge.Settings.AllowEveryone = !bridge.Settings.AllowEveryone
			default:
				return "`" + opts.Arguments["setting"] + "` is not a known setting! Try `" +
					opts.Prefix + "bridge help` for valid options."
			}

			if err := database.createBridge(bridge); err != nil {
				return getErr(opts.Prefix, "update the database", err)
			}

			return "Toggled `" + opts.Arguments["setting"] + "` successfully. Try `" + opts.Prefix +
				"bridge status` for the current value."
		}),
	}
}
