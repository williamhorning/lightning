package matrix

import (
	"context"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func (p *matrixPlugin) lightningToMatrixMessage(
	msg *lightning.Message, opts *lightning.SendOptions,
) []*event.MessageEventContent {
	for idx := range msg.Embeds {
		msg.Content += "\n\n" + msg.Embeds[idx].ToMarkdown()
	}

	message := format.RenderMarkdown(msg.Content, true, false)

	var url *id.ContentURIString

	if msg.Author != nil {
		if msg.Author.ProfilePicture != "" {
			url = p.uploadFile(msg.Author.ProfilePicture)
		}

		message.BeeperPerMessageProfile = &event.BeeperPerMessageProfile{
			ID:          msg.Author.ID,
			Displayname: msg.Author.Username,
			AvatarURL:   url,
		}
	}

	if !opts.AllowEveryonePings {
		message.Body = strings.ReplaceAll(message.Body, "@room", "@\u200Broom")
		message.FormattedBody = strings.ReplaceAll(message.FormattedBody, "@room", "@\u200Broom")
	}

	if len(msg.RepliedTo) != 0 {
		message.RelatesTo = &event.RelatesTo{Type: "m.in_reply_to", EventID: id.EventID(msg.RepliedTo[0])}
	}

	message.AddPerMessageProfileFallback()

	messages := make([]*event.MessageEventContent, 0, len(msg.Attachments)+1)

	for _, attachment := range msg.Attachments {
		if mxc := p.uploadFile(attachment.URL); mxc != nil {
			messages = append(messages, &event.MessageEventContent{
				RelatesTo:               message.RelatesTo,
				FileName:                attachment.Name,
				BeeperPerMessageProfile: message.BeeperPerMessageProfile,
				MsgType:                 event.MsgFile,
				URL:                     *mxc,
			})
		}
	}

	for _, mxSend := range messages {
		mxSend.AddPerMessageProfileFallback()
	}

	return messages
}

func (p *matrixPlugin) uploadFile(url string) *id.ContentURIString {
	if cached, ok := p.mxcCache.Get(url); ok {
		return &cached
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	parts := strings.Split(url, ".")

	mxc, err := p.client.UploadMedia(context.Background(), mautrix.ReqUploadMedia{
		Content:       resp.Body,
		ContentLength: resp.ContentLength,
		ContentType:   mime.TypeByExtension(parts[len(parts)-1]),
	})
	if err != nil {
		log.Printf("matrix: upload failed for %s: %v\n", url, err)

		return nil
	}

	curl := mxc.ContentURI.CUString()

	p.mxcCache.Set(url, curl)

	return &curl
}
