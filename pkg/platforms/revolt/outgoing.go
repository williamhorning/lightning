package revolt

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func getOutgoing(
	token string,
	message lightning.Message,
	opts *lightning.SendOptions,
) revoltMessageSend {
	content := message.Content

	if content == "" && len(message.Embeds) == 0 && len(message.Attachments) == 0 {
		content = "\u200B"
	}

	if opts != nil && !opts.AllowEveryonePings {
		content = strings.ReplaceAll(content, "@everyone", "@\u2800everyone")
		content = strings.ReplaceAll(content, "@online", "@\u2800online")
	}

	if len([]rune(content)) > 2000 {
		content = string([]rune(content)[:1997]) + "..." // split the message?
	}

	msg := revoltMessageSend{
		Attachments: getOutgoingAttachments(token, message.Attachments),
		Content:     content,
		Embeds:      getOutgoingEmbeds(message.Embeds),
		Replies:     getOutgoingReplies(message.RepliedTo),
	}

	if opts != nil {
		msg.Masquerade = getOutgoingMasquerade(message.Author)
	}

	return msg
}

func getOutgoingAttachments(token string, attachments []lightning.Attachment) []string {
	if len(attachments) == 0 {
		return nil
	}

	attachmentIDs := make([]string, 0, len(attachments))

	for _, attachment := range attachments {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachment.URL, nil)
		if err != nil {
			cancel()

			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()

			continue
		}

		file, err := uploadFile(token, "attachments", attachment.Name, resp.Body)
		if err == nil {
			attachmentIDs = append(attachmentIDs, file)
		}

		err = resp.Body.Close()
		if err != nil {
			slog.Warn("revolt: failed to close body", "err", err)
		}

		cancel()
	}

	return attachmentIDs
}

func getOutgoingEmbeds(embeds []lightning.Embed) []*revoltMessageEmbed {
	result := make([]*revoltMessageEmbed, 0, len(embeds))

	for _, embed := range embeds {
		result = append(result, convertOutgoingEmbed(embed))
	}

	return result
}

func convertOutgoingEmbed(embed lightning.Embed) *revoltMessageEmbed {
	revoltEmbed := &revoltMessageEmbed{
		Description: getEmbedDescription(embed),
	}

	if embed.Title != nil {
		revoltEmbed.Title = *embed.Title
	}

	if embed.URL != nil {
		revoltEmbed.URL = *embed.URL
	}

	if embed.Color != nil {
		revoltEmbed.Color = "#" + strconv.FormatInt(int64(*embed.Color), 16)
	}

	setEmbedMedia(revoltEmbed, embed)

	return revoltEmbed
}

func getEmbedDescription(embed lightning.Embed) string {
	description := ""
	if embed.Description != nil {
		description = *embed.Description
	}

	if len(embed.Fields) == 0 {
		return description
	}

	for _, field := range embed.Fields {
		if description != "" {
			description += "\n\n"
		}

		description += "**" + field.Name + "**\n" + field.Value
	}

	return description
}

func setEmbedMedia(revoltEmbed *revoltMessageEmbed, embed lightning.Embed) {
	if embed.Image != nil {
		revoltEmbed.Image = &revoltMessageEmbedImage{
			URL:    embed.Image.URL,
			Width:  embed.Image.Width,
			Height: embed.Image.Height,
		}
	}

	if embed.Video != nil {
		revoltEmbed.Video = &revoltMessageEmbedVideo{
			URL:    embed.Video.URL,
			Width:  embed.Video.Width,
			Height: embed.Video.Height,
		}
	}

	if embed.Thumbnail != nil && len([]rune(embed.Thumbnail.URL)) > 0 && len([]rune(embed.Thumbnail.URL)) <= 128 {
		revoltEmbed.IconURL = embed.Thumbnail.URL
	}
}

func getOutgoingReplies(replyIDs []string) []*revoltMessageReplies {
	if len(replyIDs) == 0 {
		return nil
	}

	replies := make([]*revoltMessageReplies, len(replyIDs))
	for i, id := range replyIDs {
		replies[i] = &revoltMessageReplies{
			ID:      id,
			Mention: false,
		}
	}

	return replies
}

func getOutgoingMasquerade(author lightning.MessageAuthor) *revoltMessageMasquerade {
	avatar := ""
	if author.ProfilePicture != nil && len([]rune(*author.ProfilePicture)) > 1 &&
		len([]rune(*author.ProfilePicture)) <= 256 {
		avatar = *author.ProfilePicture
	}

	return &revoltMessageMasquerade{
		Color: author.Color,
		Name: func() string {
			runes := []rune(author.Nickname)

			if len(runes) > 32 {
				return string(runes[:32])
			}

			return author.Nickname
		}(),
		Avatar: avatar,
	}
}
