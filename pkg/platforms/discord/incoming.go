package discord

import (
	"regexp"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/internal/cache"
	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func discordToLightning(
	webhooks *cache.Expiring[snowflake.ID, bool],
	session *bot.Client,
	msg *discord.Message,
	cdn string,
) *lightning.Message {
	if msg.Type != discord.MessageTypeDefault &&
		msg.Type != discord.MessageTypeReply &&
		msg.Type != discord.MessageTypeSlashCommand &&
		msg.Type != discord.MessageTypeContextMenuCommand {
		return nil
	}

	if msg.WebhookID == nil {
		t := snowflake.ID(1234567890)
		msg.WebhookID = &t
	}

	if exists, _ := webhooks.Get(*msg.WebhookID); exists {
		return nil
	}

	message := &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID: msg.ID.String(), ChannelID: msg.ChannelID.String(), Time: msg.ID.Time(),
		},
		Author: discordToLightningAuthor(session, msg, cdn),
		Content: discordToLightningForward(session, msg) +
			discordToLightningContent(session, msg.Content, msg.GuildID),
		Embeds:      discordToLightningEmbeds(msg.Embeds),
		Attachments: discordToLightningAttachments(msg.Attachments, msg.StickerItems, cdn),
		RepliedTo:   discordToLightningReplies(msg.MessageReference),
	}

	message.Content = discordToLightningEmoji(message, cdn)

	return message
}

func discordToLightningAuthor(session *bot.Client, msg *discord.Message, cdn string) *lightning.MessageAuthor {
	author := &lightning.MessageAuthor{
		ID:             msg.Author.ID.String(),
		Username:       msg.Author.EffectiveName(),
		Color:          "#5865F2",
		ProfilePicture: msg.Author.EffectiveAvatarURL(),
	}

	if cdn != "" {
		author.ProfilePicture = strings.ReplaceAll(
			strings.ReplaceAll(author.ProfilePicture, "cdn.discordapp.com", cdn), "media.discordapp.net", cdn,
		)
	}

	if msg.GuildID == nil {
		return author
	}

	member := msg.Member
	if member == nil {
		if m, ok := session.Caches.Member(*msg.GuildID, msg.Author.ID); ok {
			member = &m
		}
	}

	if member == nil {
		return author
	}

	member.User = msg.Author
	author.Username = member.EffectiveName()
	author.ProfilePicture = member.EffectiveAvatarURL()
	author.ProfilePicture = strings.ReplaceAll(
		strings.ReplaceAll(author.ProfilePicture, "cdn.discordapp.com", cdn), "media.discordapp.net", cdn,
	)

	return author
}

func discordToLightningForward(session *bot.Client, msg *discord.Message) string {
	if msg.MessageReference == nil || msg.MessageReference.MessageID == nil ||
		msg.MessageReference.Type != discord.MessageReferenceTypeForward || len(msg.MessageSnapshots) == 0 {
		return ""
	}

	out := ""

	for _, snapshot := range msg.MessageSnapshots {
		content := discordToLightningContent(session, snapshot.Message.Content, nil)
		out += "> " + strings.ReplaceAll(content, "\n", "\n> ")
	}

	return out
}

var (
	tenorURL       = regexp.MustCompile(`https://tenor\.com/view/[^/]+-(\d+).*`)
	userMention    = regexp.MustCompile(`<@!?(\d+)>`)
	channelMention = regexp.MustCompile(`<#(\d+)>`)
	roleMention    = regexp.MustCompile(`<@&(\d+)>`)
	emojiMention   = regexp.MustCompile(`<a?:\w+:(\d+)>`)
	defaultEmoji   = regexp.MustCompile(`(?:^|[^<])(:[^:\s]*(?:::[^:\s]*)*:)`)
)

func discordToLightningContent( //nolint:cyclop,revive
	session *bot.Client, content string, guildID *snowflake.ID,
) string {
	content = defaultEmoji.ReplaceAllStringFunc(content, func(match string) string {
		if e, ok := emoji.Emoji[match]; ok {
			return e
		}

		return match
	})

	content = tenorURL.ReplaceAllStringFunc(content, func(match string) string {
		return "https://tenor.com/view/" + tenorURL.FindStringSubmatch(match)[1] + ".gif"
	})

	content = userMention.ReplaceAllStringFunc(content, func(match string) string {
		userID, err := snowflake.Parse(userMention.FindStringSubmatch(match)[1])
		if err == nil {
			if guildID != nil {
				if m, ok := session.Caches.Member(*guildID, userID); ok {
					return "@" + m.EffectiveName()
				}
			}

			if u, err := session.Rest.GetUser(userID); err == nil {
				return "@" + u.EffectiveName()
			}
		}

		return "@" + userMention.FindStringSubmatch(match)[1]
	})

	content = channelMention.ReplaceAllStringFunc(content, func(match string) string {
		channelID, err := snowflake.Parse(channelMention.FindStringSubmatch(match)[1])
		if err == nil {
			if ch, ok := session.Caches.Channel(channelID); ok {
				return "#" + ch.Name()
			}
		}

		return "#" + channelMention.FindStringSubmatch(match)[1]
	})

	content = roleMention.ReplaceAllStringFunc(content, func(match string) string {
		roleID, err := snowflake.Parse(roleMention.FindStringSubmatch(match)[1])
		if err == nil && guildID != nil {
			if r, ok := session.Caches.Role(*guildID, roleID); ok {
				return "@" + r.Name
			}
		}

		return "@&" + roleMention.FindStringSubmatch(match)[1]
	})

	return content
}

func discordToLightningEmbeds(embeds []discord.Embed) []lightning.Embed {
	out := make([]lightning.Embed, 0, len(embeds))
	for _, original := range embeds {
		embed := lightning.Embed{
			Footer: discordToLightningEmbedFooter(original.Footer),
			Author: discordToLightningEmbedAuthor(original.Author),
			Fields: discordToLightningEmbedFields(original.Fields),
			Timestamp: func() string {
				if original.Timestamp != nil {
					return original.Timestamp.Format(time.RFC3339)
				}

				return ""
			}(),
			Title:       original.Title,
			URL:         original.URL,
			Description: original.Description,
			Color:       original.Color,
		}

		if original.Image != nil && original.Image.URL != "" {
			embed.Image = &lightning.Media{URL: original.Image.URL}
		}

		if original.Thumbnail != nil && original.Thumbnail.URL != "" {
			embed.Thumbnail = &lightning.Media{URL: original.Thumbnail.URL}
		}

		out = append(out, embed)
	}

	return out
}

func discordToLightningEmbedFooter(original *discord.EmbedFooter) *lightning.EmbedFooter {
	if original == nil {
		return nil
	}

	return &lightning.EmbedFooter{Text: original.Text, IconURL: original.IconURL}
}

func discordToLightningEmbedAuthor(original *discord.EmbedAuthor) *lightning.EmbedAuthor {
	if original == nil {
		return nil
	}

	return &lightning.EmbedAuthor{Name: original.Name, URL: original.URL, IconURL: original.URL}
}

func discordToLightningEmbedFields(original []discord.EmbedField) []lightning.EmbedField {
	fields := make([]lightning.EmbedField, len(original))

	for idx, field := range original {
		if field.Inline == nil {
			t := false
			field.Inline = &t
		}

		fields[idx] = lightning.EmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: *field.Inline,
		}
	}

	return fields
}

func discordToLightningAttachments(
	attachments []discord.Attachment,
	stickers []discord.MessageSticker,
	cdn string,
) []lightning.Attachment {
	result := make([]lightning.Attachment, 0, len(attachments)+len(stickers))

	for _, a := range attachments {
		result = append(result, lightning.Attachment{
			URL:  a.URL,
			Name: a.Filename,
			Size: int64(a.Size),
		})
	}

	for _, sticker := range stickers {
		url := "https://" + cdn + "/stickers/" + sticker.ID.String()

		switch sticker.FormatType {
		case discord.StickerFormatTypePNG, discord.StickerFormatTypeAPNG:
			url += ".png"
		case discord.StickerFormatTypeGIF:
			url += ".gif"
			url = strings.ReplaceAll(url, "cdn.discordapp.com", "media.discordapp.net")
		case discord.StickerFormatTypeLottie:
			url += ".json"
		default:
		}

		result = append(result, lightning.Attachment{
			URL:  url + "?size=160",
			Name: sticker.Name,
		})
	}

	return result
}

func discordToLightningReplies(reference *discord.MessageReference) []string {
	if reference == nil || reference.MessageID == nil || reference.Type != discord.MessageReferenceTypeDefault {
		return nil
	}

	return []string{reference.MessageID.String()}
}

func discordToLightningEmoji(msg *lightning.Message, cdn string) string {
	return emojiMention.ReplaceAllStringFunc(msg.Content, func(match string) string {
		parts := strings.Split(match, ":")
		if len(parts) < 3 {
			return match
		}

		emojiID := parts[2]
		emojiName := parts[1]

		url := "https://" + cdn + "/emojis/" + emojiID
		if strings.HasPrefix(match, "<a") {
			url += ".gif?size=48"
		} else {
			url += ".png?size=48"
		}

		msg.Emoji = append(msg.Emoji, lightning.Emoji{ID: emojiID, Name: emojiName, URL: url})

		return ":" + emojiName + ":"
	})
}
