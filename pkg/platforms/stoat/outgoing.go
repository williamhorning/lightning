package stoat

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/williamhorning/lightning/internal/emoji"
	"github.com/williamhorning/lightning/internal/rvapi"
	"github.com/williamhorning/lightning/internal/workaround"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *stoatPlugin) getOutgoing(
	message *lightning.Message,
	opts *lightning.SendOptions,
) rvapi.DataMessageSend {
	content := spoilerRegex.ReplaceAllStringFunc(
		emojiSendRegex.ReplaceAllStringFunc(message.Content, p.replaceOutgoingEmoji(message)),
		func(match string) string { return "!!" + match[2:len(match)-2] + "!!" },
	)

	if opts != nil && !opts.AllowEveryonePings {
		content = strings.ReplaceAll(content, "@everyone", "@\u2800everyone")
		content = strings.ReplaceAll(content, "@online", "@\u2800online")
	}

	if len([]rune(content)) > 2000 {
		content = string([]rune(content)[:1997]) + "..." // split the message?
	}

	msg := rvapi.DataMessageSend{
		Attachments: p.getOutgoingAttachments(message.Attachments),
		Content:     content,
		Embeds:      getOutgoingEmbeds(message.Embeds),
		Replies:     getOutgoingReplies(message.RepliedTo),
	}

	if len(content) == 0 && len(msg.Embeds) == 0 && len(msg.Attachments) == 0 {
		msg.Content = "\u200B"
	}

	if opts != nil {
		msg.Masquerade = getOutgoingMasquerade(message.Author)
	}

	return msg
}

var emojiSendRegex = regexp.MustCompile(`:\w+:`)

func (p *stoatPlugin) replaceOutgoingEmoji(message *lightning.Message) func(string) string {
	return func(match string) string {
		if emoji.IsEmoji(match) {
			return match
		}

		name := strings.ReplaceAll(match, ":", "")

		channel := p.session.Channel(message.ChannelID)

		if channel != nil && channel.ChannelType == rvapi.ChannelTypeText && channel.Server != nil {
			for _, emoji := range p.session.ServerEmoji(*channel.Server) {
				if emoji.Name == name {
					return ":" + emoji.ID + ":"
				}
			}
		}

		for _, emoji := range message.Emoji {
			if emoji.Name == name && emoji.URL != nil {
				return "[" + emoji.Name + "](" + *emoji.URL + ")"
			}
		}

		return match
	}
}

func (p *stoatPlugin) getOutgoingAttachments(attachments []lightning.Attachment) []string {
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

		resp, err := workaround.Client.Do(req)
		if err != nil {
			cancel()

			continue
		}

		file, err := p.session.UploadFile("attachments", attachment.Name, resp.Body)
		if err == nil {
			attachmentIDs = append(attachmentIDs, file.ID)
		} else {
			log.Printf("%v\n", err)
		}

		err = resp.Body.Close()
		if err != nil {
			log.Printf("stoat: failed to close upload body: %v\n", err)
		}

		cancel()
	}

	return attachmentIDs
}

func getOutgoingEmbeds(embeds []lightning.Embed) []rvapi.SendableEmbed {
	result := make([]rvapi.SendableEmbed, 0, len(embeds))

	for _, embed := range embeds {
		result = append(result, convertOutgoingEmbed(embed))
	}

	return result
}

func convertOutgoingEmbed(embed lightning.Embed) rvapi.SendableEmbed {
	stoatEmbed := rvapi.SendableEmbed{
		Title:       embed.Title,
		Description: getEmbedDescription(embed),
		URL:         embed.URL,
	}

	if embed.Color != nil {
		color := "#" + strconv.FormatInt(int64(*embed.Color), 16)

		stoatEmbed.Colour = &color
	}

	setEmbedMedia(&stoatEmbed, embed)

	return stoatEmbed
}

func getEmbedDescription(embed lightning.Embed) *string {
	description := ""
	if embed.Description != nil {
		description = *embed.Description
	}

	if len(embed.Fields) == 0 {
		return &description
	}

	for _, field := range embed.Fields {
		if description != "" {
			description += "\n\n"
		}

		description += "**" + field.Name + "**\n" + field.Value
	}

	if description == "" {
		return nil
	}

	return &description
}

func setEmbedMedia(stoatEmbed *rvapi.SendableEmbed, embed lightning.Embed) {
	if embed.Image != nil {
		stoatEmbed.Media = &embed.Image.URL
	}

	if embed.Video != nil {
		stoatEmbed.Media = &embed.Video.URL
	}

	if embed.Thumbnail != nil && len([]rune(embed.Thumbnail.URL)) > 0 && len([]rune(embed.Thumbnail.URL)) <= 128 {
		stoatEmbed.IconURL = &embed.Thumbnail.URL
	}
}

func getOutgoingReplies(replyIDs []string) []rvapi.ReplyIntent {
	replies := make([]rvapi.ReplyIntent, len(replyIDs))
	for i, id := range replyIDs {
		replies[i] = rvapi.ReplyIntent{
			ID:              id,
			Mention:         false,
			FailIfNotExists: false,
		}
	}

	return replies
}

func getOutgoingMasquerade(author *lightning.MessageAuthor) *rvapi.Masquerade {
	avatar := ""
	if author.ProfilePicture != nil && len([]rune(*author.ProfilePicture)) > 1 &&
		len([]rune(*author.ProfilePicture)) <= 256 {
		avatar = *author.ProfilePicture
	}

	nickname := author.Nickname

	if len([]rune(nickname)) > 32 {
		nickname = string([]rune(nickname))[:32]
	}

	return &rvapi.Masquerade{
		Colour: author.Color,
		Name:   nickname,
		Avatar: avatar,
	}
}
