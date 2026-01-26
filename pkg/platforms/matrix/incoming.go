package matrix

import (
	"context"
	"regexp"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
)

func (p *matrixPlugin) matrixToLightningMessage(
	ctx context.Context,
	evt *event.Event,
	client *mautrix.Client,
	regex string,
) *lightning.Message {
	if evt.Type != event.EventMessage {
		return nil
	}

	msg := evt.Content.AsMessage()

	if msg.BeeperPerMessageProfile != nil && regexp.MustCompile(regex).Match([]byte(evt.Sender)) {
		return nil
	}

	if msg.FormattedBody == "" {
		msg.FormattedBody = msg.Body
	}

	attachments := make([]lightning.Attachment, 0)
	content := ""

	if msg.FileName == msg.Body {
		url := p.getFile(string(msg.URL))

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
		Author:      p.matrixToLightningAuthor(ctx, client, evt, msg),
		Content:     content,
		RepliedTo:   matrixToLightningReplies(msg),
	}
}

func (p *matrixPlugin) matrixToLightningAuthor(
	ctx context.Context,
	client *mautrix.Client,
	evt *event.Event,
	msg *event.MessageEventContent,
) *lightning.MessageAuthor {
	author := &lightning.MessageAuthor{
		ID:             string(evt.Sender),
		Username:       string(evt.Sender),
		ProfilePicture: "",
		Color:          "#ffffff",
	}

	globalProfile, err := client.GetProfile(ctx, evt.Sender)
	if err == nil && globalProfile != nil {
		author.Username = globalProfile.DisplayName
		if !globalProfile.AvatarURL.IsEmpty() {
			author.ProfilePicture = p.getFile(globalProfile.AvatarURL.String())
		}
	}

	if msg.BeeperPerMessageProfile != nil {
		if msg.BeeperPerMessageProfile.Displayname != "" {
			author.Username = msg.BeeperPerMessageProfile.Displayname
		}

		if msg.BeeperPerMessageProfile.AvatarURL != nil && *msg.BeeperPerMessageProfile.AvatarURL != "" {
			author.ProfilePicture = p.getFile(string(*msg.BeeperPerMessageProfile.AvatarURL))
		}
	}

	return author
}

func matrixToLightningReplies(msg *event.MessageEventContent) []string {
	replyIDs := []string{}

	if msg.RelatesTo != nil && msg.RelatesTo.InReplyTo != nil {
		replyIDs = append(replyIDs, string(msg.RelatesTo.InReplyTo.EventID))
	}

	return replyIDs
}

func (p *matrixPlugin) getFile(file string) string {
	return p.proxy + "/" + file
}
