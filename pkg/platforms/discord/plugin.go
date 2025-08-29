// Package discord provides a [lightning.Plugin] implementation for Discord.
// To use Discord support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("discord", discord.New)
//
//	bot.UsePluginType("discord", "", map[string]any{
//		// ...
//	})
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/internal/cache"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Discord support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"token": "", // a string with your Discord bot token
//	}
//
// Note that you MUST enable the Message Content intent for the plugin to work.
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "discord", Message: "invalid config"}
	}

	token, ok := cfg["token"].(string)
	if !ok || token == "" {
		return nil, lightning.PluginConfigError{Plugin: "discord", Message: "invalid token"}
	}

	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord: failed to create session: %w", err)
	}

	discord.Identify.Intents = 16813601
	discord.StateEnabled = true
	discord.ShouldReconnectOnError = true
	discord.LogLevel = 1
	discord.UserAgent = "lightning/" + lightning.VERSION + " DiscordGo/" + discordgo.VERSION
	discordgo.Logger = func(msgL, _ int, format string, args ...any) {
		slog.Log(context.Background(), slog.Level(msgL), "discordgo: "+format, "args", args)
	}

	if err = discord.Open(); err != nil {
		return nil, fmt.Errorf("discord: failed to open session: %w", err)
	}

	app, err := discord.Application("@me")
	if err != nil {
		return nil, fmt.Errorf("discord: failed to get application info: %w", err)
	}

	slog.Info("discord: ready!",
		"username", app.Name,
		"servers", len(discord.State.Guilds),
		"invite", "https://discord.com/oauth2/authorize?client_id="+app.ID+"&scope=bot&permissions=8",
	)

	return &discordPlugin{cfg, discord, cache.New[string, bool](cache.DefaultTTL)}, nil
}

type discordPlugin struct {
	config       map[string]any
	discord      *discordgo.Session
	webhookCache *cache.Expiring[string, bool]
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

		res, err := p.discord.WebhookExecute(webhook.ID, webhook.Token, true, msg.Webhook())
		if err != nil {
			return nil, getError(err, map[string]any{"msg": msg}, "Failed to send message to Discord via webhook")
		}

		return []string{res.ID}, nil
	}

	res, err := p.discord.ChannelMessageSendComplex(message.ChannelID, msg.Message())
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

	_, err = p.discord.WebhookMessageEdit(
		webhook.ID,
		webhook.Token,
		ids[0],
		getOutgoingMessage(p.discord, message, opts).WebhookEdit(),
	)
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
