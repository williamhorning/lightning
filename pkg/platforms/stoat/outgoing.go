package stoat

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/williamhorning/lightning/internal/v2/emoji"
	"github.com/williamhorning/lightning/internal/v2/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *stoatPlugin) getOutgoing(
	message *lightning.Message,
	opts *lightning.SendOptions,
) stoat.DataMessageSend {
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

	msg := stoat.DataMessageSend{
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
		if str, ok := emoji.GetEmoji(match); ok {
			return str
		}

		name := strings.ReplaceAll(match, ":", "")

		channel := stoat.Get(p.session, "/channels/"+message.ChannelID, message.ChannelID, &p.session.ChannelCache)

		if channel != nil && channel.ChannelType == stoat.ChannelTypeText && channel.Server != nil {
			serverEmojis := *stoat.Get(p.session, "/servers/"+*channel.Server+"/emojis", *channel.Server,
				&p.session.ServerEmojiCache)
			for _, emoji := range serverEmojis {
				if emoji.Name == name {
					return ":" + emoji.ID + ":"
				}
			}
		}

		for _, emoji := range message.Emoji {
			if emoji.Name == name {
				return "[" + emoji.Name + "](" + emoji.URL + ")"
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
		file, err := p.session.UploadFile("attachments", attachment.URL, attachment.Name)
		if err == nil {
			attachmentIDs = append(attachmentIDs, file.ID)
		} else {
			log.Printf("%v\n", err)
		}
	}

	return attachmentIDs
}

func getOutgoingEmbeds(embeds []lightning.Embed) []stoat.SendableEmbed {
	result := make([]stoat.SendableEmbed, 0, len(embeds))

	for _, embed := range embeds {
		result = append(result, convertOutgoingEmbed(embed))
	}

	return result
}

func convertOutgoingEmbed(embed lightning.Embed) stoat.SendableEmbed {
	stoatEmbed := stoat.SendableEmbed{
		Title:       embed.Title,
		Description: *getEmbedDescription(embed),
	}

	if embed.URL != "" {
		if len(embed.URL) > 256 {
			embed.URL = embed.URL[:256]
		}

		stoatEmbed.URL = embed.URL
	}

	if embed.Color != 0 {
		stoatEmbed.Colour = "#" + strconv.FormatInt(int64(embed.Color), 16)
	}

	setEmbedMedia(&stoatEmbed, embed)

	return stoatEmbed
}

func getEmbedDescription(embed lightning.Embed) *string {
	if len(embed.Fields) == 0 {
		return &embed.Description
	}

	for _, field := range embed.Fields {
		if embed.Description != "" {
			embed.Description += "\n\n"
		}

		embed.Description += "**" + field.Name + "**\n" + field.Value
	}

	if embed.Description == "" {
		return nil
	}

	return &embed.Description
}

func setEmbedMedia(stoatEmbed *stoat.SendableEmbed, embed lightning.Embed) {
	if embed.Image != nil {
		stoatEmbed.Media = embed.Image.URL
	}

	if embed.Video != nil {
		stoatEmbed.Media = embed.Video.URL
	}

	if embed.Thumbnail != nil && len([]rune(embed.Thumbnail.URL)) > 0 && len([]rune(embed.Thumbnail.URL)) <= 128 {
		stoatEmbed.IconURL = embed.Thumbnail.URL
	}
}

func getOutgoingReplies(replyIDs []string) []stoat.ReplyIntent {
	replies := make([]stoat.ReplyIntent, len(replyIDs))
	for i, id := range replyIDs {
		replies[i] = stoat.ReplyIntent{
			ID:              id,
			Mention:         false,
			FailIfNotExists: false,
		}
	}

	return replies
}

func getOutgoingMasquerade(author *lightning.MessageAuthor) *stoat.Masquerade {
	avatar := ""
	if len([]rune(author.ProfilePicture)) > 1 && len([]rune(author.ProfilePicture)) <= 256 {
		avatar = author.ProfilePicture
	}

	nickname := author.Nickname

	if len([]rune(nickname)) > 32 {
		nickname = string([]rune(nickname))[:32]
	}

	return &stoat.Masquerade{
		Colour: author.Color,
		Name:   nickname,
		Avatar: avatar,
	}
}
