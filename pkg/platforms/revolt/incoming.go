package revolt

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/sentinelb51/revoltgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func getLightningMessage(s *revoltgo.Session, m revoltgo.Message) *lightning.Message {
	if m.Author == s.State.Self().ID && m.Masquerade != nil {
		return nil
	}

	timestamp, err := getLightningTime(m)
	if err != nil {
		timestamp = time.Now()
	}

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   m.ID,
			ChannelID: m.Channel,
			Plugin:    "bolt-revolt",
			Time:      timestamp,
		},
		Attachments: getLightningAttachment(m.Attachments),
		Author:      getLightningAuthor(s, m.Author, m.Channel, m.Masquerade),
		Content:     getLightningContent(s, m.Channel, m.Content),
		Embeds:      getLightningEmbeds(m.Embeds),
		RepliedTo:   m.Replies,
	}
}

func getLightningTime(m revoltgo.Message) (time.Time, error) {
	if !m.Edited.IsZero() {
		return m.Edited, nil
	}

	id, err := ulid.Parse(m.ID)
	if err != nil {
		return time.Time{}, lightning.LogError(err, "Failed to parse ULID from Revolt message ID", map[string]any{"message_id": m.ID}, nil)
	}

	return time.UnixMilli(int64(id.Time())), nil
}

func getLightningAttachment(attachments []*revoltgo.Attachment) []lightning.Attachment {
	result := make([]lightning.Attachment, len(attachments))
	for i, att := range attachments {
		result[i] = lightning.Attachment{
			URL:  getURL(att),
			Name: att.Filename,
			Size: float64(att.Size) / 1048576,
		}
	}
	return result
}

func getUser(s *revoltgo.Session, id string) *revoltgo.User {
	if user := s.State.User(id); user != nil {
		return user
	}
	user, _ := s.User(id)
	return user
}

func getChannel(s *revoltgo.Session, id string) *revoltgo.Channel {
	if channel := s.State.Channel(id); channel != nil {
		return channel
	}
	channel, _ := s.Channel(id)
	return channel
}

func getMember(s *revoltgo.Session, serverID, userID string) *revoltgo.ServerMember {
	if member := s.State.Member(serverID, userID); member != nil {
		return member
	}
	member, _ := s.ServerMember(serverID, userID)
	return member
}

func getLightningAuthor(s *revoltgo.Session, id string, chID string, masquerade *revoltgo.MessageMasquerade) lightning.MessageAuthor {
	author := lightning.MessageAuthor{
		ID:       id,
		Username: "RevoltUser",
		Nickname: "Revolt User",
		Color:    "#FF4654",
	}

	user := getUser(s, id)
	if user == nil {
		return applyMasquerade(author, masquerade)
	}

	author.Username = user.Username
	author.Nickname = user.Username
	if user.Avatar != nil {
		profilePic := getURL(user.Avatar)
		author.ProfilePicture = &profilePic
	}

	channel := getChannel(s, chID)
	if channel != nil && channel.ChannelType == "TextChannel" && channel.Server != "" {
		if member := getMember(s, channel.Server, id); member != nil {
			if member.Nickname != nil {
				author.Nickname = *member.Nickname
			}
			if member.Avatar != nil {
				memberAvatar := getURL(member.Avatar)
				author.ProfilePicture = &memberAvatar
			}
		}
	}

	return applyMasquerade(author, masquerade)
}

func getURL(file *revoltgo.Attachment) string {
	return "https://cdn.revoltusercontent.com/" + file.Tag + "/" + file.ID
}

func applyMasquerade(author lightning.MessageAuthor, masquerade *revoltgo.MessageMasquerade) lightning.MessageAuthor {
	if masquerade == nil {
		return author
	}

	if masquerade.Name != "" {
		author.Nickname = masquerade.Name
	}
	if masquerade.Colour != "" {
		author.Color = masquerade.Colour
	}
	if masquerade.Avatar != "" {
		author.ProfilePicture = &masquerade.Avatar
	}

	return author
}

var (
	emojiRegex   = regexp.MustCompile(":([0-7][0-9A-HJKMNP-TV-Z]{25}):")
	mentionRegex = regexp.MustCompile("<@([0-7][0-9A-HJKMNP-TV-Z]{25})>")
	channelRegex = regexp.MustCompile("<#([0-7][0-9A-HJKMNP-TV-Z]{25})>")
)

func getLightningContent(s *revoltgo.Session, channelID string, content string) string {
	content = emojiRegex.ReplaceAllStringFunc(content, func(match string) string {
		if emojiID := extractID(match, emojiRegex); emojiID != "" {
			if emoji := s.State.Emoji(emojiID); emoji != nil {
				return ":" + emoji.Name + ":"
			}
			return ":" + emojiID + ":"
		}
		return match
	})

	content = mentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		userID := extractID(match, mentionRegex)
		if userID == "" {
			return match
		}

		user := getUser(s, userID)
		if user == nil {
			return "@" + userID
		}

		channel := getChannel(s, channelID)
		if channel != nil && channel.Server != "" {
			if member := getMember(s, channel.Server, userID); member != nil && member.Nickname != nil {
				return "@" + *member.Nickname
			}
		}
		return "@" + user.Username
	})

	content = channelRegex.ReplaceAllStringFunc(content, func(match string) string {
		chanID := extractID(match, channelRegex)
		if chanID == "" {
			return match
		}

		channel := getChannel(s, chanID)
		if channel == nil {
			return "#" + chanID
		}

		if channel.ChannelType == "DirectMessage" || channel.ChannelType == "GroupDM" {
			return "#DM" + chanID
		}
		return "#" + channel.Name
	})

	return content
}

func extractID(match string, re *regexp.Regexp) string {
	matches := re.FindStringSubmatch(match)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func getLightningEmbeds(embeds []*revoltgo.MessageEmbed) []lightning.Embed {
	if len(embeds) == 0 {
		return nil
	}

	result := make([]lightning.Embed, 0, len(embeds))
	for _, e := range embeds {
		embed := lightning.Embed{}

		if e.Title != "" {
			embed.Title = &e.Title
		}
		if e.Description != "" {
			embed.Description = &e.Description
		}
		if e.URL != "" {
			embed.URL = &e.URL
		}

		if e.Colour != "" {
			if colorInt, err := strconv.ParseInt(strings.TrimPrefix(e.Colour, "#"), 16, 32); err == nil {
				colorVal := int(colorInt)
				embed.Color = &colorVal
			}
		}

		if e.Image != nil && e.Image.URL != "" {
			embed.Image = &lightning.Media{
				URL:    e.Image.URL,
				Width:  e.Image.Width,
				Height: e.Image.Height,
			}
		}

		if e.Video != nil && e.Video.URL != "" {
			embed.Video = &lightning.Media{
				URL:    e.Video.URL,
				Width:  e.Video.Width,
				Height: e.Video.Height,
			}
		}

		if e.IconURL != "" {
			embed.Thumbnail = &lightning.Media{URL: e.IconURL}
		}

		result = append(result, embed)
	}

	return result
}
