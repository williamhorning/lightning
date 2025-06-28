package discord

import (
	"regexp"
	"slices"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning"
)

var allowedTypes = []discordgo.MessageType{0, 7, 19, 20, 23}

func getLightningMessage(s *discordgo.Session, m *discordgo.Message) *lightning.Message {
	if !slices.Contains(allowedTypes, m.Type) {
		return nil
	}

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   m.ID,
			ChannelID: m.ChannelID,
			Plugin:    "bolt-discord",
			Time:      m.Timestamp,
		},
		Attachments: getLightningAttachments(m.Attachments, m.StickerItems),
		Author:      getLightningAuthor(s, m),
		Content:     getLightningContent(s, m),
		Embeds:      getLightningEmbeds(m.Embeds),
		RepliedTo:   getLightningReplies(m),
	}
}

func getLightningAttachments(attachments []*discordgo.MessageAttachment, stickers []*discordgo.StickerItem) []lightning.Attachment {
	if len(attachments) == 0 {
		return nil
	}

	result := make([]lightning.Attachment, 0)
	for _, a := range attachments {
		result = append(result, lightning.Attachment{
			URL:  a.URL,
			Name: a.Filename,
			Size: float64(a.Size) / 1048576, // bytes -> MiB
		})
	}

	for _, sticker := range stickers {
		stickerURL := ""

		// Handle different sticker formats
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

func getLightningAuthor(s *discordgo.Session, m *discordgo.Message) lightning.MessageAuthor {
	profilePicture := m.Author.AvatarURL("")
	author := lightning.MessageAuthor{
		ID:             m.Author.ID,
		Nickname:       m.Author.DisplayName(),
		Username:       m.Author.Username,
		Color:          "#5865F2",
		ProfilePicture: &profilePicture,
	}

	if m.GuildID == "" {
		return author
	}

	if m.Member == nil {
		if member, err := s.State.Member(m.GuildID, m.Author.ID); err == nil {
			m.Member = member
		} else {
			return author
		}
	}

	m.Member.User = m.Author
	author.Nickname = m.Member.DisplayName()
	profilePicture = m.Member.AvatarURL("")

	return author
}

var (
	userMention    = regexp.MustCompile(`<@!?(\d+)>`)
	channelMention = regexp.MustCompile(`<#(\d+)>`)
	roleMention    = regexp.MustCompile(`<@&(\d+)>`)
	emojiMention   = regexp.MustCompile(`<a?:\w+:(\d+)>`)
)

func getLightningContent(s *discordgo.Session, m *discordgo.Message) string {
	content := userMention.ReplaceAllStringFunc(m.Content, func(match string) string {
		userID := userMention.FindStringSubmatch(match)[1]

		if m.GuildID != "" {
			if member, err := s.State.Member(m.GuildID, userID); err == nil {
				return "@" + member.DisplayName()
			}
		}

		if user, err := s.User(userID); err == nil {
			return "@" + user.DisplayName()
		}
		return "@" + match
	})

	content = channelMention.ReplaceAllStringFunc(content, func(match string) string {
		channelID := channelMention.FindStringSubmatch(match)[1]
		if channel, err := s.State.Channel(channelID); err == nil {
			return "#" + channel.Name
		}
		return "#" + match
	})

	content = roleMention.ReplaceAllStringFunc(content, func(match string) string {
		roleID := roleMention.FindStringSubmatch(match)[1]
		if guild, err := s.State.Guild(m.GuildID); err == nil {
			for _, role := range guild.Roles {
				if role.ID == roleID {
					return "@" + role.Name
				}
			}
		}
		return "@&" + match
	})

	return emojiMention.ReplaceAllStringFunc(content, func(match string) string {
		return emojiMention.FindStringSubmatch(match)[0]
	})
}

func getLightningEmbeds(embeds []*discordgo.MessageEmbed) []lightning.Embed {
	if len(embeds) == 0 {
		return nil
	}

	result := make([]lightning.Embed, 0, len(embeds))
	for _, e := range embeds {
		embed := lightning.Embed{}

		if e.Title != "" {
			embed.Title = &e.Title
		}

		if e.Timestamp != "" {
			if timestamp, err := strconv.ParseInt(e.Timestamp, 10, 64); err == nil {
				embed.Timestamp = &timestamp
			}
		}

		if e.URL != "" {
			embed.URL = &e.URL
		}

		if e.Color != 0 {
			embed.Color = &e.Color
		}

		if e.Description != "" {
			embed.Description = &e.Description
		}

		if e.Footer != nil {
			footer := &lightning.EmbedFooter{Text: e.Footer.Text}
			if e.Footer.IconURL != "" {
				footer.IconURL = &e.Footer.IconURL
			}
			embed.Footer = footer
		}

		if e.Image != nil && e.Image.URL != "" {
			embed.Image = &lightning.Media{URL: e.Image.URL}
		}

		if e.Thumbnail != nil && e.Thumbnail.URL != "" {
			embed.Thumbnail = &lightning.Media{URL: e.Thumbnail.URL}
		}

		if e.Author != nil {
			author := &lightning.EmbedAuthor{Name: e.Author.Name}
			if e.Author.URL != "" {
				author.URL = &e.Author.URL
			}
			if e.Author.IconURL != "" {
				author.IconURL = &e.Author.IconURL
			}
			embed.Author = author
		}

		if len(e.Fields) > 0 {
			fields := make([]lightning.EmbedField, len(e.Fields))
			for i, field := range e.Fields {
				fields[i] = lightning.EmbedField{
					Name:   field.Name,
					Value:  field.Value,
					Inline: field.Inline,
				}
			}
			embed.Fields = fields
		}

		result = append(result, embed)
	}

	return result
}

func getLightningReplies(m *discordgo.Message) []string {
	if m.MessageReference == nil || m.MessageReference.MessageID == "" ||
		m.MessageReference.Type != discordgo.MessageReferenceTypeDefault {
		return []string{}
	}
	return []string{m.MessageReference.MessageID}
}
