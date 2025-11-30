package discord

import (
	"regexp"
	"strings"

	"codeberg.org/jersey/lightning/internal/cache"
	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
)

func discordToLightning(
	webhooks *cache.Expiring[string, bool],
	session *discordgo.Session,
	msg *discordgo.Message,
) *lightning.Message {
	if msg.Type != discordgo.MessageTypeDefault &&
		msg.Type != discordgo.MessageTypeReply &&
		msg.Type != discordgo.MessageTypeChatInputCommand &&
		msg.Type != discordgo.MessageTypeContextMenuCommand {
		return nil
	}

	if exists, _ := webhooks.Get(msg.WebhookID); exists {
		return nil
	}

	message := &lightning.Message{
		BaseMessage: lightning.BaseMessage{EventID: msg.ID, ChannelID: msg.ChannelID, Time: msg.Timestamp},
		Author:      discordToLightningAuthor(session, msg),
		Content:     discordToLightningForward(session, msg) + discordToLightningContent(session, msg),
		Embeds:      discordToLightningEmbeds(msg.Embeds),
		Attachments: discordToLightningAttachments(msg.Attachments, msg.StickerItems),
		RepliedTo:   discordToLightningReplies(msg.MessageReference),
	}

	message.Content = discordToLightningEmoji(message)

	return message
}

func discordToLightningAuthor(session *discordgo.Session, msg *discordgo.Message) *lightning.MessageAuthor {
	author := &lightning.MessageAuthor{
		ID:             msg.Author.ID,
		Username:       msg.Author.Username,
		Nickname:       msg.Author.DisplayName(),
		Color:          "#5865F2",
		ProfilePicture: msg.Author.AvatarURL(""),
	}

	if msg.GuildID == "" {
		return author
	}

	member := msg.Member
	if member == nil {
		if m, err := session.State.Member(msg.GuildID, msg.Author.ID); err == nil {
			member = m
		}
	}

	if member == nil {
		return author
	}

	member.User = msg.Author
	author.Nickname = member.DisplayName()
	author.ProfilePicture = member.AvatarURL("")

	return author
}

func discordToLightningForward(session *discordgo.Session, msg *discordgo.Message) string {
	if msg.MessageReference == nil || msg.MessageReference.MessageID == "" ||
		msg.MessageReference.Type != discordgo.MessageReferenceTypeForward || len(msg.MessageSnapshots) == 0 {
		return ""
	}

	out := ""

	for _, snapshot := range msg.MessageSnapshots {
		content := discordToLightningContent(session, snapshot.Message)
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

func discordToLightningContent(session *discordgo.Session, msg *discordgo.Message) string {
	content := defaultEmoji.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if e, ok := emoji.GetEmoji(match); ok {
			return e
		}

		return match
	})

	content = tenorURL.ReplaceAllStringFunc(content, func(match string) string {
		return "https://tenor.com/view/" + tenorURL.FindStringSubmatch(match)[1] + ".gif"
	})

	content = userMention.ReplaceAllStringFunc(content, func(match string) string {
		userID := userMention.FindStringSubmatch(match)[1]

		if msg.GuildID != "" {
			if m, err := session.State.Member(msg.GuildID, userID); err == nil {
				return "@" + m.DisplayName()
			}
		}

		if u, err := session.User(userID); err == nil {
			return "@" + u.DisplayName()
		}

		return "@" + userID
	})

	content = channelMention.ReplaceAllStringFunc(content, func(match string) string {
		channelID := channelMention.FindStringSubmatch(match)[1]

		if ch, err := session.State.Channel(channelID); err == nil {
			return "#" + ch.Name
		}

		return "#" + channelID
	})

	content = roleMention.ReplaceAllStringFunc(content, func(match string) string {
		roleID := roleMention.FindStringSubmatch(match)[1]

		if g, err := session.State.Guild(msg.GuildID); err == nil {
			for _, r := range g.Roles {
				if r.ID == roleID {
					return "@" + r.Name
				}
			}
		}

		return "@&" + roleID
	})

	return content
}

func discordToLightningEmbeds(embeds []*discordgo.MessageEmbed) []lightning.Embed {
	out := make([]lightning.Embed, 0, len(embeds))
	for _, original := range embeds {
		embed := lightning.Embed{
			Footer:      discordToLightningEmbedFooter(original.Footer),
			Author:      discordToLightningEmbedAuthor(original.Author),
			Fields:      discordToLightningEmbedFields(original.Fields),
			Timestamp:   original.Timestamp,
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

func discordToLightningEmbedFooter(original *discordgo.MessageEmbedFooter) *lightning.EmbedFooter {
	if original == nil {
		return nil
	}

	return &lightning.EmbedFooter{Text: original.Text, IconURL: original.IconURL}
}

func discordToLightningEmbedAuthor(original *discordgo.MessageEmbedAuthor) *lightning.EmbedAuthor {
	if original == nil {
		return nil
	}

	return &lightning.EmbedAuthor{Name: original.Name, URL: original.URL, IconURL: original.URL}
}

func discordToLightningEmbedFields(original []*discordgo.MessageEmbedField) []lightning.EmbedField {
	fields := make([]lightning.EmbedField, len(original))

	for i, f := range original {
		fields[i] = lightning.EmbedField{
			Name:   f.Name,
			Value:  f.Value,
			Inline: f.Inline,
		}
	}

	return fields
}

func discordToLightningAttachments(
	attachments []*discordgo.MessageAttachment,
	stickers []*discordgo.StickerItem,
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
		url := "https://cdn.discordapp.com/stickers/" + sticker.ID

		switch sticker.FormatType {
		case discordgo.StickerFormatTypePNG, discordgo.StickerFormatTypeAPNG:
			url += ".png"
		case discordgo.StickerFormatTypeGIF:
			url += ".gif"
		case discordgo.StickerFormatTypeLottie:
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

func discordToLightningReplies(reference *discordgo.MessageReference) []string {
	if reference == nil || reference.MessageID == "" || reference.Type != discordgo.MessageReferenceTypeDefault {
		return nil
	}

	return []string{reference.MessageID}
}

func discordToLightningEmoji(msg *lightning.Message) string {
	return emojiMention.ReplaceAllStringFunc(msg.Content, func(match string) string {
		parts := strings.Split(match, ":")
		if len(parts) < 3 {
			return match
		}

		emojiID := parts[2]
		emojiName := parts[1]

		url := "https://cdn.discordapp.com/emojis/" + emojiID
		if strings.HasPrefix(match, "<a") {
			url += ".gif?size=48"
		} else {
			url += ".png?size=48"
		}

		msg.Emoji = append(msg.Emoji, lightning.Emoji{ID: emojiID, Name: emojiName, URL: url})

		return ":" + emojiName + ":"
	})
}
