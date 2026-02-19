package discord

import (
	"regexp"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
)

func discordToLightning(
	bot *client,
	msg *message,
) *lightning.Message {
	if msg.Type != messageTypeDefault &&
		msg.Type != messageTypeReply &&
		msg.Type != messageTypeChatInputCommand &&
		msg.Type != messageTypeContextMenuCommand {
		return nil
	}

	if msg.WebhookID == nil {
		t := snowflake("")
		msg.WebhookID = &t
	}

	if webhook, ok := bot.getWebhook(msg.WebhookID); ok && webhook.ApplicationID == bot.application.ID {
		return nil
	}

	message := &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID: string(msg.ID), ChannelID: string(msg.ChannelID), Time: msg.Timestamp,
		},
		Author: discordToLightningAuthor(bot, &msg.Author, msg.Member, msg.GuildID),
		Content: discordToLightningForward(bot, msg) +
			discordToLightningContent(bot, msg.Content, msg.GuildID),
		Embeds:      discordToLightningEmbeds(msg.Embeds),
		Attachments: discordToLightningAttachments(bot, msg.Attachments, msg.StickerItems),
		RepliedTo:   discordToLightningReplies(msg.MessageReference),
	}

	message.Content = discordToLightningEmoji(bot, message)

	return message
}

func discordToLightningAuthor(bot *client, original *user, member *member, guild *snowflake) *lightning.MessageAuthor {
	author := &lightning.MessageAuthor{
		ID:             string(original.ID),
		Username:       original.displayName(),
		ProfilePicture: original.avatarURL(bot),
		Color:          "#5865F2",
	}

	if guild == nil {
		return author
	}

	if member == nil {
		member, _ = bot.getMember(guild, original.ID)
	}

	if member == nil {
		return author
	}

	member.User = original
	author.Username = member.displayName()
	author.ProfilePicture = member.avatarURL(bot, *guild)

	return author
}

func discordToLightningForward(bot *client, msg *message) string {
	if msg.MessageReference == nil || msg.MessageReference.MessageID == "" ||
		msg.MessageReference.Type != forwardReference || len(msg.MessageSnapshots) == 0 {
		return ""
	}

	var out strings.Builder

	for idx := range msg.MessageSnapshots {
		content := discordToLightningContent(bot, msg.MessageSnapshots[idx].Content, nil)
		out.WriteString("> " + strings.ReplaceAll(content, "\n", "\n> "))
	}

	return out.String()
}

var (
	userMention    = regexp.MustCompile(`<@!?(\d+)>`)
	channelMention = regexp.MustCompile(`<#(\d+)>`)
	roleMention    = regexp.MustCompile(`<@&(\d+)>`)
	emojiMention   = regexp.MustCompile(`<a?:\w+:(\d+)>`)
	defaultEmoji   = regexp.MustCompile(`(?:^|[^<])(:[^:\s]*(?:::[^:\s]*)*:)`)
)

func discordToLightningContent(
	bot *client, content string, guildID *snowflake,
) string {
	content = defaultEmoji.ReplaceAllStringFunc(content, func(match string) string {
		if e, ok := emoji.Emoji[match]; ok {
			return e
		}

		return match
	})

	content = userMention.ReplaceAllStringFunc(content, func(match string) string {
		if member, ok := bot.getMember(guildID, snowflake(userMention.FindStringSubmatch(match)[1])); ok {
			return "@" + member.displayName()
		}

		if u, ok := bot.getUser(userMention.FindStringSubmatch(match)[1]); ok {
			return "@" + u.displayName()
		}

		return "@" + userMention.FindStringSubmatch(match)[1]
	})

	content = channelMention.ReplaceAllStringFunc(content, func(match string) string {
		if ch, ok := bot.getChannel(channelMention.FindStringSubmatch(match)[1]); ok {
			return "#" + ch.Name
		}

		return "#" + channelMention.FindStringSubmatch(match)[1]
	})

	content = roleMention.ReplaceAllStringFunc(content, func(match string) string {
		if r, ok := bot.getRole(guildID, roleMention.FindStringSubmatch(match)[1]); ok {
			return "@" + r.Name
		}

		return "@&" + roleMention.FindStringSubmatch(match)[1]
	})

	return content
}

func discordToLightningEmbeds(embeds []embed) []lightning.Embed {
	out := make([]lightning.Embed, 0, len(embeds))
	for idx := range embeds {
		embed := lightning.Embed{
			Footer: discordToLightningEmbedFooter(embeds[idx].Footer),
			Author: discordToLightningEmbedAuthor(embeds[idx].Author),
			Fields: discordToLightningEmbedFields(embeds[idx].Fields),
			Timestamp: func() string {
				if embeds[idx].Timestamp != nil {
					return embeds[idx].Timestamp.Format(time.RFC3339)
				}

				return ""
			}(),
			Title:       embeds[idx].Title,
			URL:         embeds[idx].URL,
			Description: embeds[idx].Description,
			Color:       embeds[idx].Color,
		}

		if embeds[idx].Image != nil && embeds[idx].Image.URL != "" {
			embed.Image = &lightning.Media{URL: embeds[idx].Image.URL}
		}

		if embeds[idx].Thumbnail != nil && embeds[idx].Thumbnail.URL != "" {
			embed.Thumbnail = &lightning.Media{URL: embeds[idx].Thumbnail.URL}
		}

		out = append(out, embed)
	}

	return out
}

func discordToLightningEmbedFooter(original *embedFooter) *lightning.EmbedFooter {
	if original == nil {
		return nil
	}

	return &lightning.EmbedFooter{Text: original.Text, IconURL: original.IconURL}
}

func discordToLightningEmbedAuthor(original *embedAuthor) *lightning.EmbedAuthor {
	if original == nil {
		return nil
	}

	return &lightning.EmbedAuthor{Name: original.Name, URL: original.URL, IconURL: original.URL}
}

func discordToLightningEmbedFields(original []embedField) []lightning.EmbedField {
	fields := make([]lightning.EmbedField, len(original))

	for idx, field := range original {
		fields[idx] = lightning.EmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: field.Inline,
		}
	}

	return fields
}

func discordToLightningAttachments(
	bot *client,
	attachments []attachment,
	stickers []stickerItem,
) []lightning.Attachment {
	result := make([]lightning.Attachment, 0, len(attachments)+len(stickers))

	for idx := range attachments {
		result = append(result, lightning.Attachment{
			URL:  attachments[idx].URL,
			Name: attachments[idx].Filename,
			Size: int64(attachments[idx].Size),
		})
	}

	for _, sticker := range stickers {
		url := "https://" + bot.cdnHost + "/stickers/" + sticker.ID
		ext := ""

		switch sticker.FormatType {
		case stickerPNG, stickerAPNG:
			url += ".png"
			ext += ".png"
		case stickerGIF:
			url += ".gif"
			ext += ".gif"
			url = strings.ReplaceAll(url, "cdn.discordapp.com", "media.discordapp.net")
		case stickerLottie:
			url += ".json"
			ext += ".json"
		default:
		}

		result = append(result, lightning.Attachment{
			URL:  url + "?size=160",
			Name: sticker.Name + ext,
		})
	}

	return result
}

func discordToLightningReplies(reference *messageReference) []string {
	if reference == nil || reference.MessageID == "" || reference.Type != defaultReference {
		return nil
	}

	return []string{string(reference.MessageID)}
}

func discordToLightningEmoji(bot *client, msg *lightning.Message) string {
	return emojiMention.ReplaceAllStringFunc(msg.Content, func(match string) string {
		parts := strings.Split(match, ":")
		if len(parts) < 3 {
			return match
		}

		emojiID := strings.TrimSuffix(parts[2], ">")
		emojiName := parts[1]

		url := "https://" + bot.cdnHost + "/emojis/" + emojiID
		if strings.HasPrefix(match, "<a") {
			url += ".gif?size=48"
		} else {
			url += ".png?size=48"
		}

		msg.Emoji = append(msg.Emoji, lightning.Emoji{ID: emojiID, Name: emojiName, URL: url})

		return ":" + emojiName + ":"
	})
}
