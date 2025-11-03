package guilded

import (
	"encoding/json"
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
		Embeds:          p.appendReplyEmbed(message),
	}

	if message.Author != nil {
		if message.Author.ProfilePicture != "" {
			base.AvatarURL = message.Author.ProfilePicture
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

func (p *guildedPlugin) appendReplyEmbed(message *lightning.Message) []lightning.Embed {
	resp, err := guildedMakeRequest(p.token, "GET",
		"/channels/"+message.ChannelID+"/messages/"+message.RepliedTo[0], nil)
	if err != nil {
		return message.Embeds
	}

	if resp.Body.Close() != nil {
		log.Println("guilded: failed to close request body when getting reply embed")
	}

	var messageResp guildedChatMessageWrapper
	if json.NewDecoder(resp.Body).Decode(&messageResp) != nil {
		return message.Embeds
	}

	author := p.getIncomingAuthor(&messageResp.Message)
	title := "reply to " + author.Nickname

	return append(message.Embeds, lightning.Embed{
		Author: &lightning.EmbedAuthor{
			Name:    title,
			IconURL: author.ProfilePicture,
		},
		Description: messageResp.Message.Content,
	})
}
