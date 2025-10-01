package discord

import (
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *discordPlugin) getLightningMessage(msg *discordgo.Message) *lightning.Message {
	if msg.Type != discordgo.MessageTypeDefault &&
		msg.Type != discordgo.MessageTypeReply &&
		msg.Type != discordgo.MessageTypeChatInputCommand &&
		msg.Type != discordgo.MessageTypeContextMenuCommand {
		return nil
	}

	if exists, _ := p.webhookCache.Get(msg.WebhookID); exists {
		return nil
	}

	message := &lightning.Message{
		BaseMessage: lightning.BaseMessage{EventID: msg.ID, ChannelID: msg.ChannelID, Time: &msg.Timestamp},
		Attachments: getLightningAttachments(msg.Attachments, msg.StickerItems),
		Author:      getLightningAuthor(p.discord, msg),
		Content:     getLightningForward(p.discord, msg) + getLightningContent(p.discord, msg),
		Embeds:      getLightningEmbeds(msg.Embeds),
		RepliedTo:   getLightningReplies(msg),
	}

	message.Content = replaceIncomingEmoji(message)

	return message
}

func getLightningAttachments(
	attachments []*discordgo.MessageAttachment,
	stickers []*discordgo.StickerItem,
) []lightning.Attachment {
	result := make([]lightning.Attachment, 0, len(attachments)+len(stickers))
	for _, a := range attachments {
		result = append(result, lightning.Attachment{URL: a.URL, Name: a.Filename, Size: int64(a.Size)})
	}

	for _, sticker := range stickers {
		stickerURL := "https://cdn.discordapp.com/stickers/" + sticker.ID

		switch sticker.FormatType {
		case discordgo.StickerFormatTypePNG, discordgo.StickerFormatTypeAPNG:
			stickerURL += ".png"
		case discordgo.StickerFormatTypeLottie:
			stickerURL += ".json"
		case discordgo.StickerFormatTypeGIF:
			stickerURL += ".gif"
		default:
		}

		result = append(result, lightning.Attachment{URL: stickerURL + "?size=160", Name: sticker.Name, Size: 0})
	}

	return result
}

func getLightningAuthor(session *discordgo.Session, message *discordgo.Message) *lightning.MessageAuthor {
	profilePicture := message.Author.AvatarURL("")
	author := lightning.MessageAuthor{
		ID:             message.Author.ID,
		Nickname:       message.Author.DisplayName(),
		Username:       message.Author.Username,
		Color:          "#5865F2",
		ProfilePicture: &profilePicture,
	}

	if message.GuildID == "" {
		return &author
	}

	if message.Member == nil {
		member, err := session.State.Member(message.GuildID, message.Author.ID)
		if err != nil {
			return &author
		}

		message.Member = member
	}

	if message.Member.GuildID == "" {
		message.Member.GuildID = message.GuildID
	}

	message.Member.User = message.Author
	author.Nickname = message.Member.DisplayName()
	profilePicture = message.Member.AvatarURL("")
	author.ProfilePicture = &profilePicture

	return &author
}

var (
	tenorURL       = regexp.MustCompile(`https://tenor\.com/view/[^/]+-(\d+).*`)
	userMention    = regexp.MustCompile(`<@!?(\d+)>`)
	channelMention = regexp.MustCompile(`<#(\d+)>`)
	roleMention    = regexp.MustCompile(`<@&(\d+)>`)
	emojiMention   = regexp.MustCompile(`<a?:\w+:(\d+)>`)
)

func getLightningContent(session *discordgo.Session, message *discordgo.Message) string {
	content := tenorURL.ReplaceAllStringFunc(message.Content, func(match string) string {
		return "https://tenor.com/view/" + tenorURL.FindStringSubmatch(match)[1] + ".gif"
	})

	content = userMention.ReplaceAllStringFunc(content, func(match string) string {
		userID := userMention.FindStringSubmatch(match)[1]

		if message.GuildID != "" {
			if member, err := session.State.Member(message.GuildID, userID); err == nil {
				return "@" + member.DisplayName()
			}
		}

		if user, err := session.User(userID); err == nil {
			return "@" + user.DisplayName()
		}

		return "@" + userID
	})

	content = channelMention.ReplaceAllStringFunc(content, func(match string) string {
		channelID := channelMention.FindStringSubmatch(match)[1]
		if channel, err := session.State.Channel(channelID); err == nil {
			return "#" + channel.Name
		}

		return "#" + channelID
	})

	return roleMention.ReplaceAllStringFunc(content, func(match string) string {
		roleID := roleMention.FindStringSubmatch(match)[1]

		if guild, err := session.State.Guild(message.GuildID); err == nil {
			for _, role := range guild.Roles {
				if role.ID == roleID {
					return "@" + role.Name
				}
			}
		}

		return "@&" + roleID
	})
}

func replaceIncomingEmoji(msg *lightning.Message) string {
	return emojiMention.ReplaceAllStringFunc(msg.Content, func(match string) string {
		split := strings.Split(match, ":")
		url := "https://cdn.discordapp.com/emojis/" + split[2]

		if strings.Contains(match, "<a") {
			url += ".gif?size=48"
		} else {
			url += ".png?size=48"
		}

		msg.Emoji = append(msg.Emoji, lightning.Emoji{
			ID:   split[2],
			Name: split[1],
			URL:  &url,
		})

		return ":" + split[1] + ":"
	})
}

func getLightningEmbeds(embeds []*discordgo.MessageEmbed) []lightning.Embed {
	result := make([]lightning.Embed, 0, len(embeds))
	for _, embed := range embeds {
		lightningEmbed := lightning.Embed{
			Author:      getLightningEmbedAuthor(embed),
			Fields:      getLightningEmbedFields(embed),
			Footer:      getLightningFooter(embed),
			Timestamp:   toPtr(embed.Timestamp),
			Title:       toPtr(embed.Title),
			URL:         toPtr(embed.URL),
			Description: toPtr(embed.Description),
			Color:       toPtr(embed.Color),
		}

		if embed.Image != nil && embed.Image.URL != "" {
			lightningEmbed.Image = &lightning.Media{URL: embed.Image.URL}
		}

		if embed.Thumbnail != nil && embed.Thumbnail.URL != "" {
			lightningEmbed.Thumbnail = &lightning.Media{URL: embed.Thumbnail.URL}
		}

		result = append(result, lightningEmbed)
	}

	return result
}

func toPtr[T comparable](val T) *T {
	var t T

	if val == t {
		return nil
	}

	return &val
}

func getLightningFooter(embed *discordgo.MessageEmbed) *lightning.EmbedFooter {
	if embed.Footer != nil {
		footer := &lightning.EmbedFooter{Text: embed.Footer.Text}
		if embed.Footer.IconURL != "" {
			footer.IconURL = &embed.Footer.IconURL
		}

		return footer
	}

	return nil
}

func getLightningEmbedAuthor(embed *discordgo.MessageEmbed) *lightning.EmbedAuthor {
	if embed.Author != nil {
		author := &lightning.EmbedAuthor{Name: embed.Author.Name}
		if embed.Author.URL != "" {
			author.URL = &embed.Author.URL
		}

		if embed.Author.IconURL != "" {
			author.IconURL = &embed.Author.IconURL
		}

		return author
	}

	return nil
}

func getLightningEmbedFields(embed *discordgo.MessageEmbed) []lightning.EmbedField {
	fields := make([]lightning.EmbedField, len(embed.Fields))
	for i, field := range embed.Fields {
		fields[i] = lightning.EmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: field.Inline,
		}
	}

	return fields
}

func getLightningReplies(message *discordgo.Message) []string {
	if message.MessageReference == nil || message.MessageReference.MessageID == "" ||
		message.MessageReference.Type != discordgo.MessageReferenceTypeDefault {
		return []string{}
	}

	return []string{message.MessageReference.MessageID}
}

func getLightningForward(session *discordgo.Session, message *discordgo.Message) string {
	if message.MessageReference == nil || message.MessageReference.MessageID == "" ||
		message.MessageReference.Type != discordgo.MessageReferenceTypeForward ||
		message.MessageSnapshots == nil || len(message.MessageSnapshots) == 0 {
		return ""
	}

	snapshot := ""

	for _, snap := range message.MessageSnapshots {
		snapshot += "> " + strings.ReplaceAll(getLightningContent(session, snap.Message), "\n", "\n> ")
	}

	return snapshot
}
