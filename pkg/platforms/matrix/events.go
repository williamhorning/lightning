package matrix

import (
	"context"
	"log"
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
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := client.JoinRoomByID(ctx, evt.RoomID)
			if err != nil {
				log.Printf("matrix: failed to join room: %v\n", err)
			}
		}
	})

	syncer.OnSync(func(ctx context.Context, resp *mautrix.RespSync, since string) bool {
		if since != "" {
			return true
		}

		return client.DontProcessOldEvents(ctx, resp, since)
	})

	syncer.OnEventType(event.EventMessage, onMessageHandler(client, msgChannel, editChannel))

	go func() {
		if err := client.Sync(); err != nil {
			log.Printf("matrix: failed to sync: %v\n", err)

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

		if string(evt.Sender) == string(client.UserID) && msg.BeeperPerMessageProfile != nil {
			return
		}

		if msg.FormattedBody == "" {
			msg.FormattedBody = msg.Body
		}

		attachments := make([]lightning.Attachment, 0)
		content := ""
		timestamp := time.UnixMilli(evt.Timestamp)

		if msg.FileName == msg.Body {
			url := getMXC(client, string(msg.URL))

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
				EventID:   string(evt.ID),
				ChannelID: string(evt.RoomID),
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
		log.Printf("matrix: failed to get default message profile: %v\n", err)

		if msg.BeeperPerMessageProfile == nil {
			return &lightning.MessageAuthor{
				ID:             string(evt.Sender),
				Nickname:       string(evt.Sender),
				Username:       string(evt.Sender),
				ProfilePicture: nil,
				Color:          "#ffffff",
			}
		}
	}

	var defaultPic *string

	if err == nil {
		if !defaultProfile.AvatarURL.IsEmpty() {
			url := getMXC(client, "mxc://"+defaultProfile.AvatarURL.Homeserver+"/"+defaultProfile.AvatarURL.FileID)
			defaultPic = &url
		}
	}

	if msg.BeeperPerMessageProfile != nil {
		var profile *string

		if msg.BeeperPerMessageProfile.AvatarURL != nil && *msg.BeeperPerMessageProfile.AvatarURL != "" {
			url := getMXC(client, string(*msg.BeeperPerMessageProfile.AvatarURL))
			profile = &url
		} else if *msg.BeeperPerMessageProfile.AvatarURL == "" && !defaultProfile.AvatarURL.IsEmpty() {
			profile = defaultPic
		}

		return &lightning.MessageAuthor{
			ID:             string(evt.Sender),
			Nickname:       msg.BeeperPerMessageProfile.Displayname,
			Username:       defaultProfile.DisplayName,
			ProfilePicture: profile,
			Color:          "#ffffff",
		}
	}

	return &lightning.MessageAuthor{
		ID:             string(evt.Sender),
		Nickname:       defaultProfile.DisplayName,
		Username:       defaultProfile.DisplayName,
		ProfilePicture: defaultPic,
		Color:          "#ffffff",
	}
}

func getRepliedTo(msg *event.MessageEventContent) []string {
	replyIDs := []string{}

	if msg.RelatesTo != nil && msg.RelatesTo.InReplyTo != nil {
		replyIDs = append(replyIDs, string(msg.RelatesTo.InReplyTo.EventID))
	}

	return replyIDs
}

func getMXC(client *mautrix.Client, file string) string {
	return client.HomeserverURL.JoinPath(
		"_matrix/media/r0/download",
		file[5:],
	).String()
}
