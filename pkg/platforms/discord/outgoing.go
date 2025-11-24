package discord

import (
	"regexp"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/internal/emoji"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type discordSendable struct {
	discordgo.MessageSend

	Username  string
	AvatarURL string
}

func lightningToDiscordSendable(
	session *discordgo.Session,
	msg *lightning.Message,
	opts *lightning.SendOptions,
) []discordSendable {
	toSend := []discordSendable{{
		MessageSend: discordgo.MessageSend{
			Content:         lightningToDiscordContent(session, msg),
			Embeds:          lightningToDiscordEmbeds(msg.Embeds),
			AllowedMentions: lightningToDiscordAllowedMentions(opts),
			Components:      lightningToDiscordComponents(session, msg),
			Reference:       lightningToDiscordReference(msg),
			Files:           lightningToDiscordFiles(session, msg),
		},
	}}

	if msg.Author != nil {
		toSend[0].AvatarURL = msg.Author.ProfilePicture
		toSend[0].Username = msg.Author.Nickname
	}

	if len(toSend[0].Content) > 2000 {
		leftover := toSend[0].Content[2000:]

		toSend[0].Content = toSend[0].Content[:2000]

		for chunk := range slices.Chunk([]byte(leftover), 2000) {
			toSend = append(toSend, discordSendable{
				AvatarURL:   toSend[0].AvatarURL,
				Username:    toSend[0].Username,
				MessageSend: discordgo.MessageSend{Content: string(chunk)},
			})
		}
	}

	for i := range toSend {
		if toSend[i].Content == "" && len(toSend[i].Embeds) == 0 && len(toSend[i].Files) == 0 {
			toSend[i].Content = "_ _"
		}
	}

	return toSend
}

func (msg *discordSendable) toWebhook() *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		Content: msg.Content, Username: msg.Username, AvatarURL: msg.AvatarURL, TTS: msg.TTS, Files: msg.Files,
		Components: msg.Components, Embeds: msg.Embeds, AllowedMentions: msg.AllowedMentions, Flags: msg.Flags,
	}
}

func (msg *discordSendable) toWebhookEdit() *discordgo.WebhookEdit {
	return &discordgo.WebhookEdit{
		Content: &msg.Content, Components: &msg.Components, Embeds: &msg.Embeds,
		Files: msg.Files, AllowedMentions: msg.AllowedMentions,
	}
}

func (msg *discordSendable) toInteractionResponseData() *discordgo.InteractionResponseData {
	return &discordgo.InteractionResponseData{
		Content: msg.Content, TTS: msg.TTS, Files: msg.Files, Components: msg.Components, Embeds: msg.Embeds,
		AllowedMentions: msg.AllowedMentions, Flags: msg.Flags,
	}
}

var sendableEmojiRegex = regexp.MustCompile(`:\w+:`)

func lightningToDiscordContent(session *discordgo.Session, msg *lightning.Message) string {
	return sendableEmojiRegex.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if emoji, ok := emoji.GetEmoji(match); ok {
			return emoji
		}

		match = strings.ReplaceAll(match, ":", "")

		channel, err := session.State.Channel(msg.ChannelID)
		if err == nil {
			serverEmoji, err := session.GuildEmojis(channel.GuildID)
			if err == nil {
				for _, emoji := range serverEmoji {
					if emoji.Name == match {
						return emoji.MessageFormat()
					}
				}
			}
		}

		for _, emoji := range msg.Emoji {
			if emoji.Name == match {
				return "[" + emoji.Name + "](" + emoji.URL + ")"
			}
		}

		return match
	})
}

func lightningToDiscordEmbeds(src []lightning.Embed) []*discordgo.MessageEmbed {
	toImage := func(media *lightning.Media) *discordgo.MessageEmbedImage {
		if media == nil {
			return nil
		}

		return &discordgo.MessageEmbedImage{URL: media.URL, Width: media.Width, Height: media.Height}
	}

	toThumbnail := func(media *lightning.Media) *discordgo.MessageEmbedThumbnail {
		if media == nil {
			return nil
		}

		return &discordgo.MessageEmbedThumbnail{URL: media.URL, Width: media.Width, Height: media.Height}
	}

	embeds := make([]*discordgo.MessageEmbed, len(src))
	for idx, embed := range src {
		embeds[idx] = &discordgo.MessageEmbed{
			URL: embed.URL, Title: embed.Title, Description: embed.Description, Timestamp: embed.Timestamp,
			Color: embed.Color, Image: toImage(embed.Image), Thumbnail: toThumbnail(embed.Image),
			Video: func() *discordgo.MessageEmbedVideo {
				if embed.Video == nil {
					return nil
				}

				return &discordgo.MessageEmbedVideo{
					URL: embed.Video.URL, Width: embed.Video.Width, Height: embed.Video.Height,
				}
			}(),
			Footer: func() *discordgo.MessageEmbedFooter {
				if embed.Footer == nil {
					return nil
				}

				return &discordgo.MessageEmbedFooter{Text: embed.Footer.Text, IconURL: embed.Footer.IconURL}
			}(),
			Author: func() *discordgo.MessageEmbedAuthor {
				if embed.Author == nil {
					return nil
				}

				return &discordgo.MessageEmbedAuthor{
					Name: embed.Author.Name, URL: embed.Author.URL, IconURL: embed.Author.IconURL,
				}
			}(),
			Fields: func() []*discordgo.MessageEmbedField {
				out := make([]*discordgo.MessageEmbedField, len(embed.Fields))

				for i, f := range embed.Fields {
					out[i] = &discordgo.MessageEmbedField{Name: f.Name, Value: f.Value, Inline: f.Inline}
				}

				return out
			}(),
		}
	}

	return embeds
}

func lightningToDiscordAllowedMentions(opts *lightning.SendOptions) *discordgo.MessageAllowedMentions {
	if opts == nil || opts.AllowEveryonePings {
		return nil
	}

	return &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeRoles, discordgo.AllowedMentionTypeUsers},
	}
}

func lightningToDiscordComponents(session *discordgo.Session, msg *lightning.Message) []discordgo.MessageComponent {
	if len(msg.RepliedTo) == 0 {
		return nil
	}

	replyMessage, err := session.State.Message(msg.ChannelID, msg.RepliedTo[0])
	if err != nil {
		return nil
	}

	return []discordgo.MessageComponent{discordgo.Button{
		Label: "reply to " + replyMessage.Member.DisplayName(),
		Style: discordgo.LinkButton,
		URL: "https://discord.com/channels/" + replyMessage.GuildID + "/" + replyMessage.ChannelID +
			"/" + replyMessage.ID,
	}}
}

func lightningToDiscordReference(msg *lightning.Message) *discordgo.MessageReference {
	if len(msg.RepliedTo) == 0 {
		return nil
	}

	return &discordgo.MessageReference{
		Type:      discordgo.MessageReferenceTypeDefault,
		MessageID: msg.RepliedTo[0],
		ChannelID: msg.ChannelID,
	}
}
