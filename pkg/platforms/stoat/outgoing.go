package stoat

import (
	"log"
	"regexp"
	"strconv"
	"strings"

	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
)

var stoatOutgoingEmojiRegex = regexp.MustCompile(`:\w+:`)

func lightningToStoatMessage(
	session *session,
	message *lightning.Message,
	opts *lightning.SendOptions,
) stDataMessageSend {
	content := stoatOutgoingEmojiRegex.ReplaceAllStringFunc(message.Content,
		func(match string) string { return replaceStoatOutgoingEmoji(session, message, match) },
	)

	if !opts.AllowEveryonePings {
		content = strings.ReplaceAll(content, "@everyone", "@\u2800everyone")
		content = strings.ReplaceAll(content, "@online", "@\u2800online")
	}

	if len(content) > 2000 {
		content = content[:1997] + "..."
	}

	msg := stDataMessageSend{
		Attachments: lightningToStoatAttachments(session, message.Attachments),
		Content:     content,
		Embeds:      lightningToStoatEmbeds(message.Embeds),
		Replies:     lightningToStoatReplies(message.RepliedTo),
	}

	if len(content) == 0 && len(msg.Embeds) == 0 && len(msg.Attachments) == 0 {
		msg.Content = "\u200B"
	}

	if message.Author != nil {
		msg.Masquerade = lightningToStoatMasquerade(*message.Author)
	}

	return msg
}

func replaceStoatOutgoingEmoji(session *session, message *lightning.Message, match string) string {
	if str, ok := emoji.Emoji[match]; ok {
		return str
	}

	name := strings.ReplaceAll(match, ":", "")

	channel, err := get(session, "/channels/"+message.ChannelID, message.ChannelID, &session.channelCache)
	if err == nil && channel.Server != nil {
		serverEmojis, err := get(session, "/servers/"+*channel.Server+"/emojis", *channel.Server,
			&session.serverEmojiCache)
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

func lightningToStoatAttachments(session *session, attachments []lightning.Attachment) []string {
	out := make([]string, 0, len(attachments))
	for _, att := range attachments {
		file, err := session.uploadFile(att.URL, att.Name)
		if err == nil {
			out = append(out, file)
		} else {
			log.Printf("stoat: %v\n", err) // TODO
		}
	}

	return out
}

func lightningToStoatEmbeds(embeds []lightning.Embed) []stSendableEmbed {
	out := make([]stSendableEmbed, 0, len(embeds))

	for _, e := range embeds {
		out = append(out, lightningToStoatEmbed(e))
	}

	return out
}

func lightningToStoatEmbed(embed lightning.Embed) stSendableEmbed {
	newEmbed := stSendableEmbed{
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

func setStoatEmbedMedia(sEmbed *stSendableEmbed, embed lightning.Embed) {
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

func lightningToStoatReplies(replyIDs []string) []stReplyIntent {
	replies := make([]stReplyIntent, len(replyIDs))

	for i, id := range replyIDs {
		replies[i] = stReplyIntent{
			ID:              id,
			Mention:         false,
			FailIfNotExists: false,
		}
	}

	return replies
}

func lightningToStoatMasquerade(author lightning.MessageAuthor) *stMasquerade {
	if len(author.ProfilePicture) >= 256 {
		author.ProfilePicture = author.ProfilePicture[:256]
	}

	if len(author.Username) > 32 {
		author.Username = author.Username[:32]
	}

	return &stMasquerade{
		Colour: author.Color,
		Name:   author.Username,
		Avatar: author.ProfilePicture,
	}
}
