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
	"net/http"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/internal/cache"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
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
	transport := http.DefaultTransport

	if _, ok := cfg["api_host"]; ok {
		transport = &rewriteTransport{apiHost: cfg["api_host"], cdnHost: cfg["cdn_host"]}
	} else {
		cfg["cdn_host"] = "cdn.discordapp.com"
	}

	session, err := discordgo.New("Bot " + cfg["token"])
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	session.Client.Transport = transport
	session.Identify.Intents |= discordgo.IntentMessageContent
	session.UserAgent += " lightning/0.8.0-rc.9"

	if err = session.Open(); err != nil {
		return nil, fmt.Errorf("failed to open Discord session: %w", err)
	}

	invite := "https://discord.com/oauth2/authorize?client_id=" + session.State.Application.ID + "&permissions=8"

	if transport != http.DefaultTransport {
		invite = strings.ReplaceAll(invite, "discord.com", "fermi.chat")
	}

	log.Printf("discord: ready as %s in %d servers! %s\n", session.State.User.Username, len(session.State.Guilds),
		invite)

	return &discordPlugin{session: session, cdnHost: cfg["cdn_host"]}, nil
}

type discordPlugin struct {
	session  *discordgo.Session
	webhooks cache.Expiring[string, bool]
	cdnHost  string
}

func (p *discordPlugin) SetupChannel(channel string) (map[string]string, error) {
	wh, err := p.session.WebhookCreate(channel, channel, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook for channel: %w", err)
	}

	return map[string]string{"id": wh.ID, "token": wh.Token}, nil
}

func (p *discordPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	if opts.CommandResponse {
		channel, err := p.session.UserChannelCreate(opts.CommandUser)
		if err != nil {
			return nil, getError(err, "Failed to create DM channel for "+opts.CommandUser+" in command response")
		}

		message.ChannelID = channel.ID
		message.RepliedTo = nil
	}

	msg := lightningToDiscordSendable(p.session, message, opts)

	defer func() {
		for _, cancel := range msg.cancels {
			cancel()
		}
	}()

	if opts.ChannelData != nil {
		p.webhooks.Set(opts.ChannelData["id"], true)

		res, err := p.session.WebhookExecute(
			opts.ChannelData["id"], opts.ChannelData["token"], true, msg.toWebhook(),
		)
		if err != nil {
			return nil, getError(err, "send (via webhook)")
		}

		return []string{res.ID}, nil
	}

	res, err := p.session.ChannelMessageSendComplex(message.ChannelID, &msg.MessageSend)
	if err == nil {
		return []string{res.ID}, nil
	}

	return nil, getError(err, "send")
}

func (p *discordPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	if opts.ChannelData == nil || len(ids) == 0 {
		return nil
	}

	p.webhooks.Set(opts.ChannelData["id"], true)
	msg := lightningToDiscordSendable(p.session, message, opts)

	defer func() {
		for _, cancel := range msg.cancels {
			cancel()
		}
	}()

	_, err := p.session.WebhookMessageEdit(opts.ChannelData["id"], opts.ChannelData["id"], ids[0],
		msg.toWebhookEdit())
	if err == nil {
		return nil
	}

	return getError(err, "edit (via webhook)")
}

func (p *discordPlugin) DeleteMessage(channel string, ids []string) error {
	return getError(p.session.ChannelMessagesBulkDelete(channel, ids), "Failed to delete messages")
}

func (p *discordPlugin) SetupCommands(command map[string]*lightning.Command) {
	_, err := p.session.ApplicationCommandBulkOverwrite(p.session.State.Application.ID, "",
		lightningToDiscordCommands(command))
	if err != nil {
		log.Printf("discord: failed to setup commands: %v\n", err)
	}
}

func (p *discordPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	p.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if msg := discordToLightning(&p.webhooks, p.session, m.Message, p.cdnHost); msg != nil {
			channel <- msg
		}
	})

	return channel
}

func (p *discordPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	p.session.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageUpdate) {
		if msg := discordToLightning(&p.webhooks, p.session, message.Message, p.cdnHost); msg != nil {
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
