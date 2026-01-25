// Package discord provides a [lightning.Plugin] implementation for Discord.
// To use Discord support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("discord", discord.New)
//
//	bot.UsePluginType("discord", "", map[string]string{
//		// ...
//	})
package discord

import (
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Discord support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"api_host": "", // optional, can specify a non-Discord API implementation. must be paired with cdn_host
//		"cdn_host": "", // optional, can specify a non-Discord CDN implementation. must be paired with api_host
//		"token": "",    // required, a string with your Discord bot token
//	}
//
// Note that you MUST enable the Message Content intent for the plugin to work.
func New(cfg map[string]string) (lightning.Plugin, error) {
	if _, ok := cfg["api_host"]; !ok {
		cfg["api_host"] = "discord.com"
	}

	if _, ok := cfg["cdn_host"]; !ok {
		cfg["cdn_host"] = "cdn.discordapp.com"
	}

	session := &client{
		apiHost: cfg["api_host"], cdnHost: cfg["cdn_host"], frontend: "discord.com", version: "10",
		spacebar: cfg["api_host"] != "discord.com", token: cfg["token"], product: "discord",
		intents: intentsNotPrivileged | intentMessageContent,
	}

	if session.spacebar {
		session.frontend = "fermi.chat"
		session.product = "spacebar"
	}

	addHandler(session, eventReady, func(e *readyEvent) {
		log.Printf("%s: ready as %s in %d servers!\n", session.product, e.User.Username, len(e.Guilds))
		log.Printf("%s: invite at https://%s/oauth2/authorize?client_id=%s&permissions=8&scope=bot",
			session.product, session.frontend, e.Application.ID)
	})

	if err := session.connect(); err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %w", err)
	}

	return &discordPlugin{bot: session}, nil
}

type discordPlugin struct {
	bot *client
}

func (p *discordPlugin) SetupChannel(user, channel string) (map[string]string, error) {
	setup, ok := p.bot.getChannel(channel)
	if !ok {
		return nil, &permissionCheckError{"get channel on permissions check"}
	}

	guild, ok := p.bot.getGuild(setup.GuildID)
	if !ok {
		return nil, &permissionCheckError{"get guild on permissions check"}
	}

	member, ok := p.bot.getMember(setup.GuildID, snowflake(user))
	if !ok {
		return nil, &permissionCheckError{"get member on permissions check"}
	}

	for _, id := range member.Roles {
		for _, role := range guild.Roles {
			if id == string(role.ID) {
				perms, err := strconv.ParseInt(role.Permissions, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse role permissions for %q: %w", id, err)
				}

				if perms&8 == 8 {
					wh, err := p.bot.createWebhook(channel, channel)
					if err != nil {
						return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
					}

					return map[string]string{"id": string(wh.ID), "token": wh.Token}, nil
				}
			}
		}
	}

	return nil, &permissionCheckError{
		"find any administrative permissions you had. you may need to wait a minute or two for permissions to update",
	}
}

func (p *discordPlugin) SendMessage(original *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	if opts.CommandResponse {
		channel, err := p.bot.startDM(opts.CommandUser)
		if err != nil {
			return nil, err
		}

		original.RepliedTo = nil
		original.ChannelID = string(channel.ID)
	}

	msg := lightningToDiscordSendable(p.bot, original, opts)
	chunks := slices.Collect(slices.Chunk([]rune(msg.Content), 2000))

	if len(chunks) == 0 {
		chunks = [][]rune{nil}
	}

	ids := make([]string, 0, len(chunks))

	for idx, chunk := range chunks {
		if idx != 0 {
			msg.Files = nil
			msg.Embeds = nil
		}

		msg.Content = string(chunk)

		var res *message

		var err error

		if opts.ChannelData == nil {
			res, err = p.bot.sendMessage(original.ChannelID, &msg.messageSend)
		} else {
			res, err = p.bot.sendWebhook(opts.ChannelData["id"], opts.ChannelData["token"], msg.toWebhook())
		}

		if err != nil {
			return ids, err
		}

		ids = append(ids, string(res.ID))
	}

	return ids, nil
}

func (p *discordPlugin) EditMessage(
	original *lightning.Message, ids []string, opts *lightning.SendOptions,
) ([]string, error) {
	if opts.CommandResponse {
		channel, err := p.bot.startDM(opts.CommandUser)
		if err != nil {
			return nil, err
		}

		original.RepliedTo = nil
		original.ChannelID = string(channel.ID)
	}

	original.Attachments = nil
	msg := lightningToDiscordSendable(p.bot, original, opts)
	chunks := slices.Collect(slices.Chunk([]rune(msg.Content), 2000))

	if len(chunks) == 0 {
		chunks = [][]rune{nil}
	}

	for idx, chunk := range chunks {
		if idx != 0 {
			msg.Embeds = nil
		}

		msg.Content = string(chunk)

		var err error

		if opts.ChannelData == nil {
			_, err = p.bot.editMessage(original.ChannelID, ids[idx], msg.toEdit())
		} else {
			_, err = p.bot.editWebhook(
				opts.ChannelData["id"], opts.ChannelData["token"], ids[idx],
				msg.toWebhookEdit(),
			)
		}

		if err != nil {
			return ids, err
		}
	}

	return ids, nil
}

func (p *discordPlugin) DeleteMessage(channel string, ids []string) error {
	if len(ids) == 1 {
		return p.bot.deleteMessage(channel, ids[0])
	}

	return p.bot.bulkDelete(channel, ids)
}

func (p *discordPlugin) SetupCommands(command map[string]lightning.Command) {
	err := p.bot.overwriteCommands(lightningToDiscordCommands(command))
	if err != nil {
		log.Printf("%s: failed to setup commands: %v\n", p.bot.product, err)
	}
}

func (p *discordPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	addHandler(p.bot, eventMessageCreate, func(m *message) {
		if msg := discordToLightning(p.bot, m); msg != nil {
			channel <- msg
		}
	})

	return channel
}

func (p *discordPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	addHandler(p.bot, eventMessageEdit, func(evt *message) {
		if msg := discordToLightning(p.bot, evt); msg != nil {
			if evt.EditedTimestamp == nil {
				now := time.Now()
				evt.EditedTimestamp = &now
			}

			channel <- &lightning.EditedMessage{Message: msg, Edited: *evt.EditedTimestamp}
		}
	})

	return channel
}

func (p *discordPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	addHandler(p.bot, eventMessageDelete, func(m *messageDelete) {
		channel <- &lightning.BaseMessage{
			EventID: string(m.ID), ChannelID: string(m.ChannelID), Time: time.Now(),
		}
	})

	return channel
}

func (p *discordPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	channel := make(chan *lightning.CommandEvent, 1000)

	addHandler(p.bot, eventInteractionCreate, func(m *interactionCreateEvent) {
		if cmd := discordToLightningCommand(p.bot, m); cmd != nil {
			channel <- cmd
		}
	})

	return channel
}
