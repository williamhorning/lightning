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
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/internal/cache"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Discord support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"base_url": "", // optional, allows you to specify a non-Discord API implementation
//		"cdn_url": "",  // optional, allows you to specify a non-Discord CDN implementation
//		"token": "",    // required, a string with your Discord bot token
//	}
//
// Note that you MUST enable the Message Content intent for the plugin to work.
func New(cfg map[string]string) (lightning.Plugin, error) {
	if base, ok := cfg["base_url"]; ok {
		setBaseURL(base)
	}

	if cdn, ok := cfg["cdn_url"]; ok {
		setCDNURL(cdn)
	}

	session, err := discordgo.New("Bot " + cfg["token"])
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	session.Identify.Intents |= discordgo.IntentMessageContent
	session.UserAgent += " lightning/" + lightning.VERSION

	if err = session.Open(); err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %w", err)
	}

	log.Printf("discord: ready as %s in %d servers\n", session.State.User.DisplayName(), len(session.State.Guilds))

	return &discordPlugin{session: session}, nil
}

type discordPlugin struct {
	session  *discordgo.Session
	webhooks cache.Expiring[string, bool]
}

func (p *discordPlugin) SetupChannel(channel string) (map[string]string, error) {
	wh, err := p.session.WebhookCreate(channel, channel, "")
	if err != nil {
		return nil, getError(err, "Failed to create webhook for channel")
	}

	return map[string]string{"id": wh.ID, "token": wh.Token}, nil
}

func (p *discordPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	user string,
) ([]string, error) {
	channel, err := p.session.UserChannelCreate(user)
	if err != nil {
		return nil, getError(err, "Failed to create DM channel for "+user+" in command response")
	}

	message.ChannelID = channel.ID
	message.RepliedTo = nil

	return p.SendMessage(message, opts)
}

func (p *discordPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msgs := lightningToDiscordSendable(p.session, message, opts)

	if opts != nil {
		p.webhooks.Set(opts.ChannelData["id"], true)

		res, err := p.session.WebhookExecute(
			opts.ChannelData["id"], opts.ChannelData["token"], true, msgs[0].toWebhook(),
		)
		if err != nil {
			return nil, getError(err, "Failed to send message to Discord via webhook")
		}

		return []string{res.ID}, nil
	}

	res, err := p.session.ChannelMessageSendComplex(message.ChannelID, &msgs[0].MessageSend)
	if err == nil {
		return []string{res.ID}, nil
	}

	return nil, getError(err, "Failed to send message to Discord")
}

func (p *discordPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	p.webhooks.Set(opts.ChannelData["id"], true)

	if len(ids) == 0 {
		return nil
	}

	msgs := lightningToDiscordSendable(p.session, message, opts)

	_, err := p.session.WebhookMessageEdit(opts.ChannelData["id"], opts.ChannelData["id"], ids[0],
		msgs[0].toWebhookEdit())
	if err == nil {
		return nil
	}

	err = getError(err, "Failed to edit message in Discord via webhook")
	if err != nil {
		return err
	}

	return nil
}

func (p *discordPlugin) DeleteMessage(channel string, ids []string) error {
	if err := p.session.ChannelMessagesBulkDelete(channel, ids); err != nil {
		if err = getError(err, "Failed to delete messages in Discord"); err != nil {
			return err
		}
	}

	return nil
}

func (p *discordPlugin) SetupCommands(command map[string]*lightning.Command) error {
	app, err := p.session.Application("@me")
	if err != nil {
		return getError(err, "failed to get application info for Discord commands")
	}

	_, err = p.session.ApplicationCommandBulkOverwrite(app.ID, "", lightningToDiscordCommands(command))
	if err != nil {
		return getError(err, "failed to setup Discord commands")
	}

	return nil
}

func (p *discordPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	p.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if msg := discordToLightning(&p.webhooks, p.session, m.Message); msg != nil {
			channel <- msg
		}
	})

	return channel
}

func (p *discordPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	p.session.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageUpdate) {
		if msg := discordToLightning(&p.webhooks, p.session, message.Message); msg != nil {
			if message.EditedTimestamp == nil {
				now := time.Now()
				message.EditedTimestamp = &now
			}

			channel <- &lightning.EditedMessage{Message: msg, Edited: *message.EditedTimestamp}
		}
	})

	return channel
}

func (p *discordPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	p.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageDelete) {
		channel <- &lightning.BaseMessage{EventID: m.ID, ChannelID: m.ChannelID, Time: m.Timestamp}
	})

	return channel
}

func (p *discordPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	channel := make(chan *lightning.CommandEvent, 1000)

	p.session.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		if cmd := discordToLightningCommand(s, m); cmd != nil {
			channel <- cmd
		}
	})

	return channel
}
