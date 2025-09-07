package matrix

import (
	"context"
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
)

func setupEvents(
	syncer *mautrix.DefaultSyncer,
	client *mautrix.Client,
	msgChannel chan<- *lightning.Message,
	editChannel chan<- *lightning.EditedMessage,
) {
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

	syncer.OnEventType(event.EventMessage, onMessageHandler(client, msgChannel, editChannel))

	go func() {
		if err := client.Sync(); err != nil {
			slog.Error("Failed to sync Matrix client", "err", err)

			return
		}
	}()
}

func onMessageHandler(
	client *mautrix.Client,
	msgChannel chan<- *lightning.Message,
	editChannel chan<- *lightning.EditedMessage,
) mautrix.EventHandler {
	return func(_ context.Context, evt *event.Event) {
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

		attachments := make([]lightning.Attachment, 0)
		content := ""
		timestamp := time.UnixMilli(evt.Timestamp)

		if msg.FileName == msg.Body {
			attachments = append(attachments, lightning.Attachment{
				Name: msg.FileName,
				URL:  msg.URL.ParseOrIgnore().String(), // TODO: mxc URI -> https URL
				Size: 0,
			})
		} else {
			content, _ = format.HTMLToMarkdownFull(nil, msg.FormattedBody)
		}

		newMessage := lightning.Message{
			BaseMessage: lightning.BaseMessage{
				Time:      &timestamp,
				EventID:   evt.ID.String(),
				ChannelID: evt.RoomID.String(),
			},
			Attachments: attachments,
			// TODO: get message author information. color is also currently just white,
			// this may be odd on revolt. there's not really a good matrix color to use here.
			Author: &lightning.MessageAuthor{
				ID:             evt.Sender.String(),
				Nickname:       "",
				Username:       "",
				ProfilePicture: nil,
				Color:          "#ffffff",
			},
			Content:   content,
			RepliedTo: replyIDs,
		}

		if msg.NewContent != nil {
			if msg.NewContent.FormattedBody == "" {
				msg.NewContent.FormattedBody = msg.NewContent.Body
			}

			newContent, _ := format.HTMLToMarkdownFull(nil, msg.NewContent.FormattedBody)
			newMessage.Content = newContent

			editChannel <- &lightning.EditedMessage{Edited: &evt.Mautrix.EditedAt, Message: &newMessage}
		} else {
			msgChannel <- &newMessage
		}
	}
}
