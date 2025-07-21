package discord

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *discordPlugin) getLightningMessage(message *discordgo.Message) *lightning.Message {
	if !slices.Contains([]discordgo.MessageType{
		discordgo.MessageTypeDefault,
		discordgo.MessageTypeGuildMemberJoin,
		discordgo.MessageTypeReply,
		discordgo.MessageTypeChatInputCommand,
		discordgo.MessageTypeContextMenuCommand,
	}, message.Type) {
		return nil
	}

	if exists, _ := p.webhookCache.Get(message.WebhookID); exists {
		return nil
	}

	if message.Type == discordgo.MessageTypeGuildMemberJoin {
		message.Content = "**joined on Discord**"
	}

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   message.ID,
			ChannelID: message.ChannelID,
			Time:      message.Timestamp,
		},
		Attachments: getLightningAttachments(message.Attachments, message.StickerItems),
		Author:      getLightningAuthor(p.discord, message),
		Content:     getLightningForward(p.discord, message) + getLightningContent(p.discord, message),
		Embeds:      getLightningEmbeds(message.Embeds),
		RepliedTo:   getLightningReplies(message),
	}
}

func getLightningAttachments(
	attachments []*discordgo.MessageAttachment,
	stickers []*discordgo.StickerItem,
) []lightning.Attachment {
	if len(attachments) == 0 {
		return nil
	}

	result := make([]lightning.Attachment, 0)
	for _, a := range attachments {
		result = append(result, lightning.Attachment{
			URL:  a.URL,
			Name: a.Filename,
			Size: int64(a.Size),
		})
	}

	for _, sticker := range stickers {
		stickerURL := ""

		switch sticker.FormatType {
		case discordgo.StickerFormatTypePNG, discordgo.StickerFormatTypeAPNG:
			stickerURL = "https://cdn.discordapp.com/stickers/" + sticker.ID + ".png"
		case discordgo.StickerFormatTypeLottie:
			stickerURL = "https://cdn.discordapp.com/stickers/" + sticker.ID + ".json"
		case discordgo.StickerFormatTypeGIF:
			stickerURL = "https://cdn.discordapp.com/stickers/" + sticker.ID + ".gif"
		}

		result = append(result, lightning.Attachment{
			URL:  stickerURL,
			Name: sticker.Name + " (Sticker)",
			Size: 0, // size information isn't available for stickers?
		})
	}

	return result
}

func getLightningAuthor(session *discordgo.Session, message *discordgo.Message) lightning.MessageAuthor {
	profilePicture := message.Author.AvatarURL("")
	author := lightning.MessageAuthor{
		ID:             message.Author.ID,
		Nickname:       message.Author.DisplayName(),
		Username:       message.Author.Username,
		Color:          "#5865F2",
		ProfilePicture: &profilePicture,
	}

	if message.GuildID == "" {
		return author
	}

	if message.Member == nil {
		member, err := session.State.Member(message.GuildID, message.Author.ID)
		if err != nil {
			return author
		}

		message.Member = member
	}

	if message.Member.GuildID == "" {
		message.Member.GuildID = message.GuildID
	}

	message.Member.User = message.Author
	author.Nickname = message.Member.DisplayName()
	profilePicture = message.Member.AvatarURL("")

	return author
}

var (
	userMention    = regexp.MustCompile(`<@!?(\d+)>`)
	channelMention = regexp.MustCompile(`<#(\d+)>`)
	roleMention    = regexp.MustCompile(`<@&(\d+)>`)
	emojiMention   = regexp.MustCompile(`<a?:\w+:(\d+)>`)
)

func getLightningContent(session *discordgo.Session, message *discordgo.Message) string {
	content := userMention.ReplaceAllStringFunc(message.Content, func(match string) string {
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

	content = roleMention.ReplaceAllStringFunc(content, func(match string) string {
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

	return emojiMention.ReplaceAllStringFunc(content, func(match string) string {
		return ":" + strings.Split(match, ":")[1] + ":"
	})
}

func getLightningEmbeds(embeds []*discordgo.MessageEmbed) []lightning.Embed {
	result := make([]lightning.Embed, 0, len(embeds))
	for _, embed := range embeds {
		lightningEmbed := lightning.Embed{
			Author:    getLightningEmbedAuthor(embed),
			Fields:    getLightningEmbedFields(embed),
			Footer:    getLightningFooter(embed),
			Timestamp: getLightningEmbedTime(embed),
		}

		if embed.Title != "" {
			lightningEmbed.Title = &embed.Title
		}

		if embed.URL != "" {
			lightningEmbed.URL = &embed.URL
		}

		if embed.Color != 0 {
			lightningEmbed.Color = &embed.Color
		}

		if embed.Description != "" {
			lightningEmbed.Description = &embed.Description
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
	if len(embed.Fields) > 0 {
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

	return nil
}

func getLightningEmbedTime(embed *discordgo.MessageEmbed) *time.Time {
	if embed.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, embed.Timestamp); err == nil {
			return &t
		}
	}

	return nil
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
		snapshot += "> *Forwarded:*\n> " + strings.ReplaceAll(getLightningContent(session, snap.Message), "\n", "\n> ")
	}

	return snapshot
}
