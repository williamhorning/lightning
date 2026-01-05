package matrix

import (
	"context"
	"log"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
)

func matrixToLightningMessage(
	ctx context.Context,
	evt *event.Event,
	client *mautrix.Client,
) *lightning.Message {
	msg := evt.Content.AsMessage()

	if string(evt.Sender) == string(client.UserID) && msg.BeeperPerMessageProfile != nil {
		return nil
	}

	if msg.FormattedBody == "" {
		msg.FormattedBody = msg.Body
	}

	attachments := make([]lightning.Attachment, 0)
	content := ""

	if msg.FileName == msg.Body {
		url := getFile(client, string(msg.URL))

		attachments = append(attachments, lightning.Attachment{
			Name: msg.FileName,
			URL:  url,
			Size: 0,
		})
	} else {
		msg.RemovePerMessageProfileFallback()

		content, _ = format.HTMLToMarkdownFull(nil, msg.FormattedBody)
	}

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			Time:      time.UnixMilli(evt.Timestamp),
			EventID:   string(evt.ID),
			ChannelID: string(evt.RoomID),
		},
		Attachments: attachments,
		Author:      matrixToLightningAuthor(ctx, client, evt, msg),
		Content:     content,
		RepliedTo:   matrixToLightningReplies(msg),
	}
}

func matrixToLightningAuthor(
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
				Username:       string(evt.Sender),
				ProfilePicture: "",
				Color:          "#ffffff",
			}
		}
	}

	var profile string

	if err == nil {
		if !defaultProfile.AvatarURL.IsEmpty() {
			profile = getFile(client, "mxc://"+defaultProfile.AvatarURL.Homeserver+"/"+defaultProfile.AvatarURL.FileID)
		}
	}

	if msg.BeeperPerMessageProfile != nil {
		if msg.BeeperPerMessageProfile.AvatarURL != nil && *msg.BeeperPerMessageProfile.AvatarURL != "" {
			profile = getFile(client, string(*msg.BeeperPerMessageProfile.AvatarURL))
		}

		if msg.BeeperPerMessageProfile.Displayname == "" {
			msg.BeeperPerMessageProfile.Displayname = defaultProfile.DisplayName
		}

		return &lightning.MessageAuthor{
			ID:             string(evt.Sender),
			Username:       msg.BeeperPerMessageProfile.Displayname,
			ProfilePicture: profile,
			Color:          "#ffffff",
		}
	}

	return &lightning.MessageAuthor{
		ID:             string(evt.Sender),
		Username:       defaultProfile.DisplayName,
		ProfilePicture: profile,
		Color:          "#ffffff",
	}
}

func matrixToLightningReplies(msg *event.MessageEventContent) []string {
	replyIDs := []string{}

	if msg.RelatesTo != nil && msg.RelatesTo.InReplyTo != nil {
		replyIDs = append(replyIDs, string(msg.RelatesTo.InReplyTo.EventID))
	}

	return replyIDs
}

func getFile(client *mautrix.Client, file string) string {
	if len(file) < 6 || file[:6] != "mxc://" {
		log.Printf("matrix: invalid MXC URL: %q\n", file)

		return ""
	}

	return client.HomeserverURL.JoinPath(
		"_matrix/media/r0/download",
		file[5:],
	).String()
}
