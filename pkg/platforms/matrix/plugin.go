// Package matrix provides a [lightning.Plugin] implementation for Matrix.
// To use Matrix support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("matrix", matrix.New)
//
//	bot.UsePluginType("matrix", map[string]any{
//		// ...
//	})
package matrix

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

// New creates a new [lightning.Plugin] that provides Matrix support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"access_token": "", // a string with your Matrix bot token
//		"homeserver": "",  // a string with your Matrix homeserver URL
//		"mxid": "",        // a string with your Matrix bot's user ID
//	}
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.LogError(lightning.PluginConfigError{}, "Invalid config for Matrix plugin", nil, nil)
	}

	accessToken, ok := cfg["access_token"].(string)
	if !ok {
		return nil, lightning.LogError(lightning.PluginConfigError{}, "Invalid token for Matrix plugin", nil, nil)
	}

	homeserver, ok := cfg["homeserver"].(string)
	if !ok {
		return nil, lightning.LogError(lightning.PluginConfigError{}, "Invalid homeserver for Matrix plugin", nil, nil)
	}

	mxid, ok := cfg["mxid"].(string)
	if !ok {
		return nil, lightning.LogError(lightning.PluginConfigError{}, "Invalid token for Matrix plugin", nil, nil)
	}

	client, err := mautrix.NewClient(homeserver, id.UserID(mxid), accessToken)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to create Matrix client", nil, nil)
	}

	client.UserAgent = "lightning/" + lightning.VERSION

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		return nil, lightning.LogError(lightning.PluginConfigError{}, "Client does not use DefaultSyncer", nil, nil)
	}

	syncer.OnSync(func(ctx context.Context, resp *mautrix.RespSync, since string) bool {
		if since != "" {
			return true
		}

		return client.DontProcessOldEvents(ctx, resp, since)
	})

	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := client.JoinRoomByID(ctx, evt.RoomID)
			if err != nil {
				slog.Warn("failed to join room", "err", err)
			}
		}
	})

	msgChannel := make(chan lightning.Message, 1000)
	editChannel := make(chan lightning.EditedMessage, 1000)

	syncer.OnEventType(event.EventMessage, func(_ context.Context, evt *event.Event) {
		msg := evt.Content.AsMessage()

		if evt.Sender.String() == client.UserID.String() && msg.BeeperPerMessageProfile != nil {
			return
		}

		replyIDs := []string{}

		if msg.RelatesTo != nil && msg.RelatesTo.InReplyTo != nil {
			replyIDs = append(replyIDs, msg.RelatesTo.InReplyTo.EventID.String())
		}

		if msg.FormattedBody == "" {
			msg.FormattedBody = msg.Body // fallback to plain text body if no formatted body
		}

		content, _ := format.HTMLToMarkdownFull(nil, msg.FormattedBody)

		newMessage := lightning.Message{
			BaseMessage: lightning.BaseMessage{
				Time:      time.UnixMilli(evt.Timestamp),
				EventID:   evt.ID.String(),
				ChannelID: evt.RoomID.String(),
			},
			Attachments: nil, // TODO: how on earth
			Author:      lightning.MessageAuthor{},
			Content:     content,
			RepliedTo:   replyIDs,
		}

		if msg.NewContent != nil {
			if msg.NewContent.FormattedBody == "" {
				msg.NewContent.FormattedBody = msg.NewContent.Body
			}

			newContent, _ := format.HTMLToMarkdownFull(nil, msg.NewContent.FormattedBody)
			newMessage.Content = newContent

			editChannel <- lightning.EditedMessage{Edited: evt.Mautrix.EditedAt, Message: newMessage}
		} else {
			msgChannel <- newMessage
		}
	})

	go func() {
		if err := client.Sync(); err != nil {
			slog.Error("Failed to sync Matrix client", "err", err)

			return
		}
	}()

	return &matrixPlugin{client, syncer, msgChannel, editChannel}, nil
}

type matrixPlugin struct {
	client *mautrix.Client
	syncer *mautrix.DefaultSyncer

	msgChannel  chan lightning.Message
	editChannel chan lightning.EditedMessage
}

func (*matrixPlugin) SetupChannel(_ string) (any, error) {
	// TODO: permission check?
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *matrixPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := format.RenderMarkdown(message.Content, true, false)

	msg.BeeperPerMessageProfile = &event.BeeperPerMessageProfile{
		ID:          message.Author.ID,
		Displayname: message.Author.Nickname,
		// TODO: avatar URL and cache
		HasFallback: false,
	}

	resp, err := p.client.SendMessageEvent(
		context.Background(),
		id.RoomID(message.ChannelID),
		event.EventMessage,
		msg,
		mautrix.ReqSendEvent{},
	)
	if err != nil {
		// TODO: look at permissions with HTTPError
		return nil, lightning.LogError(err, "Failed to send Matrix message",
			map[string]any{"channel": message.ChannelID, "content": message.Content}, nil)
	}

	return []string{resp.EventID.String()}, nil
}

func (p *matrixPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.SendOptions) error {
	fmt.Printf("%+v\n%+v\n", message, opts)
	for _, msgID := range ids {
		println(msgID)
	}
	return nil
}

func (p *matrixPlugin) DeleteMessage(channel string, ids []string) error {
	for _, msgID := range ids {
		if _, err := p.client.RedactEvent(
			context.Background(), id.RoomID(channel), id.EventID(msgID), mautrix.ReqRedact{Reason: "deleted in bridge"},
		); err != nil {
			// TODO: look at permissions with HTTPError
			return lightning.LogError(err, "Failed to redact Matrix message",
				map[string]any{"channel": channel, "message_id": msgID}, nil)
		}
	}

	return nil
}

func (*matrixPlugin) SetupCommands(_ map[string]lightning.Command) error {
	return nil
}

func (p *matrixPlugin) ListenMessages() <-chan lightning.Message {
	return p.msgChannel
}

func (p *matrixPlugin) ListenEdits() <-chan lightning.EditedMessage {
	return p.editChannel
}

func (p *matrixPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	channel := make(chan lightning.BaseMessage, 1000)

	p.syncer.OnEventType(event.EventRedaction, func(_ context.Context, evt *event.Event) {
		channel <- lightning.BaseMessage{
			Time:      time.UnixMilli(evt.Timestamp),
			EventID:   evt.Content.AsRedaction().Redacts.String(),
			ChannelID: evt.RoomID.String(),
		}
	})

	return channel
}

func (*matrixPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return nil
}
