package guilded

import (
	"encoding/json"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/williamhorning/lightning/pkg/lightning"
)

var usernameRegex = regexp.MustCompile(`(?ms)^[a-zA-Z0-9_ ()-]{1,25}$`)

func (p *guildedPlugin) getOutgoingMessage(message *lightning.Message, opts *lightning.SendOptions) *guildedPayload {
	base := &guildedPayload{
		Content:         message.Content,
		ReplyMessageIDs: message.RepliedTo,
		Embeds:          p.getOutgoingEmbeds(message, opts),
	}

	if message.Author != nil {
		if message.Author.ProfilePicture != nil {
			base.AvatarURL = *message.Author.ProfilePicture
		}

		base.Username = message.Author.ID

		if usernameRegex.MatchString(message.Author.Username) {
			base.Username = message.Author.Username
		}

		if usernameRegex.MatchString(message.Author.Nickname) {
			base.Username = message.Author.Nickname
		}
	}

	if len(base.Content) == 0 && len(base.Embeds) == 0 {
		base.Content = "\u2800"
	}

	if opts != nil && !opts.AllowEveryonePings {
		base.Content = strings.ReplaceAll(base.Content, "@everyone", "@\u2800everyone")
		base.Content = strings.ReplaceAll(base.Content, "@here", "@\u2800here")
	}

	return base
}

func (p *guildedPlugin) getOutgoingEmbeds(message *lightning.Message, opts *lightning.SendOptions) []guildedChatEmbed {
	guildedEmbeds := make([]guildedChatEmbed, 0)

	for _, embed := range message.Embeds {
		guildedEmbeds = append(guildedEmbeds, guildedChatEmbed{
			Title:       embed.Title,
			Description: embed.Description,
			Color:       embed.Color,
			Image:       getEmbedImage(&embed),
			Thumbnail:   getEmbedThumbnail(&embed),
			Footer:      getEmbedFooter(&embed),
			Author:      getEmbedAuthor(&embed),
			Fields:      getEmbedFields(&embed),
			Timestamp:   embed.Timestamp,
			URL:         embed.URL,
		})
	}

	if len(message.Attachments) > 0 {
		description := ""

		for _, attachment := range message.Attachments {
			description += "[" + attachment.Name + "](" + attachment.URL + ")\n"
		}

		guildedEmbeds = append(guildedEmbeds, guildedChatEmbed{
			Description: &description,
		})
	}

	if opts != nil && len(message.RepliedTo) > 0 {
		guildedEmbeds = p.appendReplyEmbed(guildedEmbeds, message)
	}

	return guildedEmbeds
}

func getEmbedImage(embed *lightning.Embed) *guildedChatEmbedMedia {
	if embed.Image != nil && embed.Image.URL != "" {
		return &guildedChatEmbedMedia{
			URL: &embed.Image.URL,
		}
	}

	return nil
}

func getEmbedThumbnail(embed *lightning.Embed) *guildedChatEmbedMedia {
	if embed.Thumbnail != nil && embed.Thumbnail.URL != "" {
		return &guildedChatEmbedMedia{
			URL: &embed.Thumbnail.URL,
		}
	}

	return nil
}

func getEmbedFooter(embed *lightning.Embed) *guildedChatEmbedFooter {
	if embed.Footer != nil {
		footer := &guildedChatEmbedFooter{
			Text: embed.Footer.Text,
		}
		if embed.Footer.IconURL != nil {
			footer.IconURL = embed.Footer.IconURL
		}

		return footer
	}

	return nil
}

func getEmbedAuthor(embed *lightning.Embed) *guildedChatEmbedAuthor {
	if embed.Author != nil {
		author := &guildedChatEmbedAuthor{
			Name: &embed.Author.Name,
			URL:  embed.Author.URL,
		}
		if embed.Author.IconURL != nil {
			author.IconURL = embed.Author.IconURL
		}

		return author
	}

	return nil
}

func getEmbedFields(embed *lightning.Embed) *[]guildedChatEmbedField {
	if len(embed.Fields) > 0 {
		convertedFields := make([]guildedChatEmbedField, len(embed.Fields))
		for i, field := range embed.Fields {
			convertedFields[i] = guildedChatEmbedField{
				Inline: &field.Inline,
				Name:   field.Name,
				Value:  field.Value,
			}
		}

		return &convertedFields
	}

	return nil
}

func (p *guildedPlugin) appendReplyEmbed(embeds []guildedChatEmbed, message *lightning.Message) []guildedChatEmbed {
	resp, err := guildedMakeRequest(p.token, "GET",
		"/channels/"+message.ChannelID+"/messages/"+message.RepliedTo[0], nil)
	if err != nil {
		return embeds
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return embeds
	}

	if resp.Body.Close() != nil {
		log.Println("guilded: failed to close request body when getting reply embed")
	}

	var messageResp guildedChatMessageResponse
	if json.Unmarshal(body, &messageResp) != nil {
		return embeds
	}

	author := p.getIncomingAuthor(&messageResp.Message)
	title := "reply to " + author.Nickname

	return append(embeds, guildedChatEmbed{
		Author: &guildedChatEmbedAuthor{
			Name:    &title,
			IconURL: author.ProfilePicture,
		},
		Description: messageResp.Message.Content,
	})
}
