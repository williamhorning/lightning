package app

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// RegisterCommands setups commands for the bridge bot.
func RegisterCommands(bot *lightning.Bot, database data.Database, username string) {
	bot.AddCommand(lightning.Command{
		Name:        "about",
		Description: "describes the bot",
		Executor: getExecutor(
			"about lightning", "https://williamhorning.dev/lightning", false, `
Lightning is a project developing *truly powerful* cross-platform bots, with the underlying *Lightning framework*
being used for *Lightning bridge*, which is what runs *Bolt*, the hosted bridge bot. The goal is to also make the
framework itself usable by other developers, to create their own bots, and to make the bridge easy to self-host, while
also supporting the principles of connecting communities, extensibility, ease of use, and strength.

Lightning, the framework and bridge bot, is licensed under the MIT license. The framework and plugins will always
remain under the MIT license, though the bridge bot may have a different license in the future, but will always be
free to use. Bolt is also free to use, but is also subject to its Terms of Service.`),
	}, lightning.Command{
		Name:        "bridge",
		Description: "manage bridges between channels",
		Executor: getExecutor(
			"the `bridge` command", "", false,
			"This command allows you to create and manage bridges between channels on different platforms. "+
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
		Executor: getExecutor("help", "", false, "Hi, I'm "+username+" "+lightning.VERSION+"! available commands are:"+
			"\n- `about`: learn about this bot\n- `bridge`: manage bridges between channels\n- `help`: returns this "+
			"help message\n- `ping`: checks the one way ping of the bot\n\n"+
			"read the [docs](https://williamhorning.dev/lightning) for more help"),
	}, lightning.Command{
		Name:        "ping",
		Description: "check the bot's one way ping",
		Executor: getExecutor("Pong! 🏓", "", false, func(opts *lightning.CommandOptions) string {
			return strconv.FormatInt(time.Since(opts.Time).Milliseconds(), 10) + "ms"
		}),
	})
}

func getExecutor[T string | func(*lightning.CommandOptions) string](
	title, url string, secret bool, description T,
) func(*lightning.CommandOptions) {
	return func(opts *lightning.CommandOptions) {
		opts.Reply(&lightning.Message{Embeds: []lightning.Embed{{
			Title: title, URL: url, Description: (func() string {
				if str, ok := any(description).(string); ok {
					return str
				} else if fn, ok := any(description).(func(*lightning.CommandOptions) string); ok {
					return fn(opts)
				}

				return ""
			})(), Color: 0x487c7e,
			Footer: &lightning.EmbedFooter{
				Text:    "powered by lightning",
				IconURL: "https://williamhorning.dev/assets/lightning.png",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}}}, secret)
	}
}

type alreadyInBridgeError struct{}

func (alreadyInBridgeError) Error() string { return "this channel is already in a bridge" }

const notInBridge = "this channel is not in a bridge"

func getErr(msg string, err error) string {
	return "uh oh! looks like you got struck by an error: " +
		msg + "\n\n```\n" + err.Error() + "\n```\nif you think this is a bug, or need more help, see the " +
		"[docs](https://williamhorning.dev/lightning/bridge)"
}

func prepareChannelForBridge(db data.Database, opts *lightning.CommandOptions) (*data.BridgeChannel, error) {
	if br, err := db.GetBridgeByChannel(opts.ChannelID); br.ID != "" || err != nil {
		return nil, alreadyInBridgeError{}
	}

	channelData, err := opts.Bot.SetupChannel(opts.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to setup %s for bridge: %w", opts.ChannelID, err)
	}

	return &data.BridgeChannel{Data: channelData, ID: opts.ChannelID}, nil
}

func getCreate(database data.Database) lightning.Command {
	return lightning.Command{
		Name:        "create",
		Description: "Create a new bridge containing this channel",
		Executor: getExecutor("bridge create", "", true, func(opts *lightning.CommandOptions) string {
			channel, err := prepareChannelForBridge(database, opts)
			if err != nil {
				return getErr("failed to prepare channel data", err)
			}

			bridge := data.Bridge{
				ID:       ulid.Make().String(),
				Channels: []data.BridgeChannel{*channel},
				Settings: data.BridgeSettings{},
			}

			if err := database.CreateBridge(bridge); err != nil {
				return getErr("failed to insert bridge row", err)
			}

			return "you can now join the bridge you made in other channels by using ||`" +
				opts.Prefix + "bridge join " + bridge.ID + "`||. Keep that command secret!"
		}),
	}
}

func getJoin(database data.Database, name string) lightning.Command {
	cmd := lightning.Command{
		Name:        name,
		Description: "Join an existing bridge with the given ID",
		Arguments:   []lightning.CommandArgument{{Name: "id", Description: "bridge ID", Required: true}},
		Executor: getExecutor("bridge join", "", true, func(opts *lightning.CommandOptions) string {
			bridge, err := database.GetBridge(opts.Arguments["id"])
			if err != nil || bridge.ID == "" {
				return "that bridge doesn't exist"
			}

			channel, err := prepareChannelForBridge(database, opts)
			if err != nil {
				return getErr("failed to prepare channel data", err)
			}

			if name == "subscribe" {
				channel.Disabled.Read = true
			}

			bridge.Channels = append(bridge.Channels, *channel)

			if err := database.CreateBridge(bridge); err != nil {
				return getErr("failed to update bridge row", err)
			}

			return "bridge joined (or subscribed) successfully!"
		}),
	}

	if name == "subscribe" {
		cmd.Description = "Subscribe to an existing bridge with the given ID (read-only)"
	}

	return cmd
}

func getLeave(database data.Database) lightning.Command {
	return lightning.Command{
		Name:        "leave",
		Description: "Leave the bridge that this channel is part of",
		Arguments:   []lightning.CommandArgument{{Name: "id", Description: "bridge ID", Required: true}},
		Executor: getExecutor("bridge leave", "", true, func(opts *lightning.CommandOptions) string {
			bridge, err := database.GetBridge(opts.Arguments["id"])
			if err != nil || bridge.ID == "" || bridge.ID != opts.Arguments["id"] {
				return notInBridge
			}

			for idx, channel := range bridge.Channels {
				if channel.ID == opts.ChannelID {
					bridge.Channels = slices.Delete(bridge.Channels, idx, idx+1)

					break
				}
			}

			if err := database.CreateBridge(bridge); err != nil {
				return getErr("failed to update bridge row", err)
			}

			return "channel removed from the bridge."
		}),
	}
}

func getReset(database data.Database) lightning.Command {
	return lightning.Command{
		Name:        "reset",
		Description: "Tries to reset the state of channels in a bridge",
		Executor: getExecutor("bridge reset", "", false, func(opts *lightning.CommandOptions) string {
			bridge, err := database.GetBridgeByChannel(opts.ChannelID)
			if err != nil || bridge.ID == "" {
				return notInBridge
			}

			restored := 0

			for idx, ch := range bridge.Channels {
				if !ch.Disabled.Read && !ch.Disabled.Write {
					continue
				}

				data, err := opts.Bot.SetupChannel(ch.ID)
				if err != nil {
					continue
				}

				bridge.Channels[idx].Data = data
				bridge.Channels[idx].Disabled.Read = false
				bridge.Channels[idx].Disabled.Write = false
				restored++
			}

			if err := database.CreateBridge(bridge); err != nil {
				return getErr("failed to update bridge row", err)
			}

			return "finished resetting: " + strconv.FormatInt(int64(restored), 10) + " channels reset"
		}),
	}
}

func getStatus(database data.Database) lightning.Command {
	return lightning.Command{
		Name:        "status",
		Description: "view channels and settings in this bridge",
		Executor: getExecutor("bridge status", "", false, func(opts *lightning.CommandOptions) string {
			bridge, err := database.GetBridgeByChannel(opts.ChannelID)
			if err != nil || bridge.ID == "" {
				return notInBridge
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
			}

			return "\n**Settings:**\n- AllowEveryone: " + strconv.FormatBool(bridge.Settings.AllowEveryone)
		}),
	}
}

func getToggle(database data.Database) lightning.Command {
	return lightning.Command{
		Name:        "toggle",
		Description: "toggle a bridge setting",
		Arguments:   []lightning.CommandArgument{{Name: "setting", Description: "setting name", Required: true}},
		Executor: getExecutor("bridge toggle", "", false, func(opts *lightning.CommandOptions) string {
			bridge, err := database.GetBridgeByChannel(opts.ChannelID)
			if err != nil || bridge.ID == "" {
				return notInBridge
			}

			switch strings.ToLower(opts.Arguments["setting"]) {
			case "alloweveryone", "allow_everyone":
				bridge.Settings.AllowEveryone = !bridge.Settings.AllowEveryone
			default:
				return "unknown setting: " + opts.Arguments["setting"]
			}

			if err := database.CreateBridge(bridge); err != nil {
				return getErr("failed to update bridge row", err)
			}

			return "setting toggled successfully."
		}),
	}
}
