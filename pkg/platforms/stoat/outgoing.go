package stoat

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/williamhorning/lightning/internal/emoji"
	"github.com/williamhorning/lightning/internal/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func lightningToStoatMessage(
	session *stoat.Session,
	message *lightning.Message,
	opts *lightning.SendOptions,
) stoat.DataMessageSend {
	content := stoatOutgoingSpoilerRegex.ReplaceAllStringFunc(stoatOutgoingEmojiRegex.ReplaceAllStringFunc(
		message.Content, func(match string) string { return replaceStoatOutgoingEmoji(session, message, match) }),
		func(match string) string { return "!!" + match[2:len(match)-2] + "!!" },
	)

	if opts != nil && !opts.AllowEveryonePings {
		content = strings.ReplaceAll(content, "@everyone", "@\u2800everyone")
		content = strings.ReplaceAll(content, "@online", "@\u2800online")
	}

	if len(content) > 2000 {
		content = content[:1997] + "..."
	}

	msg := stoat.DataMessageSend{
		Attachments: lightningToStoatAttachments(session, message.Attachments),
		Content:     content,
		Embeds:      lightningToStoatEmbeds(message.Embeds),
		Replies:     lightningToStoatReplies(message.RepliedTo),
	}

	if len(content) == 0 && len(msg.Embeds) == 0 && len(msg.Attachments) == 0 {
		msg.Content = "\u200B"
	}

	if opts != nil && message.Author != nil {
		msg.Masquerade = lightningToStoatMasquerade(*message.Author)
	}

	return msg
}

var (
	stoatOutgoingEmojiRegex   = regexp.MustCompile(`:\w+:`)
	stoatOutgoingSpoilerRegex = regexp.MustCompile(`\|\|(.+?)\|\|`)
)

func replaceStoatOutgoingEmoji(session *stoat.Session, message *lightning.Message, match string) string {
	if str, ok := emoji.GetEmoji(match); ok {
		return str
	}

	name := strings.ReplaceAll(match, ":", "")

	channel, err := stoat.Get(session, "/channels/"+message.ChannelID, message.ChannelID, &session.ChannelCache)
	if err == nil && channel.ChannelType == stoat.ChannelTypeText && channel.Server != nil {
		serverEmojis, err := stoat.Get(session, "/servers/"+*channel.Server+"/emojis", *channel.Server,
			&session.ServerEmojiCache)
		if err == nil {
			for _, instance := range *serverEmojis {
				if instance.Name == name {
					return ":" + instance.ID + ":"
				}
			}
		}
	}

	for _, e := range message.Emoji {
		if e.Name == name {
			return "[" + e.Name + "](" + e.URL + ")"
		}
	}

	return match
}

func lightningToStoatAttachments(session *stoat.Session, attachments []lightning.Attachment) []string {
	out := make([]string, 0, len(attachments))
	for _, att := range attachments {
		file, err := session.UploadFile("attachments", att.URL, att.Name)
		if err == nil {
			out = append(out, file.ID)
		} else {
			log.Printf("%v\n", err)
		}
	}

	return out
}

func lightningToStoatEmbeds(embeds []lightning.Embed) []stoat.SendableEmbed {
	out := make([]stoat.SendableEmbed, 0, len(embeds))

	for _, e := range embeds {
		out = append(out, lightningToStoatEmbed(e))
	}

	return out
}

func lightningToStoatEmbed(embed lightning.Embed) stoat.SendableEmbed {
	newEmbed := stoat.SendableEmbed{
		Title:       embed.Title,
		Description: *stoatEmbedDescription(embed),
	}

	if embed.URL != "" {
		if len(embed.URL) > 256 {
			embed.URL = embed.URL[:256]
		}

		newEmbed.URL = embed.URL
	}

	if embed.Color != 0 {
		newEmbed.Colour = "#" + strconv.FormatInt(int64(embed.Color), 16)
	}

	setStoatEmbedMedia(&newEmbed, embed)

	return newEmbed
}

func stoatEmbedDescription(embed lightning.Embed) *string {
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

func setStoatEmbedMedia(sEmbed *stoat.SendableEmbed, embed lightning.Embed) {
	if embed.Image != nil {
		sEmbed.Media = embed.Image.URL
	}

	if embed.Video != nil {
		sEmbed.Media = embed.Video.URL
	}

	if embed.Thumbnail != nil && len(embed.Thumbnail.URL) > 0 && len(embed.Thumbnail.URL) <= 128 {
		sEmbed.IconURL = embed.Thumbnail.URL
	}
}

func lightningToStoatReplies(replyIDs []string) []stoat.ReplyIntent {
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

func lightningToStoatMasquerade(author lightning.MessageAuthor) *stoat.Masquerade {
	if len(author.ProfilePicture) >= 256 {
		author.ProfilePicture = author.ProfilePicture[:256]
	}

	if len(author.Nickname) > 32 {
		author.Nickname = author.Nickname[:32]
	}

	return &stoat.Masquerade{
		Colour: author.Color,
		Name:   author.Nickname,
		Avatar: author.ProfilePicture,
	}
}
