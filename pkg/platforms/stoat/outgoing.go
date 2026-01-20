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
) []stDataMessageSend {
	content := stoatOutgoingEmojiRegex.ReplaceAllStringFunc(message.Content,
		func(match string) string { return replaceStoatOutgoingEmoji(session, message, match) },
	)

	if !opts.AllowEveryonePings {
		content = strings.ReplaceAll(content, "@everyone", "@\u2800everyone")
		content = strings.ReplaceAll(content, "@online", "@\u2800online")
	}

	msg := stDataMessageSend{
		Attachments: lightningToStoatAttachments(session, message.Attachments),
		Content:     content,
		Embeds:      lightningToStoatEmbeds(session, message.Embeds),
		Replies:     lightningToStoatReplies(message.RepliedTo),
	}

	if content == "" && len(msg.Embeds) == 0 && len(msg.Attachments) == 0 {
		msg.Content = "\u200B"
	}

	if message.Author != nil {
		msg.Masquerade = lightningToStoatMasquerade(*message.Author)
	}

	return splitMessageSend(&msg)
}

func splitMessageSend(msg *stDataMessageSend) []stDataMessageSend { //nolint:cyclop,revive
	chunks, content, embeds, attachments := []stDataMessageSend{}, []rune(msg.Content), msg.Embeds, msg.Attachments

	for len(content) > 0 || len(embeds) > 0 || len(attachments) > 0 {
		chunk := stDataMessageSend{Masquerade: msg.Masquerade, Replies: msg.Replies}
		budget := 2000
		take := min(len(content), budget)
		chunk.Content, content, budget = string(content[:take]), content[take:], budget-take

		for len(embeds) > 0 && len(chunk.Embeds) < 5 && budget > 0 {
			desc := []rune(embeds[0].Description)

			if len(desc) <= budget {
				chunk.Embeds = append(chunk.Embeds, embeds[0])
				budget -= len(desc)
				embeds = embeds[1:]
			} else {
				split := embeds[0]
				split.Description = string(desc[:budget])
				chunk.Embeds = append(chunk.Embeds, split)
				embeds[0].Description = string(desc[budget:])
				budget = 0
			}
		}

		chunk.Attachments = append([]string(nil), attachments[:min(len(attachments), 5)]...)
		attachments = attachments[min(len(attachments), 5):]

		if chunk.Content == "" && len(chunk.Embeds) == 0 && len(chunk.Attachments) == 0 {
			break
		}

		chunks = append(chunks, chunk)
	}

	return chunks
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
			log.Printf("stoat: %v\n", err)
		}
	}

	return out
}

func lightningToStoatEmbeds(session *session, embeds []lightning.Embed) []stSendableEmbed {
	out := make([]stSendableEmbed, 0, len(embeds))

	embeds = embeds[:min(len(embeds), 10)]

	for idx := range embeds {
		out = append(out, lightningToStoatEmbed(session, &embeds[idx]))
	}

	return out
}

func lightningToStoatEmbed(session *session, embed *lightning.Embed) stSendableEmbed {
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

	setStoatEmbedMedia(session, &newEmbed, embed)

	return newEmbed
}

func stoatEmbedDescription(embed *lightning.Embed) *string {
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

func setStoatEmbedMedia(session *session, sEmbed *stSendableEmbed, embed *lightning.Embed) {
	if embed.Video != nil {
		name := strings.Split(embed.Video.URL, "/")

		if id, err := session.uploadFile(embed.Video.URL, name[len(name)-1]); err == nil {
			sEmbed.Media = id
		}
	} else if embed.Image != nil {
		name := strings.Split(embed.Image.URL, "/")

		if id, err := session.uploadFile(embed.Image.URL, name[len(name)-1]); err == nil {
			sEmbed.Media = id
		}
	}

	if embed.Thumbnail != nil && embed.Thumbnail.URL != "" && len(embed.Thumbnail.URL) <= 128 {
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
