package guilded

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

var usernameRegex = regexp.MustCompile(`(?ms)^[a-zA-Z0-9_ ()-]{1,25}$`)

func getValidUsername(author lightning.MessageAuthor) string {
	if usernameRegex.MatchString(author.Nickname) {
		return author.Nickname
	} else if usernameRegex.MatchString(author.Username) {
		return author.Username
	} else {
		return author.ID
	}
}

func getOutgoingMessage(message lightning.Message, opts *lightning.SendOptions, token string) *guildedPayload {
	base := &guildedPayload{
		Content:         message.Content,
		AvatarURL:       *message.Author.ProfilePicture,
		Username:        getValidUsername(message.Author),
		ReplyMessageIDs: message.RepliedTo,
		Embeds:          getOutgoingEmbeds(message, opts != nil, token),
	}

	if len(base.Content) <= 0 && len(base.Embeds) <= 0 {
		base.Content = "\u2800"
	}

	if opts != nil && !opts.AllowEveryonePings {
		base.Content = strings.ReplaceAll(base.Content, "@everyone", "@\u2800everyone")
		base.Content = strings.ReplaceAll(base.Content, "@here", "@\u2800here")
	}

	return base
}
func getOutgoingEmbeds(message lightning.Message, incl bool, token string) []guildedChatEmbed {
	guildedEmbeds := make([]guildedChatEmbed, 0)
	for _, embed := range message.Embeds {
		var image *guildedChatEmbedMedia
		if embed.Image != nil && embed.Image.URL != "" {
			image = &guildedChatEmbedMedia{
				Url: &embed.Image.URL,
			}
		}

		var thumbnail *guildedChatEmbedMedia
		if embed.Thumbnail != nil && embed.Thumbnail.URL != "" {
			thumbnail = &guildedChatEmbedMedia{
				Url: &embed.Thumbnail.URL,
			}
		}
		var timestamp *time.Time
		if embed.Timestamp != nil {
			t := time.Unix(*embed.Timestamp, 0)
			timestamp = &t
		}

		var footer *guildedChatEmbedFooter
		if embed.Footer != nil {
			footer = &guildedChatEmbedFooter{
				Text: embed.Footer.Text,
			}
			if embed.Footer.IconURL != nil {
				footer.IconUrl = embed.Footer.IconURL
			}
		}

		var author *guildedChatEmbedAuthor
		if embed.Author != nil {
			author = &guildedChatEmbedAuthor{
				Name: &embed.Author.Name,
				Url:  embed.Author.URL,
			}
			if embed.Author.IconURL != nil {
				author.IconUrl = embed.Author.IconURL
			}
		}

		var fields *[]guildedChatEmbedField

		if len(embed.Fields) > 0 {
			convertedFields := make([]guildedChatEmbedField, len(embed.Fields))

			for i, field := range embed.Fields {
				convertedFields[i] = guildedChatEmbedField{
					Inline: &field.Inline,
					Name:   field.Name,
					Value:  field.Value,
				}
			}

			fields = &convertedFields
		}

		guildedEmbeds = append(guildedEmbeds, guildedChatEmbed{
			Title:       embed.Title,
			Description: embed.Description,
			Color:       embed.Color,
			Image:       image,
			Thumbnail:   thumbnail,
			Footer:      footer,
			Author:      author,
			Fields:      fields,
			Timestamp:   timestamp,
			Url:         embed.URL,
		})
	}

	if len(message.Attachments) > 0 {
		title := "Attachments"
		attachmentStr := ""

		for _, attachment := range message.Attachments {
			attachmentStr += "[" + attachment.Name + "](" + attachment.URL + ")\n"
		}

		guildedEmbeds = append(guildedEmbeds, guildedChatEmbed{
			Title:       &title,
			Description: &attachmentStr,
		})
	}
	if incl && len(message.RepliedTo) > 0 {
		resp, err := guildedMakeRequest(token, "GET", "/channels/"+message.ChannelID+"/messages/"+message.RepliedTo[0], nil)

		if err == nil {
			var messageResp guildedChatMessageResponse

			body, err := io.ReadAll(resp.Body)
			if err == nil && json.Unmarshal(body, &messageResp) == nil {
				author := getIncomingAuthor(token, &messageResp.Message)
				title := "reply to " + author.Nickname
				guildedEmbeds = append(guildedEmbeds, guildedChatEmbed{
					Author: &guildedChatEmbedAuthor{
						Name:    &title,
						IconUrl: author.ProfilePicture,
					},
					Description: messageResp.Message.Content,
				})
			}
		}
	}

	return guildedEmbeds
}
