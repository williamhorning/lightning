package discord

import (
	"regexp"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/internal/emoji"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

type discordSendable struct {
	discord.MessageCreate

	Username  string
	AvatarURL string

	cancels []func()
}

func lightningToDiscordSendable(
	session *bot.Client,
	msg *lightning.Message,
	opts *lightning.SendOptions,
	cdn string,
) discordSendable {
	files, cancels := lightningToDiscordFiles(session, msg)

	toSend := discordSendable{
		MessageCreate: discord.MessageCreate{
			Content:          lightningToDiscordContent(session, msg),
			Embeds:           lightningToDiscordEmbeds(msg.Embeds),
			AllowedMentions:  lightningToDiscordAllowedMentions(opts),
			Components:       lightningToDiscordComponents(session, msg, cdn),
			MessageReference: lightningToDiscordReference(msg),
			Files:            files,
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

func (msg *discordSendable) toWebhook() discord.WebhookMessageCreate {
	return discord.WebhookMessageCreate{
		Content: msg.Content, Username: msg.Username, AvatarURL: msg.AvatarURL, Embeds: msg.Embeds,
		Components: msg.Components, Files: msg.Files, AllowedMentions: msg.AllowedMentions, Flags: msg.Flags,
	}
}

func (msg *discordSendable) toEdit() discord.MessageUpdate {
	return discord.MessageUpdate{
		Content: &msg.Content, Embeds: &msg.Embeds, Components: &msg.Components,
		AllowedMentions: msg.AllowedMentions, Flags: &msg.Flags,
	}
}

func (msg *discordSendable) toWebhookEdit() discord.WebhookMessageUpdate {
	return discord.WebhookMessageUpdate{
		Content: &msg.Content, Embeds: &msg.Embeds, Components: &msg.Components, AllowedMentions: msg.AllowedMentions,
	}
}

var sendableEmojiRegex = regexp.MustCompile(`:\w+:`)

func lightningToDiscordContent(session *bot.Client, msg *lightning.Message) string {
	return sendableEmojiRegex.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if name, ok := emoji.Emoji[match]; ok {
			return name
		}

		match = strings.ReplaceAll(match, ":", "")

		matchID, err := snowflake.Parse(msg.RepliedTo[0])
		if err != nil {
			return match
		}

		channelID, err := snowflake.Parse(msg.ChannelID)
		if err != nil {
			return match
		}

		channel, ok := session.Caches.Channel(channelID)
		if ok {
			cached, ok := session.Caches.Emoji(channel.GuildID(), matchID)
			if ok {
				return cached.Mention()
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

func lightningToDiscordEmbeds(src []lightning.Embed) []discord.Embed {
	toImage := func(media *lightning.Media) *discord.EmbedResource {
		if media == nil {
			return nil
		}

		return &discord.EmbedResource{URL: media.URL, Width: media.Width, Height: media.Height}
	}

	embeds := make([]discord.Embed, len(src))

	for idx := range src {
		embed := src[idx]
		embeds[idx] = discord.Embed{
			URL: src[idx].URL, Title: src[idx].Title, Description: src[idx].Description, Timestamp: func() *time.Time {
				if t, err := time.Parse(time.RFC3339, src[idx].Timestamp); err == nil {
					return &t
				}

				return nil
			}(),
			Color: src[idx].Color, Image: toImage(src[idx].Image), Thumbnail: toImage(src[idx].Thumbnail),
			Video: toImage(embed.Video), Footer: func() *discord.EmbedFooter {
				if src[idx].Footer == nil {
					return nil
				}

				return &discord.EmbedFooter{Text: src[idx].Footer.Text, IconURL: src[idx].Footer.IconURL}
			}(),
			Author: func() *discord.EmbedAuthor {
				if src[idx].Author == nil {
					return nil
				}

				return &discord.EmbedAuthor{
					Name: src[idx].Author.Name, URL: src[idx].Author.URL, IconURL: src[idx].Author.IconURL,
				}
			}(),
			Fields: func() []discord.EmbedField {
				out := make([]discord.EmbedField, len(src[idx].Fields))

				for i := range src[idx].Fields {
					out[i] = discord.EmbedField{
						Name: src[idx].Fields[i].Name, Value: src[idx].Fields[i].Value,
						Inline: &src[idx].Fields[i].Inline,
					}
				}

				return out
			}(),
		}
	}

	return embeds
}

func lightningToDiscordAllowedMentions(opts *lightning.SendOptions) *discord.AllowedMentions {
	if opts.AllowEveryonePings {
		return nil
	}

	return &discord.AllowedMentions{
		Parse: []discord.AllowedMentionType{discord.AllowedMentionTypeRoles, discord.AllowedMentionTypeUsers},
	}
}

func lightningToDiscordComponents(session *bot.Client, msg *lightning.Message, cdn string) []discord.LayoutComponent {
	if len(msg.RepliedTo) == 0 {
		return []discord.LayoutComponent{}
	}

	replyID, err := snowflake.Parse(msg.RepliedTo[0])
	if err != nil {
		return []discord.LayoutComponent{}
	}

	channelID, err := snowflake.Parse(msg.ChannelID)
	if err != nil {
		return []discord.LayoutComponent{}
	}

	replyMessage, err := session.Rest.GetMessage(channelID, replyID)
	if err != nil {
		return []discord.LayoutComponent{}
	}

	author := discordToLightningAuthor(session, replyMessage, cdn)

	url := replyMessage.JumpURL()

	if cdn != "cdn.discordapp.com" {
		url = strings.ReplaceAll(url, "discord.com", "fermi.chat")
	}

	return []discord.LayoutComponent{
		discord.ActionRowComponent{Components: []discord.InteractiveComponent{discord.ButtonComponent{
			Label: "↪️ " + author.Username + " > " +
				replyMessage.Content[:min(len(replyMessage.Content), 42)],
			Style: discord.ButtonStyleLink, URL: url,
		}}},
	}
}

func lightningToDiscordReference(msg *lightning.Message) *discord.MessageReference {
	if len(msg.RepliedTo) == 0 {
		return nil
	}

	replyID, err := snowflake.Parse(msg.RepliedTo[0])
	if err != nil {
		return nil
	}

	channelID, err := snowflake.Parse(msg.ChannelID)
	if err != nil {
		return nil
	}

	return &discord.MessageReference{
		Type:            discord.MessageReferenceTypeDefault,
		MessageID:       &replyID,
		ChannelID:       &channelID,
		FailIfNotExists: false,
	}
}

func (p *discordPlugin) getOutgoingChannel(
	message *lightning.Message,
	opts *lightning.SendOptions,
) (snowflake.ID, error) {
	if opts.ChannelData != nil {
		whID, err := snowflake.Parse(opts.ChannelData["id"])
		if err != nil {
			return 0, &snowflakeError{opts.ChannelData["id"], true}
		}

		p.webhooks.Set(whID, true)

		return whID, nil
	}

	if opts.CommandResponse {
		id, err := snowflake.Parse(opts.CommandUser)
		if err != nil {
			return 0, &snowflakeError{opts.CommandUser, true}
		}

		channel, err := p.session.Rest.CreateDMChannel(id)
		if err != nil {
			return 0, getError(err, "Failed to create DM channel for "+opts.CommandUser+" in command response")
		}

		message.RepliedTo = nil

		return channel.ID(), nil
	}

	channelID, err := snowflake.Parse(message.ChannelID)
	if err != nil {
		return 0, &snowflakeError{message.ChannelID, true}
	}

	return channelID, nil
}
