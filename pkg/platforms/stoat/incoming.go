package stoat

import (
	"regexp"
	"strconv"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/oklog/ulid/v2"
)

func stoatToLightningMessage(
	session *session,
	selfID string,
	message *stMessage,
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

func stoatToLightningTime(message *stMessage) time.Time {
	if !message.Edited.IsZero() {
		return message.Edited
	}

	id, err := ulid.Parse(message.ID)
	if err != nil {
		return time.Now()
	}

	return id.Timestamp()
}

func stoatToLightningAuthor(session *session, msg *stMessage) *lightning.MessageAuthor {
	author := &lightning.MessageAuthor{ID: msg.Author, Username: "Stoat User", Color: "#8C24EC"}

	if u, err := get(session, "/users/"+msg.Author, msg.Author, &session.userCache); err == nil {
		author.Username = u.Username
		if u.Avatar != nil {
			author.ProfilePicture = getStoatFileURL(u.Avatar)
		}
	}

	if mem := getStoatMember(session, msg.Channel, msg.Author); mem != nil {
		if mem.Nickname != nil {
			author.Username = *mem.Nickname
		}

		if mem.Avatar != nil {
			author.ProfilePicture = getStoatFileURL(mem.Avatar)
		}
	}

	if msg.Masquerade.Name != "" {
		author.Username = msg.Masquerade.Name
	}

	if msg.Masquerade.Colour != "" {
		author.Color = msg.Masquerade.Colour
	}

	if msg.Masquerade.Avatar != "" {
		author.ProfilePicture = msg.Masquerade.Avatar
	}

	return author
}

func getStoatMember(session *session, chID, author string) *stMember {
	channel, err := get(session, "/channels/"+chID, chID, &session.channelCache)
	if err != nil || channel.Server == nil {
		return nil
	}

	mem, err := get(
		session,
		"/servers/"+*channel.Server+"/members/"+author,
		*channel.Server+"-"+author,
		&session.memberCache,
	)
	if err != nil {
		return nil
	}

	return mem
}

func stoatToLightningAttachments(attachments []stFile) []lightning.Attachment {
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
	stoatEmojiRegex   = regexp.MustCompile(":([0-7][0-9A-HJKMNP-TV-Z]{25}):")
	stoatMentionRegex = regexp.MustCompile("<@([0-7][0-9A-HJKMNP-TV-Z]{25})>")
	stoatChannelRegex = regexp.MustCompile("<#([0-7][0-9A-HJKMNP-TV-Z]{25})>")
)

func stoatToLightningContent(session *session, message *stMessage) string {
	content := message.Content

	content = stoatEmojiRegex.ReplaceAllStringFunc(content, func(match string) string {
		if emojiID := extractStoatID(match, stoatEmojiRegex); emojiID != "" {
			e, err := get(session, "/custom/emoji/"+emojiID, emojiID, &session.emojiCache)
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

		user, err := get(session, "/users/"+userID, userID, &session.userCache)
		if err != nil {
			return "@" + userID
		}

		if member := getStoatMember(session, message.Channel, user.ID); member != nil && member.Nickname != nil {
			return "@" + *member.Nickname
		}

		return "@" + user.Username
	})

	content = stoatChannelRegex.ReplaceAllStringFunc(content, func(match string) string {
		channelID := extractStoatID(match, stoatChannelRegex)
		if channelID == "" {
			return match
		}

		ch, err := get(session, "/channels/"+channelID, channelID, &session.channelCache)
		if err != nil {
			return "#" + channelID
		}

		return "#" + ch.Name
	})

	return content
}

func stoatToLightningEmbeds(embeds []stEmbed) []lightning.Embed {
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

func stoatToLightningEmoji(session *session, msg *lightning.Message) string {
	return stoatEmojiRegex.ReplaceAllStringFunc(msg.Content, func(match string) string {
		if emojiID := extractStoatID(match, stoatEmojiRegex); emojiID != "" {
			emoji, err := get(session, "/custom/emoji/"+emojiID, emojiID, &session.emojiCache)
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

func getStoatFileURL(file *stFile) string {
	return "https://cdn.stoatusercontent.com/" + file.Tag + "/" + file.ID
}

func getStoatEmbedMedia(media *stMedia) *lightning.Media {
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
