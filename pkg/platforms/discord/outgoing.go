package discord

import (
	"net/http"
	"regexp"
	"strings"

	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
)

type discordSendable struct {
	discordgo.MessageSend

	Username  string
	AvatarURL string

	cancels []func()
}

func lightningToDiscordSendable(
	session *discordgo.Session,
	msg *lightning.Message,
	opts *lightning.SendOptions,
) *discordSendable {
	files, cancels := lightningToDiscordFiles(session, msg)

	toSend := &discordSendable{
		MessageSend: discordgo.MessageSend{
			Content:         lightningToDiscordContent(session, msg),
			Embeds:          lightningToDiscordEmbeds(msg.Embeds),
			AllowedMentions: lightningToDiscordAllowedMentions(opts),
			Components:      lightningToDiscordComponents(session, msg),
			Reference:       lightningToDiscordReference(msg),
			Files:           files,
		},
		cancels: cancels,
	}

	if msg.Author != nil {
		toSend.AvatarURL = msg.Author.ProfilePicture
		toSend.Username = msg.Author.Username
	}

	if toSend.Content == "" && len(toSend.Embeds) == 0 && len(toSend.Files) == 0 {
		toSend.Content = "_ _"
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
		Content: msg.Content, TTS: msg.TTS, Files: msg.Files, Embeds: msg.Embeds,
		AllowedMentions: msg.AllowedMentions, Flags: msg.Flags,
	}
}

var sendableEmojiRegex = regexp.MustCompile(`:\w+:`)

func lightningToDiscordContent(session *discordgo.Session, msg *lightning.Message) string {
	return sendableEmojiRegex.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if emoji, ok := emoji.Emoji[match]; ok {
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
	if opts.AllowEveryonePings {
		return nil
	}

	return &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeRoles, discordgo.AllowedMentionTypeUsers},
	}
}

func lightningToDiscordComponents(session *discordgo.Session, msg *lightning.Message) []discordgo.MessageComponent {
	if len(msg.RepliedTo) == 0 {
		return []discordgo.MessageComponent{}
	}

	replyMessage, err := session.ChannelMessage(msg.ChannelID, msg.RepliedTo[0])
	if err != nil {
		return []discordgo.MessageComponent{}
	}

	author := discordToLightningAuthor(session, replyMessage)

	baseURL := "https://discord.com/channels/"

	if session.Client.Transport != http.DefaultTransport {
		baseURL = "https://fermi.chat/channels/"
	}

	return []discordgo.MessageComponent{
		&discordgo.ActionsRow{Components: []discordgo.MessageComponent{&discordgo.Button{
			Label: "↪️ " + author.Username + " > " +
				replyMessage.ContentWithMentionsReplaced()[:min(len(replyMessage.ContentWithMentionsReplaced()), 42)],
			Style: discordgo.LinkButton,
			URL:   baseURL + replyMessage.GuildID + "/" + replyMessage.ChannelID + "/" + replyMessage.ID,
		}}},
	}
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
