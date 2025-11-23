package stoat

import (
	"regexp"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/internal/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func stoatToLightningMessage(
	session *stoat.Session,
	selfID string,
	message *stoat.Message,
) *lightning.Message {
	if message.Author == selfID && message.Masquerade.Name != "" {
		return nil
	}

	msg := &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   message.ID,
			ChannelID: message.Channel,
			Time:      stoatToLightningTime(message),
		},
		Author:      stoatToLightningAuthor(session, message),
		Content:     stoatToLightningContent(session, message),
		Embeds:      stoatToLightningEmbeds(message.Embeds),
		Attachments: stoatToLightningAttachments(message.Attachments),
		RepliedTo:   message.Replies,
	}

	msg.Content = stoatToLightningEmoji(session, msg)

	return msg
}

func stoatToLightningTime(message *stoat.Message) time.Time {
	if !message.Edited.IsZero() {
		return message.Edited
	}

	id, err := ulid.Parse(message.ID)
	if err != nil {
		return time.Now()
	}

	return id.Timestamp()
}

func stoatToLightningAuthor(session *stoat.Session, msg *stoat.Message) *lightning.MessageAuthor {
	author := &lightning.MessageAuthor{ID: msg.Author, Username: "StoatUser", Nickname: "Stoat User", Color: "#8C24EC"}

	if u, err := stoat.Get(session, "/users/"+msg.Author, msg.Author, &session.UserCache); err == nil {
		author.Username, author.Nickname = u.Username, u.Username
		if u.Avatar != nil {
			author.ProfilePicture = getStoatFileURL(u.Avatar)
		}
	}

	if mem := getStoatMember(session, msg); mem != nil {
		if mem.Nickname != nil {
			author.Nickname = *mem.Nickname
		}

		if mem.Avatar != nil {
			author.ProfilePicture = getStoatFileURL(mem.Avatar)
		}
	}

	if msg.Masquerade.Name != "" {
		author.Nickname = msg.Masquerade.Name
	}

	if msg.Masquerade.Colour != "" {
		author.Color = msg.Masquerade.Colour
	}

	if msg.Masquerade.Avatar != "" {
		author.ProfilePicture = msg.Masquerade.Avatar
	}

	return author
}

func getStoatMember(session *stoat.Session, msg *stoat.Message) *stoat.Member {
	channel, err := stoat.Get(session, "/channels/"+msg.Channel, msg.Channel, &session.ChannelCache)
	if err != nil || channel.Server == nil {
		return nil
	}

	mem, err := stoat.Get(
		session,
		"/servers/"+*channel.Server+"/members/"+msg.Author,
		*channel.Server+"-"+msg.Author,
		&session.MemberCache,
	)
	if err != nil {
		return nil
	}

	return mem
}

func stoatToLightningAttachments(attachments []stoat.File) []lightning.Attachment {
	out := make([]lightning.Attachment, len(attachments))

	for i, att := range attachments {
		out[i] = lightning.Attachment{
			URL:  getStoatFileURL(&att),
			Name: att.Filename,
			Size: int64(att.Size),
		}
	}

	return out
}

var (
	stoatSpoilerRegex = regexp.MustCompile(`!!(.+?)!!`)
	stoatEmojiRegex   = regexp.MustCompile(":([0-7][0-9A-HJKMNP-TV-Z]{25}):")
	stoatMentionRegex = regexp.MustCompile("<@([0-7][0-9A-HJKMNP-TV-Z]{25})>")
	stoatChannelRegex = regexp.MustCompile("<#([0-7][0-9A-HJKMNP-TV-Z]{25})>")
)

func stoatToLightningContent(session *stoat.Session, message *stoat.Message) string {
	content := stoatSpoilerRegex.ReplaceAllStringFunc(message.Content, func(match string) string {
		return "||" + match[2:len(match)-2] + "||"
	})

	content = stoatEmojiRegex.ReplaceAllStringFunc(content, func(match string) string {
		if emojiID := extractStoatID(match, stoatEmojiRegex); emojiID != "" {
			e, err := stoat.Get(session, "/custom/emoji/"+emojiID, emojiID, &session.EmojiCache)
			if err == nil {
				return ":" + e.Name + ":"
			}
		}

		return match
	})

	content = stoatMentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		userID := extractStoatID(match, stoatMentionRegex)
		if userID == "" {
			return match
		}

		user, err := stoat.Get(session, "/users/"+userID, userID, &session.UserCache)
		if err != nil {
			return "@" + userID
		}

		if member := getStoatMember(session, message); member != nil && member.Nickname != nil {
			return "@" + *member.Nickname
		}

		return "@" + user.Username
	})

	content = stoatChannelRegex.ReplaceAllStringFunc(content, func(match string) string {
		channelID := extractStoatID(match, stoatChannelRegex)
		if channelID == "" {
			return match
		}

		ch, err := stoat.Get(session, "/channels/"+channelID, channelID, &session.ChannelCache)
		if err != nil {
			return "#" + channelID
		}

		return "#" + ch.Name
	})

	return content
}

func stoatToLightningEmbeds(embeds []stoat.Embed) []lightning.Embed {
	out := make([]lightning.Embed, 0, len(embeds))

	for _, embed := range embeds {
		newEmbed := lightning.Embed{
			Title: embed.Title, Description: embed.Description, URL: embed.URL,
			Image: getStoatEmbedMedia(embed.Image), Video: getStoatEmbedMedia(embed.Video),
		}

		if embed.Colour != "" {
			if c, err := strconv.ParseInt(embed.Colour[1:], 16, 32); err == nil {
				newEmbed.Color = int(c)
			}
		}

		if embed.IconURL != nil {
			newEmbed.Thumbnail = &lightning.Media{URL: *embed.IconURL}
		}

		out = append(out, newEmbed)
	}

	return out
}

func stoatToLightningEmoji(session *stoat.Session, msg *lightning.Message) string {
	return stoatEmojiRegex.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if emojiID := extractStoatID(match, stoatEmojiRegex); emojiID != "" {
			emoji, err := stoat.Get(session, "/custom/emoji/"+emojiID, emojiID, &session.EmojiCache)
			if err == nil {
				msg.Emoji = append(msg.Emoji, lightning.Emoji{
					URL:  "https://cdn.stoatusercontent.com/emojis/" + emoji.ID,
					ID:   emoji.ID,
					Name: emoji.Name,
				})

				return ":" + emoji.Name + ":"
			}
		}

		return match
	})
}

func getStoatFileURL(file *stoat.File) string {
	return "https://cdn.stoatusercontent.com/" + file.Tag + "/" + file.ID
}

func getStoatEmbedMedia(media *stoat.Media) *lightning.Media {
	if media != nil && media.URL != "" {
		return &lightning.Media{
			URL:    media.URL,
			Width:  media.Width,
			Height: media.Height,
		}
	}

	return nil
}

func extractStoatID(match string, re *regexp.Regexp) string {
	matches := re.FindStringSubmatch(match)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
