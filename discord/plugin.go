package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning"
)

func init() {
	lightning.RegisterPluginType("discord", newDiscordPlugin)
}

func newDiscordPlugin(config any) (lightning.Plugin, error) {
	if cfg, ok := config.(map[string]any); !ok {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Invalid config for Discord plugin",
			nil,
			lightning.ReadWriteDisabled{},
		)
	} else {
		token, ok := cfg["token"].(string)
		if !ok || token == "" {
			return nil, lightning.LogError(
				lightning.ErrPluginConfigInvalid,
				"Missing or invalid token in Discord plugin config",
				nil,
				lightning.ReadWriteDisabled{},
			)
		}

		discord, err := discordgo.New("Bot " + token)

		discord.Identify.Intents = 16813601
		discord.StateEnabled = true

		if err != nil {
			return nil, lightning.LogError(
				err,
				"Failed to create Discord session",
				nil,
				lightning.ReadWriteDisabled{},
			)
		}

		err = discord.Open()

		if err != nil {
			return nil, lightning.LogError(
				err,
				"Failed to open Discord session",
				nil,
				lightning.ReadWriteDisabled{},
			)
		}

		app, err := discord.Application("@me")
		lightning.Log.Info().Str("plugin", "discord").Str("username", discord.State.User.Username).Int("servers", len(discord.State.Guilds)).Msg("ready!")
		if err == nil {
			lightning.Log.Info().Str("plugin", "discord").Msg("invite me at https://discord.com/oauth2/authorize?client_id=" + app.ID + "%s&scope=bot&permissions=8")
		}

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

func (p *discordPlugin) SendMessage(message lightning.Message, opts *lightning.BridgeMessageOptions) ([]string, error) {
	msg := getOutgoingMessage(p.discord, message, opts, opts != nil)

	if opts != nil {
		id, token, err := getWebhookFromChannel(opts.Channel)

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

func (p *discordPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.BridgeMessageOptions) error {
	id, token, err := getWebhookFromChannel(opts.Channel)

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

func (p *discordPlugin) DeleteMessage(ids []string, opts *lightning.BridgeMessageOptions) error {
	if err := p.discord.ChannelMessagesBulkDelete(opts.Channel.ID, ids); err != nil {
		if err = getError(err, map[string]any{"ids": ids}, "Failed to delete messages in Discord"); err != nil {
			return err
		}
	}

	return nil
}

func (p *discordPlugin) SetupCommands(command []lightning.Command) error {
	if p.config["slash_commands"] != true {
		return nil
	}

	app, err := p.discord.Application("@me")

	if err != nil {
		return lightning.LogError(
			err,
			"Failed to get application info for Discord commands",
			nil,
			lightning.ReadWriteDisabled{Read: false, Write: false},
		)
	}

	_, err = p.discord.ApplicationCommandBulkOverwrite(app.ID, "", getDiscordCommand(command))

	if err != nil {
		return lightning.LogError(
			err,
			"Failed to setup commands in Discord",
			map[string]any{"commands": command},
			lightning.ReadWriteDisabled{Read: false, Write: false},
		)
	}

	return nil
}

func (p *discordPlugin) ListenMessages() <-chan lightning.Message {
	ch := make(chan lightning.Message, 100)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if msg := getLightningMessage(s, m.Message); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *discordPlugin) ListenEdits() <-chan lightning.Message {
	ch := make(chan lightning.Message, 100)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageUpdate) {
		if msg := getLightningMessage(s, m.Message); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *discordPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	ch := make(chan lightning.BaseMessage, 100)

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
	ch := make(chan lightning.CommandEvent, 100)

	p.discord.AddHandler(func(s *discordgo.Session, m *discordgo.InteractionCreate) {
		cmd := getLightningCommand(s, m)

		if cmd != nil {
			ch <- *cmd
		}
	})

	return ch
}
