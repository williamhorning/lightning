package matrix

import (
	"context"
	"strings"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func (p *matrixPlugin) getOutgoing(
	msg *lightning.Message,
	ids []string,
	opts *lightning.SendOptions,
) []*event.MessageEventContent {
	for _, embed := range msg.Embeds {
		msg.Content += "\n\n" + embed.ToMarkdown()
	}

	message := format.RenderMarkdown(msg.Content, true, false)

	var url *id.ContentURIString

	if msg.Author.ProfilePicture != nil {
		url = p.uploadFile(*msg.Author.ProfilePicture)
	}

	message.BeeperPerMessageProfile = &event.BeeperPerMessageProfile{
		ID:          msg.Author.ID,
		Displayname: msg.Author.Nickname,
		AvatarURL:   url,
		HasFallback: false,
	}

	if opts != nil && !opts.AllowEveryonePings {
		message.Body = strings.ReplaceAll(message.Body, "@room", "@\u200Broom")
		message.FormattedBody = strings.ReplaceAll(message.FormattedBody, "@room", "@\u200Broom")
	}

	message.AddPerMessageProfileFallback()

	if len(msg.Attachments) == 0 || len(ids) != 0 {
		return []*event.MessageEventContent{&message}
	}

	messages := make([]*event.MessageEventContent, 0, len(msg.Attachments)+1)

	for _, attachment := range msg.Attachments {
		if mxc := p.uploadFile(attachment.URL); mxc != nil {
			messages = append(messages, &event.MessageEventContent{
				MsgType: event.MsgFile,
				URL:     *mxc,
			})
		}
	}

	return messages
}

func (p *matrixPlugin) uploadFile(url string) *id.ContentURIString {
	if cached, ok := p.mxcCache.Get(url); ok {
		return &cached
	}

	resp, err := p.client.UploadLink(context.Background(), url)
	if err == nil {
		curl := resp.ContentURI.CUString()

		return &curl
	}

	return nil
}
