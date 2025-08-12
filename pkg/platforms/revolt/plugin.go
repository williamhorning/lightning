// Package revolt provides a [lightning.Plugin] implementation for Revolt.
// To use Revolt support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("revolt", revolt.New)
//
//	bot.UsePluginType("revolt", map[string]any{
//		// ...
//	})
package revolt

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Revolt support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"token": "", // a string with your Revolt bot token
//	}
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "revolt", Message: "invalid config"}
	}

	token, ok := cfg["token"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "revolt", Message: "invalid token"}
	}

	cache := newRevoltCache()
	socket := revoltNewSocketManager(token)
	plugin := &revoltPlugin{cache, nil, socket, token}
	plugin.self = plugin.getUser("@me")

	if plugin.self == nil {
		return nil, lightning.PluginConfigError{Plugin: "revolt", Message: "failed to get self user"}
	}

	socket.OnReady(func(ready *revoltEventReady) {
		plugin.setCache(ready)
		slog.Info("revolt: ready!", "username", plugin.self.Username, "servers", len(ready.Servers))
		slog.Info("revolt: invite me at https://app.revolt.chat/invite/" + plugin.self.ID)
	})

	if err := socket.Connect(); err != nil {
		slog.Error("revolt: failed to connect to socket", "error", err)

		return nil, fmt.Errorf("revolt: failed to connect to socket: %w", err)
	}

	return plugin, nil
}

type revoltPlugin struct {
	revoltCache

	self   *revoltUser
	socket *revoltSocketManager

	token string
}

func (p *revoltPlugin) SendCommandResponse(
	message lightning.Message,
	opts *lightning.SendOptions,
	user string,
) ([]string, error) {
	channel := p.getDMChannel(user)
	if channel == nil {
		return nil, revoltStatusError{"failed to get DM channel for user", 0, false}
	}

	message.ChannelID = channel.ID

	return p.SendMessage(message, opts)
}

func (p *revoltPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := getOutgoing(p.token, message, opts)
	leftoverAttachments := make([]string, 0)

	if len(msg.Attachments) > 5 {
		leftoverAttachments = msg.Attachments[5:]
		msg.Attachments = msg.Attachments[:5]
	}

	res, err := sendRevoltMessage(p.token, message.ChannelID, msg)
	if err != nil {
		return nil, getRevoltError(err, map[string]any{"msg": msg}, "Failed to send message to Revolt")
	}

	ids := []string{res}

	if len(leftoverAttachments) > 0 {
		res, err := sendRevoltMessage(p.token, message.ChannelID, revoltMessageSend{
			Attachments: leftoverAttachments,
			Masquerade:  msg.Masquerade,
			Replies:     msg.Replies,
		})
		if err != nil {
			slog.Warn("revolt: failed to send leftover attachments", "attachments", leftoverAttachments, "error", err)
		} else {
			ids = append(ids, res)
		}
	}

	return ids, nil
}

func (p *revoltPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.SendOptions) error {
	message.Attachments = nil

	err := editRevoltMessage(p.token, message.ChannelID, ids[0],
		getOutgoing(p.token, message, opts).toEdit())
	if err != nil {
		return getRevoltError(err, map[string]any{"ids": ids}, "Failed to edit message on Revolt")
	}

	return nil
}

func (p *revoltPlugin) DeleteMessage(channel string, ids []string) error {
	if err := bulkDeleteRevoltMessages(p.token, channel, revoltChannelMessageBulkDeleteData{IDs: ids}); err != nil {
		return getRevoltError(err, map[string]any{"ids": ids}, "Failed to delete messages on Revolt")
	}

	return nil
}

func (*revoltPlugin) SetupCommands(_ map[string]lightning.Command) error {
	return nil
}

func (p *revoltPlugin) ListenMessages() <-chan lightning.Message {
	channel := make(chan lightning.Message, 1000)

	p.socket.OnMessageCreated(func(m *revoltEventMessage) {
		if msg := p.getIncomingMessage(m.revoltMessage); msg != nil {
			channel <- *msg
		}
	})

	return channel
}

func (p *revoltPlugin) ListenEdits() <-chan lightning.EditedMessage {
	channel := make(chan lightning.EditedMessage, 1000)

	p.socket.OnMessageUpdated(func(m *revoltEventMessageUpdate) {
		if msg := p.getIncomingMessage(m.Data); msg != nil {
			channel <- lightning.EditedMessage{
				Message: *msg,
				Edited:  m.Data.Edited,
			}
		}
	})

	return channel
}

func (p *revoltPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	channel := make(chan lightning.BaseMessage, 1000)

	p.socket.OnMessageDeleted(func(m *revoltEventMessageDelete) {
		channel <- lightning.BaseMessage{
			EventID:   m.ID,
			ChannelID: m.Channel,
			Time:      time.Now(),
		}
	})

	return channel
}

func (*revoltPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return nil
}
