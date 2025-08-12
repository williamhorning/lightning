package revolt

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *revoltPlugin) getIncomingMessage(message revoltMessage) *lightning.Message {
	if message.Author == p.self.ID && message.Masquerade != nil {
		return nil
	}

	timestamp, err := getLightningTime(message)
	if err != nil {
		timestamp = time.Now()
	}

	content := p.replaceEmojis(message.Content)
	content = p.replaceMentions(message.Channel, content)
	content = p.replaceChannels(content)

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   message.ID,
			ChannelID: message.Channel,
			Time:      timestamp,
		},
		Attachments: getLightningAttachment(message.Attachments),
		Author:      p.getLightningAuthor(message.Author, message.Channel, message.Masquerade),
		Content:     content,
		Embeds:      getLightningEmbeds(message.Embeds),
		RepliedTo:   message.Replies,
	}
}

func getLightningTime(message revoltMessage) (time.Time, error) {
	if !message.Edited.IsZero() {
		return message.Edited, nil
	}

	msgID, err := ulid.Parse(message.ID)
	if err != nil {
		slog.Error("revolt: failed to parse message ID", "error", err, "messageID", message.ID)

		return time.Time{}, fmt.Errorf("failed to parse ULID from Revolt message ID: %w", err)
	}

	return msgID.Timestamp(), nil
}

func getLightningAttachment(attachments []*revoltAttachment) []lightning.Attachment {
	result := make([]lightning.Attachment, len(attachments))
	for i, att := range attachments {
		result[i] = lightning.Attachment{
			URL:  getURL(att),
			Name: att.Filename,
			Size: int64(att.Size),
		}
	}

	return result
}

func (p *revoltPlugin) getLightningAuthor(
	authorID string,
	channelID string,
	masquerade *revoltMessageMasquerade,
) lightning.MessageAuthor {
	author := lightning.MessageAuthor{
		ID:       authorID,
		Username: "RevoltUser",
		Nickname: "Revolt User",
		Color:    "#FF4654",
	}

	user := p.getUser(authorID)
	if user == nil {
		return applyMasquerade(author, masquerade)
	}

	author.Username = user.Username
	author.Nickname = user.Username

	if user.Avatar != nil {
		profilePic := getURL(user.Avatar)
		author.ProfilePicture = &profilePic
	}

	p.setServerMember(&author, authorID, channelID)

	return applyMasquerade(author, masquerade)
}

func (p *revoltPlugin) setServerMember(author *lightning.MessageAuthor, authorID, channelID string) {
	channel := p.getChannel(channelID)
	if channel == nil || channel.ChannelType != "TextChannel" || channel.Server == "" {
		return
	}

	member := p.getMember(channel.Server, authorID)
	if member == nil {
		return
	}

	if member.Nickname != nil {
		author.Nickname = *member.Nickname
	}

	if member.Avatar != nil {
		memberAvatar := getURL(member.Avatar)
		author.ProfilePicture = &memberAvatar
	}
}

func getURL(file *revoltAttachment) string {
	return "https://cdn.revoltusercontent.com/" + file.Tag + "/" + file.ID
}

func applyMasquerade(author lightning.MessageAuthor, masquerade *revoltMessageMasquerade) lightning.MessageAuthor {
	if masquerade == nil {
		return author
	}

	if masquerade.Name != "" {
		author.Nickname = masquerade.Name
	}

	if masquerade.Color != "" {
		author.Color = masquerade.Color
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

func (p *revoltPlugin) replaceEmojis(content string) string {
	return emojiRegex.ReplaceAllStringFunc(content, func(match string) string {
		if emojiID := extractID(match, emojiRegex); emojiID != "" {
			emoji := p.getEmoji(emojiID)
			if emoji != nil {
				return ":" + emoji.Name + ":"
			}
		}

		return match
	})
}

func (p *revoltPlugin) replaceMentions(channelID string, content string) string {
	return mentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		userID := extractID(match, mentionRegex)
		if userID == "" {
			return match
		}

		user := p.getUser(userID)
		if user == nil {
			return "@" + userID
		}

		channel := p.getChannel(channelID)
		if channel != nil && channel.Server != "" {
			member := p.getMember(channel.Server, userID)
			if member != nil && member.Nickname != nil {
				return "@" + *member.Nickname
			}
		}

		return "@" + user.Username
	})
}

func (p *revoltPlugin) replaceChannels(content string) string {
	return channelRegex.ReplaceAllStringFunc(content, func(match string) string {
		chanID := extractID(match, channelRegex)
		if chanID == "" {
			return match
		}

		channel := p.getChannel(chanID)
		if channel == nil {
			return "#" + chanID
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

func getLightningEmbeds(embeds []*revoltMessageEmbed) []lightning.Embed {
	result := make([]lightning.Embed, 0)
	for _, embed := range embeds {
		lightningEmbed := lightning.Embed{
			Image: getEmbedImage(embed),
			Video: getEmbedVideo(embed),
		}

		if embed.Title != "" {
			lightningEmbed.Title = &embed.Title
		}

		if embed.Description != "" {
			lightningEmbed.Description = &embed.Description
		}

		if embed.URL != "" {
			lightningEmbed.URL = &embed.URL
		}

		if embed.Color != "" {
			if colorInt, err := strconv.ParseInt(strings.TrimPrefix(embed.Color, "#"), 16, 32); err == nil {
				colorVal := int(colorInt)
				lightningEmbed.Color = &colorVal
			}
		}

		if embed.IconURL != "" {
			lightningEmbed.Thumbnail = &lightning.Media{URL: embed.IconURL}
		}

		result = append(result, lightningEmbed)
	}

	return result
}

func getEmbedImage(embed *revoltMessageEmbed) *lightning.Media {
	if embed.Image != nil && embed.Image.URL != "" {
		return &lightning.Media{
			URL:    embed.Image.URL,
			Width:  embed.Image.Width,
			Height: embed.Image.Height,
		}
	}

	return nil
}

func getEmbedVideo(embed *revoltMessageEmbed) *lightning.Media {
	if embed.Video != nil && embed.Video.URL != "" {
		return &lightning.Media{
			URL:    embed.Video.URL,
			Width:  embed.Video.Width,
			Height: embed.Video.Height,
		}
	}

	return nil
}
