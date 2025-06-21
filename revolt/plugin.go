package revolt

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/sentinelb51/revoltgo"
	"github.com/williamhorning/lightning"
)

func init() {
	lightning.Plugins.RegisterType("revolt", newRevoltPlugin)
}

type zerologAdapter struct{}

func (z *zerologAdapter) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	lightning.Log.Debug().
		Str("plugin", "revolt").
		Str("type", "revoltgo").
		Msg(message)
	return len(p), nil
}

func newRevoltPlugin(config any) (lightning.Plugin, error) {
	if cfg, ok := config.(map[string]any); !ok {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Invalid config for Revolt plugin",
			nil,
			lightning.ReadWriteDisabled{},
		)
	} else {
		revolt := revoltgo.New(cfg["token"].(string))

		log.SetFlags(0)
		log.SetOutput(&zerologAdapter{})

		err := revolt.Open()

		if err != nil {
			return nil, lightning.LogError(
				err,
				"Failed to open Revolt session",
				nil,
				lightning.ReadWriteDisabled{},
			)
		}

		revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventReady) {
			lightning.Log.Info().Str("plugin", "revolt").Str("username", s.State.Self().Username).Int("servers", len(m.Servers)).Msg("ready!")
			lightning.Log.Info().Str("plugin", "revolt").Msg("invite me at https://revolt.chat/invite/" + s.State.Self().ID)
		})

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

const requiredPermissions = revoltgo.PermissionChangeNickname |
	revoltgo.PermissionChangeAvatar |
	revoltgo.PermissionReadMessageHistory |
	revoltgo.PermissionSendMessage |
	revoltgo.PermissionManageMessages |
	revoltgo.PermissionSendEmbeds |
	revoltgo.PermissionUploadFiles

func (p *revoltPlugin) SetupChannel(channel string) (any, error) {
	permissions, err := p.revolt.State.ChannelPermissions(p.revolt.State.Self(), p.revolt.State.Channel(channel))

	if err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to get channel permissions in Revolt",
			map[string]any{"channel": channel},
			lightning.ReadWriteDisabled{},
		)
	}

	if (permissions & requiredPermissions) != requiredPermissions {
		return nil, lightning.LogError(
			errors.New("insufficient permissions in Revolt channel"),
			"missing ChangeNickname, ChangeAvatar, ReadMessageHistory, SendMessage, ManageMessages, SendEmbeds, UploadFiles, and/or Masquerade permissions please add them to a role, assign that role to the bot, and rejoin the bridge",
			map[string]any{"channel": channel, "permissions": permissions},
			lightning.ReadWriteDisabled{},
		)
	}

	return channel, nil
}

func (p *revoltPlugin) SendMessage(message lightning.Message, opts *lightning.BridgeMessageOptions) ([]string, error) {
	canMasquerade := opts != nil

	if opts == nil {
		chPermissions, err := p.revolt.State.ChannelPermissions(p.revolt.State.Self(), p.revolt.State.Channel(message.ChannelID))

		if err == nil {
			canMasquerade = chPermissions&revoltgo.PermissionMasquerade == revoltgo.PermissionMasquerade
		}
	}

	msg := getOutgoingMessage(p.revolt, message, false, canMasquerade)
	res, err := p.revolt.ChannelMessageSend(message.ChannelID, msg)

	if err != nil {
		return nil, getRevoltError(err, map[string]any{"msg": msg}, "Failed to send message to Revolt", false)
	}

	return []string{res.ID}, nil
}

func (p *revoltPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.BridgeMessageOptions) error {
	_, err := p.revolt.ChannelMessageEdit(opts.Channel.ID, ids[0], toEdit(getOutgoingMessage(p.revolt, message, true, false)))

	if err != nil {
		return getRevoltError(err, map[string]any{"ids": ids}, "Failed to edit message on Revolt", true)
	}

	return nil
}

func (p *revoltPlugin) DeleteMessage(ids []string, opts *lightning.BridgeMessageOptions) error {
	for _, id := range ids {
		err := p.revolt.ChannelMessageDelete(opts.Channel.ID, id)

		if err != nil {
			return getRevoltError(err, map[string]any{"ids": ids}, "Failed to delete message on Revolt", true)
		}
	}

	return nil
}

func (p *revoltPlugin) SetupCommands(command []lightning.Command) error {
	return nil
}

func (p *revoltPlugin) ListenMessages() <-chan lightning.Message {
	ch := make(chan lightning.Message, 100)

	p.revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventMessage) {
		if msg := getLightningMessage(s, m.Message); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *revoltPlugin) ListenEdits() <-chan lightning.Message {
	ch := make(chan lightning.Message, 100)

	p.revolt.AddHandler(func(s *revoltgo.Session, m *revoltgo.EventMessageUpdate) {
		if msg := getLightningMessage(s, m.Data); msg != nil {
			ch <- *msg
		}
	})

	return ch
}

func (p *revoltPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	ch := make(chan lightning.BaseMessage, 100)

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
