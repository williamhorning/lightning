package discord

import (
	"regexp"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
)

type discordSendable struct {
	messageSend

	Username  string
	AvatarURL string
}

func lightningToDiscordSendable(
	bot *client,
	msg *lightning.Message,
	opts *lightning.SendOptions,
) discordSendable {
	toSend := discordSendable{messageSend: messageSend{
		Content:         lightningToDiscordContent(bot, msg),
		Embeds:          lightningToDiscordEmbeds(msg.Embeds),
		AllowedMentions: lightningToDiscordAllowedMentions(opts),
		Components:      lightningToDiscordComponents(bot, msg, opts),
		Reference:       lightningToDiscordReference(msg),
		Files:           lightningToDiscordFiles(bot, msg),
	}}

	if msg.Author != nil {
		toSend.AvatarURL = msg.Author.ProfilePicture
		toSend.Username = msg.Author.Username
	}

	if toSend.Content == "" && len(toSend.Embeds) == 0 && len(toSend.Files) == 0 {
		toSend.Content = "_ _"
	}

	return toSend
}

func (msg *discordSendable) toWebhook() *webhookExecutePayload {
	return &webhookExecutePayload{
		Content: msg.Content, Username: msg.Username, AvatarURL: msg.AvatarURL, Embeds: msg.Embeds,
		Components: msg.Components, Files: msg.Files, AllowedMentions: msg.AllowedMentions, Flags: msg.Flags,
	}
}

func (msg *discordSendable) toInteraction() *interactionResponseData {
	return &interactionResponseData{
		Content: msg.Content, Embeds: msg.Embeds, Components: msg.Components, AllowedMentions: msg.AllowedMentions,
		Flags: msg.Flags,
	}
}

func (msg *discordSendable) toEdit() *messageEdit {
	return &messageEdit{
		Content: &msg.Content, Embeds: &msg.Embeds, Components: &msg.Components,
		AllowedMentions: msg.AllowedMentions, Flags: msg.Flags,
	}
}

func (msg *discordSendable) toWebhookEdit() *webhookEditMessagePayload {
	return &webhookEditMessagePayload{
		Content: &msg.Content, Embeds: msg.Embeds, Components: msg.Components, AllowedMentions: msg.AllowedMentions,
	}
}

var sendableEmojiRegex = regexp.MustCompile(`:\w+:`)

func lightningToDiscordContent(bot *client, msg *lightning.Message) string {
	return sendableEmojiRegex.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if name, ok := emoji.Emoji[match]; ok {
			return name
		}

		match = strings.ReplaceAll(match, ":", "")

		if guild, ok := bot.getGuild((*snowflake)(&msg.ChannelID)); ok {
			cached, ok := bot.getEmojiByName(&guild.ID, match)
			if ok {
				if cached.Animated {
					return "<a:" + string(cached.ID) + ":>"
				}

				return "<:" + string(cached.ID) + ":>"
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

func lightningToDiscordEmbeds(src []lightning.Embed) []embed {
	toImage := func(media *lightning.Media) *embedMedia {
		if media == nil {
			return nil
		}

		return &embedMedia{URL: media.URL, Width: &media.Width, Height: &media.Height}
	}

	embeds := make([]embed, len(src))

	for idx := range src {
		embeds[idx] = embed{
			URL: src[idx].URL, Title: src[idx].Title, Description: src[idx].Description, Timestamp: func() *time.Time {
				if t, err := time.Parse(time.RFC3339, src[idx].Timestamp); err == nil {
					return &t
				}

				return nil
			}(),
			Color: src[idx].Color, Image: toImage(src[idx].Image), Thumbnail: toImage(src[idx].Thumbnail),
			Video: toImage(src[idx].Video), Footer: func() *embedFooter {
				if src[idx].Footer == nil {
					return nil
				}

				return &embedFooter{Text: src[idx].Footer.Text, IconURL: src[idx].Footer.IconURL}
			}(),
			Author: func() *embedAuthor {
				if src[idx].Author == nil {
					return nil
				}

				return &embedAuthor{
					Name: src[idx].Author.Name, URL: src[idx].Author.URL, IconURL: src[idx].Author.IconURL,
				}
			}(),
			Fields: func() []embedField {
				out := make([]embedField, len(src[idx].Fields))

				for i := range src[idx].Fields {
					out[i] = embedField{
						Name: src[idx].Fields[i].Name, Value: src[idx].Fields[i].Value,
						Inline: src[idx].Fields[i].Inline,
					}
				}

				return out
			}(),
		}
	}

	return embeds
}

func lightningToDiscordAllowedMentions(opts *lightning.SendOptions) *allowedMentions {
	if opts.AllowEveryonePings {
		return nil
	}

	return &allowedMentions{
		Parse: []allowedMentionsType{allowedMentionRoles, allowedMentionUsers},
	}
}

func lightningToDiscordComponents(
	bot *client,
	msg *lightning.Message,
	opts *lightning.SendOptions,
) []component {
	if len(msg.RepliedTo) == 0 || opts.ChannelData == nil {
		return []component{}
	}

	replyMessage, ok := bot.getMessage(msg.ChannelID, msg.RepliedTo[0])
	if !ok {
		return []component{}
	}

	if replyMessage.GuildID == nil {
		if ch, ok := bot.getChannel(string(replyMessage.ChannelID)); ok && ch.GuildID != nil {
			replyMessage.GuildID = ch.GuildID
		} else {
			me := snowflake("@me")
			replyMessage.GuildID = &me
		}
	}

	author := discordToLightningAuthor(bot, replyMessage)

	url := "https://discord.com/channels/" + string(*replyMessage.GuildID) + "/" + msg.ChannelID + "/" +
		msg.RepliedTo[0]

	if bot.spacebar {
		url = strings.ReplaceAll(url, "discord.com", "fermi.chat")
	}

	btn := btnLink
	text := "↪️ " + author.Username + " > " +
		replyMessage.Content[:min(len(replyMessage.Content), 42)]

	return []component{{Type: compActionRow, Components: []component{{
		Type: compButton, Label: &text, Style: &btn, URL: &url,
	}}}}
}

func lightningToDiscordReference(msg *lightning.Message) *messageReference {
	if len(msg.RepliedTo) == 0 {
		return nil
	}

	return &messageReference{
		Type:            defaultReference,
		MessageID:       snowflake(msg.RepliedTo[0]),
		ChannelID:       snowflake(msg.ChannelID),
		FailIfNotExists: false,
	}
}
