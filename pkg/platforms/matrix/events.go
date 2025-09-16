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
	return func(ctx context.Context, evt *event.Event) {
		msg := evt.Content.AsMessage()

		if evt.Sender.String() == client.UserID.String() && msg.BeeperPerMessageProfile != nil {
			return
		}

		if msg.FormattedBody == "" {
			msg.FormattedBody = msg.Body
		}

		attachments := make([]lightning.Attachment, 0)
		content := ""
		timestamp := time.UnixMilli(evt.Timestamp)

		if msg.FileName == msg.Body {
			url := getMXC(client, &msg.URL)

			attachments = append(attachments, lightning.Attachment{
				Name: msg.FileName,
				URL:  url,
				Size: 0,
			})
		} else {
			msg.RemovePerMessageProfileFallback()

			content, _ = format.HTMLToMarkdownFull(nil, msg.FormattedBody)
		}

		newMessage := lightning.Message{
			BaseMessage: lightning.BaseMessage{
				Time:      &timestamp,
				EventID:   evt.ID.String(),
				ChannelID: evt.RoomID.String(),
			},
			Attachments: attachments,
			Author:      getAuthor(ctx, client, evt, msg),
			Content:     content,
			RepliedTo:   getRepliedTo(msg),
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

func getAuthor(
	ctx context.Context,
	client *mautrix.Client,
	evt *event.Event,
	msg *event.MessageEventContent,
) *lightning.MessageAuthor {
	defaultProfile, err := client.GetProfile(ctx, evt.Sender)
	if err != nil {
		slog.Error(fmt.Errorf("matrix: failed to get default profile on message: %w", err).Error())

		if msg.BeeperPerMessageProfile == nil {
			return &lightning.MessageAuthor{
				ID:             evt.Sender.String(),
				Nickname:       evt.Sender.String(),
				Username:       evt.Sender.String(),
				ProfilePicture: nil,
				Color:          "#ffffff",
			}
		}
	}

	var defaultPic *string

	if err == nil {
		if !defaultProfile.AvatarURL.IsEmpty() {
			cu := defaultProfile.AvatarURL.CUString()
			url := getMXC(client, &cu)
			defaultPic = &url
		}
	}

	if msg.BeeperPerMessageProfile != nil {
		var profile *string

		if msg.BeeperPerMessageProfile.AvatarURL != nil && *msg.BeeperPerMessageProfile.AvatarURL != "" {
			url := getMXC(client, msg.BeeperPerMessageProfile.AvatarURL)
			profile = &url
		} else if *msg.BeeperPerMessageProfile.AvatarURL == "" && !defaultProfile.AvatarURL.IsEmpty() {
			profile = defaultPic
		}

		return &lightning.MessageAuthor{
			ID:             evt.Sender.String(),
			Nickname:       msg.BeeperPerMessageProfile.Displayname,
			Username:       defaultProfile.DisplayName,
			ProfilePicture: profile,
			Color:          "#ffffff",
		}
	}

	return &lightning.MessageAuthor{
		ID:             evt.Sender.String(),
		Nickname:       defaultProfile.DisplayName,
		Username:       defaultProfile.DisplayName,
		ProfilePicture: defaultPic,
		Color:          "#ffffff",
	}
}

func getRepliedTo(msg *event.MessageEventContent) []string {
	replyIDs := []string{}

	if msg.RelatesTo != nil && msg.RelatesTo.InReplyTo != nil {
		replyIDs = append(replyIDs, msg.RelatesTo.InReplyTo.EventID.String())
	}

	return replyIDs
}

func getMXC(client *mautrix.Client, file *id.ContentURIString) string {
	return client.HomeserverURL.JoinPath(
		"_matrix/media/r0/download",
		file.ParseOrIgnore().Homeserver,
		file.ParseOrIgnore().FileID,
	).String()
}
