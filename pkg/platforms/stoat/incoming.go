package stoat

import (
	"regexp"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/internal/v2/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *stoatPlugin) getIncomingMessage(message stoat.Message) *lightning.Message {
	if message.Author == p.self.ID && message.Masquerade != nil {
		return nil
	}

	msg := &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   message.ID,
			ChannelID: message.Channel,
			Time:      getLightningTime(message),
		},
		Attachments: getLightningAttachment(message.Attachments),
		Author:      p.getLightningAuthor(message.Author, message.Channel, message.Masquerade),
		Embeds:      getLightningEmbeds(message.Embeds),
		RepliedTo:   message.Replies,
	}

	msg.Content = replaceSpoilers(message.Content)
	msg.Content = p.replaceEmojis(msg)
	msg.Content = p.replaceMentions(msg.ChannelID, msg.Content)
	msg.Content = p.replaceChannels(msg.Content)

	return msg
}

func getLightningTime(message stoat.Message) time.Time {
	if !message.Edited.IsZero() {
		return message.Edited
	}

	msgID, err := ulid.Parse(message.ID)
	if err != nil {
		return time.Now()
	}

	return msgID.Timestamp()
}

func getLightningAttachment(attachments []stoat.File) []lightning.Attachment {
	result := make([]lightning.Attachment, len(attachments))
	for i, att := range attachments {
		result[i] = lightning.Attachment{
			URL:  getURL(&att),
			Name: att.Filename,
			Size: int64(att.Size),
		}
	}

	return result
}

func (p *stoatPlugin) getLightningAuthor(
	authorID string,
	channelID string,
	masquerade *stoat.Masquerade,
) *lightning.MessageAuthor {
	author := lightning.MessageAuthor{
		ID:       authorID,
		Username: "StoatUser",
		Nickname: "Stoat User",
		Color:    "#8C24EC",
	}

	user := stoat.Get(p.session, "/users/"+authorID, authorID, &p.session.UserCache)
	if user == nil {
		return applyMasquerade(author, masquerade)
	}

	author.Username = user.Username
	author.Nickname = user.Username

	if user.Avatar != nil {
		author.ProfilePicture = getURL(user.Avatar)
	}

	p.setServerMember(&author, authorID, channelID)

	return applyMasquerade(author, masquerade)
}

func (p *stoatPlugin) setServerMember(author *lightning.MessageAuthor, authorID, channelID string) {
	channel := stoat.Get(p.session, "/channels/"+channelID, channelID, &p.session.ChannelCache)
	if channel == nil || channel.ChannelType != "TextChannel" || channel.Server == nil {
		return
	}

	member := stoat.Get(p.session, "/servers/"+*channel.Server+"/members/"+authorID, *channel.Server+"-"+authorID,
		&p.session.MemberCache)
	if member == nil {
		return
	}

	if member.Nickname != nil {
		author.Nickname = *member.Nickname
	}

	if member.Avatar != nil {
		author.ProfilePicture = getURL(member.Avatar)
	}
}

func getURL(file *stoat.File) string {
	return "https://cdn.stoatusercontent.com/" + file.Tag + "/" + file.ID
}

func applyMasquerade(author lightning.MessageAuthor, masquerade *stoat.Masquerade) *lightning.MessageAuthor {
	if masquerade == nil {
		return &author
	}

	if masquerade.Name != "" {
		author.Nickname = masquerade.Name
	}

	if masquerade.Colour != "" {
		author.Color = masquerade.Colour
	}

	if masquerade.Avatar != "" {
		author.ProfilePicture = masquerade.Avatar
	}

	return &author
}

var (
	stoatSpoilerRegex = regexp.MustCompile(`!!(.+?)!!`)
	spoilerRegex      = regexp.MustCompile(`\|\|(.+?)\|\|`)
	emojiRegex        = regexp.MustCompile(":([0-7][0-9A-HJKMNP-TV-Z]{25}):")
	mentionRegex      = regexp.MustCompile("<@([0-7][0-9A-HJKMNP-TV-Z]{25})>")
	channelRegex      = regexp.MustCompile("<#([0-7][0-9A-HJKMNP-TV-Z]{25})>")
)

func replaceSpoilers(content string) string {
	return stoatSpoilerRegex.ReplaceAllStringFunc(content, func(match string) string {
		return "||" + match[2:len(match)-2] + "||"
	})
}

func (p *stoatPlugin) replaceEmojis(message *lightning.Message) string {
	return emojiRegex.ReplaceAllStringFunc(message.Content, func(match string) string {
		if emojiID := extractID(match, emojiRegex); emojiID != "" {
			emoji := stoat.Get(p.session, "/custom/emoji/"+emojiID, emojiID, &p.session.EmojiCache)

			if emoji == nil {
				return match
			}

			message.Emoji = append(message.Emoji, lightning.Emoji{
				URL:  "https://cdn.stoatusercontent.com/emojis/" + emoji.ID,
				ID:   emoji.ID,
				Name: emoji.Name,
			})

			return ":" + emoji.Name + ":"
		}

		return match
	})
}

func (p *stoatPlugin) replaceMentions(channelID string, content string) string {
	return mentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		userID := extractID(match, mentionRegex)
		if userID == "" {
			return match
		}

		user := stoat.Get(p.session, "/users/"+userID, userID, &p.session.UserCache)
		if user == nil {
			return "@" + userID
		}

		channel := stoat.Get(p.session, "/channels/"+channelID, channelID, &p.session.ChannelCache)
		if channel != nil && channel.Server != nil {
			member := stoat.Get(p.session, "/servers/"+*channel.Server+"/members/"+user.ID,
				*channel.Server+"-"+user.ID, &p.session.MemberCache)
			if member != nil && member.Nickname != nil {
				return "@" + *member.Nickname
			}
		}

		return "@" + user.Username
	})
}

func (p *stoatPlugin) replaceChannels(content string) string {
	return channelRegex.ReplaceAllStringFunc(content, func(match string) string {
		channelID := extractID(match, channelRegex)
		if channelID == "" {
			return match
		}

		channel := stoat.Get(p.session, "/channels/"+channelID, channelID, &p.session.ChannelCache)
		if channel == nil {
			return "#" + channelID
		}

		return "#" + channel.Name
	})
}

func extractID(match string, re *regexp.Regexp) string {
	matches := re.FindStringSubmatch(match)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func getLightningEmbeds(embeds []stoat.Embed) []lightning.Embed {
	result := make([]lightning.Embed, 0)
	for _, embed := range embeds {
		lightningEmbed := lightning.Embed{
			Title:       embed.Title,
			Description: embed.Description,
			URL:         embed.URL,
			Image:       getEmbedImage(&embed),
			Video:       getEmbedVideo(&embed),
		}

		if embed.Colour != "" {
			if colorInt, err := strconv.ParseInt((embed.Colour)[1:], 16, 32); err == nil {
				lightningEmbed.Color = int(colorInt)
			}
		}

		if embed.IconURL != nil {
			lightningEmbed.Thumbnail = &lightning.Media{URL: *embed.IconURL}
		}

		result = append(result, lightningEmbed)
	}

	return result
}

func getEmbedImage(embed *stoat.Embed) *lightning.Media {
	if embed.Image != nil && embed.Image.URL != "" {
		return &lightning.Media{URL: embed.Image.URL, Width: embed.Image.Width, Height: embed.Image.Height}
	}

	return nil
}

func getEmbedVideo(embed *stoat.Embed) *lightning.Media {
	if embed.Video != nil && embed.Video.URL != "" {
		return &lightning.Media{URL: embed.Video.URL, Width: embed.Video.Width, Height: embed.Video.Height}
	}

	return nil
}
