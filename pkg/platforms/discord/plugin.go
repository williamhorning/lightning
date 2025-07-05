package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func init() {
	lightning.Plugins.RegisterType("discord", newDiscordPlugin)
}

func newDiscordPlugin(config any) (lightning.Plugin, error) {
	if cfg, ok := config.(map[string]any); !ok {
		return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Invalid config for Discord plugin", nil, nil)
	} else {
		token, ok := cfg["token"].(string)
		if !ok || token == "" {
			return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Missing or invalid token in Discord plugin config", nil, nil)
		}

		discord, err := discordgo.New("Bot " + token)

		discord.Identify.Intents = 16813601
		discord.StateEnabled = true
		discord.ShouldReconnectOnError = true
		discord.LogLevel = 1
		discordgo.Logger = func(msgL, caller int, format string, a ...any) {
			level := log.DebugLevel

			switch msgL {
			case 0:
				level = log.ErrorLevel
			case 1:
				level = log.InfoLevel
			}

			lightning.Log.With("plugin", "discord").Logf(level, format, a...)
		}

		if err != nil {
			return nil, lightning.LogError(err, "Failed to create Discord session", nil, nil)
		}

		if err = discord.Open(); err != nil {
			return nil, lightning.LogError(err, "Failed to open Discord session", nil, nil)
		}

		app, _ := discord.Application("@me")
		lightning.Log.Info("ready!", "plugin", "discord", "username", app.Name, "servers", len(discord.State.Guilds), "invite", "https://discord.com/oauth2/authorize?client_id="+app.ID+"&scope=bot&permissions=8")

		return &discordPlugin{cfg, discord}, nil
	}
}

type discordPlugin struct {
	config  map[string]any
	discord *discordgo.Session
}

func (p *discordPlugin) Name() string {
	return "bolt-discord"
}

func (p *discordPlugin) SetupChannel(channel string) (any, error) {
	wh, err := p.discord.WebhookCreate(channel, "Lightning Bridge", "")

	if err != nil {
		return nil, getError(err, map[string]any{"channel": channel}, "Failed to create webhook for channel")
	}

	return map[string]string{"id": wh.ID, "token": wh.Token}, nil
}

func (p *discordPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := getOutgoingMessage(p.discord, message, opts, opts != nil)

	if opts != nil {
		id, token, err := getWebhookFromChannel(opts)

		if err != nil {
			return nil, err
		}

		res, err := p.discord.WebhookExecute(id, token, true, msg.Webhook())

		if err != nil {
			return nil, getError(err, map[string]any{"msg": msg}, "Failed to send message to Discord via webhook")
		}

		return []string{res.ID}, nil
	} else {
		if res, err := p.discord.ChannelMessageSendComplex(message.ChannelID, msg.Message()); err == nil {
			return []string{res.ID}, nil
		} else {
			return nil, getError(err, map[string]any{"msg": msg}, "Failed to send message to Discord")
		}
	}
}

func (p *discordPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.SendOptions) error {
	id, token, err := getWebhookFromChannel(opts)

	if err != nil {
		return err
	}

	if _, err = p.discord.WebhookMessageEdit(id, token, ids[0], getOutgoingMessage(p.discord, message, opts, true).WebhookEdit()); err == nil {
		return nil
	} else if err = getError(err, map[string]any{"ids": ids, "msg": message}, "Failed to edit message in Discord via webhook"); err != nil {
		return err
	} else {
		return nil
	}
}

func (p *discordPlugin) DeleteMessage(ids []string, opts *lightning.SendOptions) error {
	if err := p.discord.ChannelMessagesBulkDelete(opts.ChannelID, ids); err != nil {
		if err = getError(err, map[string]any{"ids": ids}, "Failed to delete messages in Discord"); err != nil {
			return err
		}
	}

	return nil
}

func (p *discordPlugin) SetupCommands(command map[string]lightning.Command) error {
	if p.config["slash_commands"] != true {
		return nil
	}

	app, err := p.discord.Application("@me")

	if err != nil {
		return lightning.LogError(err, "Failed to get application info for Discord commands", nil, nil)
	}

	_, err = p.discord.ApplicationCommandBulkOverwrite(app.ID, "", getDiscordCommand(command))

	if err != nil {
		return lightning.LogError(err, "Failed to setup commands in Discord", map[string]any{"commands": command}, nil)
	}

	return nil
}

func (p *discordPlugin) ListenMessages() <-chan lightning.Message {
	ch := make(chan lightning.Message, 1000)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if msg := getLightningMessage(s, m.Message); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *discordPlugin) ListenEdits() <-chan lightning.Message {
	ch := make(chan lightning.Message, 1000)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		if msg := getLightningMessage(s, m.Message); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *discordPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	ch := make(chan lightning.BaseMessage, 1000)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageDelete) {
		ch <- lightning.BaseMessage{
			EventID:   m.ID,
			ChannelID: m.ChannelID,
			Plugin:    p.Name(),
			Time:      m.Timestamp,
		}
	})

	return ch
}

func (p *discordPlugin) ListenCommands() <-chan lightning.CommandEvent {
	ch := make(chan lightning.CommandEvent, 1000)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		cmd := getLightningCommand(s, m)

		if cmd != nil {
			ch <- *cmd
		}
	})

	return ch
}
