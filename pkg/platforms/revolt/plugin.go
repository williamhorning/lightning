package revolt

import (
	"errors"
	"time"

	"github.com/sentinelb51/revoltgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

var revoltLog = lightning.Log.With("plugin", "revolt")

func init() {
	lightning.Plugins.RegisterType("revolt", newRevoltPlugin)
}

func newRevoltPlugin(config any) (lightning.Plugin, error) {
	if cfg, ok := config.(map[string]any); !ok {
		return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Invalid config for Revolt plugin", nil, nil)
	} else {
		revolt := revoltgo.New(cfg["token"].(string))

		if err := revolt.Open(); err != nil {
			return nil, lightning.LogError(err, "Failed to open Revolt session", nil, nil)
		}

		revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventReady) {
			revoltLog.Info("ready!", "username", s.State.Self().Username, "servers", len(m.Servers))
			revoltLog.Info("Invite me at https://revolt.chat/invite/" + s.State.Self().ID)
		})

		revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventError) {
			revoltLog.Error("socket error", "error", m.Error)
		})

		revolt.ReconnectInterval = 100 * time.Millisecond

		return &revoltPlugin{cfg, revolt}, nil
	}
}

type revoltPlugin struct {
	config map[string]any
	revolt *revoltgo.Session
}

func (p *revoltPlugin) Name() string {
	return "bolt-revolt"
}

const correctPermissionValue = uint(485495808)

func (p *revoltPlugin) SetupChannel(channel string) (any, error) {
	permissions, err := p.revolt.State.ChannelPermissions(p.revolt.State.Self(), p.revolt.State.Channel(channel))

	if err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to get channel permissions in Revolt",
			map[string]any{"channel": channel},
			nil,
		)
	}

	revoltLog.Debug("revolt permissions", "channel", channel, "permissions", permissions)

	if (permissions & correctPermissionValue) != correctPermissionValue {
		return nil, lightning.LogError(
			errors.New("insufficient permissions in Revolt channel"),
			"Missing required permissions. Please add all permissions to a role, assign that role to the bot, and rejoin the bridge",
			map[string]any{
				"channel":              channel,
				"current_permissions":  permissions,
				"expected_permissions": correctPermissionValue,
			},
			nil,
		)
	}

	return channel, nil
}

func (p *revoltPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := getOutgoingMessage(p.revolt, message, false, opts != nil)
	leftoverAttachments := make([]string, 0)

	if len(msg.Attachments) > 5 {
		leftoverAttachments = msg.Attachments[5:]
		msg.Attachments = msg.Attachments[:5]
	}

	res, err := p.revolt.ChannelMessageSend(message.ChannelID, msg)

	if err != nil {
		return nil, getRevoltError(err, map[string]any{"msg": msg}, "Failed to send message to Revolt", false)
	}

	ids := []string{res.ID}

	if len(leftoverAttachments) > 0 {
		res, err := p.revolt.ChannelMessageSend(message.ChannelID, revoltgo.MessageSend{
			Attachments: leftoverAttachments,
			Content:     "",
			Masquerade:  msg.Masquerade,
			Replies:     msg.Replies,
		})

		if err != nil {
			revoltLog.Warn("failed to send leftover attachments", "attachments", leftoverAttachments, "error", err)
		} else {
			ids = append(ids, res.ID)
		}
	}

	return ids, nil
}

func (p *revoltPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.SendOptions) error {
	_, err := p.revolt.ChannelMessageEdit(opts.ChannelID, ids[0], toEdit(getOutgoingMessage(p.revolt, message, true, true)))

	if err != nil {
		return getRevoltError(err, map[string]any{"ids": ids}, "Failed to edit message on Revolt", true)
	}

	return nil
}

func (p *revoltPlugin) DeleteMessage(ids []string, opts *lightning.SendOptions) error {
	for _, id := range ids {
		err := p.revolt.ChannelMessageDelete(opts.ChannelID, id)

		if err != nil {
			return getRevoltError(err, map[string]any{"ids": ids}, "Failed to delete message on Revolt", true)
		}
	}

	return nil
}

func (p *revoltPlugin) SetupCommands(command map[string]lightning.Command) error {
	return nil
}

func (p *revoltPlugin) ListenMessages() <-chan lightning.Message {
	ch := make(chan lightning.Message, 1000)

	p.revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventMessage) {
		if msg := getLightningMessage(s, m.Message); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *revoltPlugin) ListenEdits() <-chan lightning.Message {
	ch := make(chan lightning.Message, 1000)

	p.revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventMessageUpdate) {
		if msg := getLightningMessage(s, m.Data); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *revoltPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	ch := make(chan lightning.BaseMessage, 1000)

	p.revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventMessageDelete) {
		ch <- lightning.BaseMessage{
			EventID:   m.ID,
			ChannelID: m.Channel,
			Plugin:    p.Name(),
			Time:      time.Now(),
		}
	})

	return ch
}

func (p *revoltPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return make(chan lightning.CommandEvent)
}
