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
//		"token": "", // a string with your Discord bot token
//	}
//
// Note that you MUST enable the Message Content intent for the plugin to work.
func New(cfg map[string]string) (lightning.Plugin, error) {
	discord, err := discordgo.New("Bot " + cfg["token"])
	if err != nil {
		return nil, fmt.Errorf("discord: failed to create session: %w", err)
	}

	discord.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMessages |
		discordgo.IntentDirectMessages | discordgo.IntentMessageContent | discordgo.IntentGuildMessagePolls
	discord.StateEnabled = true
	discord.ShouldReconnectOnError = true
	discord.LogLevel = discordgo.LogError
	discord.UserAgent = "lightning/" + lightning.VERSION + " DiscordGo/" + discordgo.VERSION

	if err = discord.Open(); err != nil {
		return nil, fmt.Errorf("discord: failed to open session: %w", err)
	}

	app, err := discord.Application("@me")
	if err != nil {
		return nil, fmt.Errorf("discord: failed to get application info: %w", err)
	}

	log.Printf("discord: ready as %s in %d servers\n", app.Name, len(discord.State.Guilds))
	log.Printf("discord: https://discord.com/oauth2/authorize?client_id=%s&scope=bot&permissions=8\n", app.ID)

	return &discordPlugin{discord: discord}, nil
}

type discordPlugin struct {
	discord      *discordgo.Session
	webhookCache cache.Expiring[string, bool]
}

func (p *discordPlugin) SetupChannel(channel string) (any, error) {
	wh, err := p.discord.WebhookCreate(channel, channel, "")
	if err != nil {
		return nil, getError(err, map[string]any{"channel": channel}, "Failed to create webhook for channel")
	}

	return map[string]string{"id": wh.ID, "token": wh.Token}, nil
}

func (p *discordPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	user string,
) ([]string, error) {
	channel, err := p.discord.UserChannelCreate(user)
	if err != nil {
		return nil, getError(err, map[string]any{"user": user}, "Failed to create DM channel for command response")
	}

	message.ChannelID = channel.ID

	return p.SendMessage(message, opts)
}

func (p *discordPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := getOutgoingMessage(p.discord, message, opts)

	if opts != nil {
		webhook, err := p.getWebhookFromChannel(message.ChannelID, opts)
		if err != nil {
			return nil, err
		}

		res, err := p.discord.WebhookExecute(webhook.ID, webhook.Token, true, &discordgo.WebhookParams{
			AllowedMentions: msg.allowedMentions, AvatarURL: msg.avatarURL, Components: msg.components,
			Content: msg.content, Embeds: msg.embeds, Files: msg.files, Username: msg.username,
		})
		if err != nil {
			return nil, getError(err, map[string]any{"msg": msg}, "Failed to send message to Discord via webhook")
		}

		return []string{res.ID}, nil
	}

	res, err := p.discord.ChannelMessageSendComplex(message.ChannelID, &discordgo.MessageSend{
		AllowedMentions: msg.allowedMentions, Components: msg.components, Content: msg.content, Embeds: msg.embeds,
		Files: msg.files, Reference: msg.reference,
	})
	if err == nil {
		return []string{res.ID}, nil
	}

	return nil, getError(err, map[string]any{"msg": msg}, "Failed to send message to Discord")
}

func (p *discordPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	webhook, err := p.getWebhookFromChannel(message.ChannelID, opts)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	msg := getOutgoingMessage(p.discord, message, opts)

	_, err = p.discord.WebhookMessageEdit(webhook.ID, webhook.Token, ids[0], &discordgo.WebhookEdit{
		AllowedMentions: msg.allowedMentions, Content: &msg.content, Components: &msg.components, Embeds: &msg.embeds,
		Files: msg.files,
	})
	if err == nil {
		return nil
	}

	err = getError(err, map[string]any{"ids": ids, "msg": message}, "Failed to edit message in Discord via webhook")
	if err != nil {
		return err
	}

	return nil
}

func (p *discordPlugin) DeleteMessage(channel string, ids []string) error {
	if err := p.discord.ChannelMessagesBulkDelete(channel, ids); err != nil {
		if err = getError(err, map[string]any{"ids": ids}, "Failed to delete messages in Discord"); err != nil {
			return err
		}
	}

	return nil
}

func (p *discordPlugin) SetupCommands(command map[string]*lightning.Command) error {
	app, err := p.discord.Application("@me")
	if err != nil {
		return getError(err, nil, "failed to get application info for Discord commands")
	}

	_, err = p.discord.ApplicationCommandBulkOverwrite(app.ID, "", getDiscordCommand(command))
	if err != nil {
		return getError(err, nil, "failed to setup Discord commands")
	}

	return nil
}

func (p *discordPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	p.discord.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if msg := p.getLightningMessage(m.Message); msg != nil {
			channel <- msg
		}
	})

	return channel
}

func (p *discordPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	p.discord.AddHandler(func(_ *discordgo.Session, message *discordgo.MessageUpdate) {
		if msg := p.getLightningMessage(message.Message); msg != nil {
			if message.EditedTimestamp == nil {
				now := time.Now()
				message.EditedTimestamp = &now
			}

			channel <- &lightning.EditedMessage{Message: msg, Edited: message.EditedTimestamp}
		}
	})

	return channel
}

func (p *discordPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	p.discord.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageDelete) {
		channel <- &lightning.BaseMessage{EventID: m.ID, ChannelID: m.ChannelID, Time: &m.Timestamp}
	})

	return channel
}

func (p *discordPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	channel := make(chan *lightning.CommandEvent, 1000)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		if cmd := getLightningCommand(s, m); cmd != nil {
			channel <- cmd
		}
	})

	return channel
}
