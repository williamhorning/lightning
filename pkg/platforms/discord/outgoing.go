package discord

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	maxContentLength    = 2000
	maxButtonReplies    = 5
	defaultMaxFileMiB   = float64(10)
	boostTier2FileMax   = 50
	boostTier3FileMax   = 100
	fileDownloadTimeout = 10 * time.Second
)

type discordOutgoingMessage struct {
	AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions,omitempty"`
	AvatarURL       string                            `json:"avatar_url,omitempty"`
	Components      []discordgo.MessageComponent      `json:"components"`
	Content         string                            `json:"content,omitempty"`
	Embeds          []*discordgo.MessageEmbed         `json:"embeds,omitempty"`
	Files           []*discordgo.File                 `json:"-"`
	Reference       *discordgo.MessageReference       `json:"message_reference,omitempty"`
	Username        string                            `json:"username,omitempty"`
}

func (o *discordOutgoingMessage) Webhook() *discordgo.WebhookParams {
	return &discordgo.WebhookParams{
		AllowedMentions: o.AllowedMentions,
		AvatarURL:       o.AvatarURL,
		Components:      o.Components,
		Content:         o.Content,
		Embeds:          o.Embeds,
		Files:           o.Files,
		Username:        o.Username,
	}
}

func (o *discordOutgoingMessage) WebhookEdit() *discordgo.WebhookEdit {
	return &discordgo.WebhookEdit{
		AllowedMentions: o.AllowedMentions,
		Content:         &o.Content,
		Components:      &o.Components,
		Embeds:          &o.Embeds,
		Files:           o.Files,
	}
}

func (o *discordOutgoingMessage) Message() *discordgo.MessageSend {
	return &discordgo.MessageSend{
		AllowedMentions: o.AllowedMentions,
		Components:      o.Components,
		Content:         o.Content,
		Embeds:          o.Embeds,
		Files:           o.Files,
		Reference:       o.Reference,
	}
}

func getWebhookFromChannel(options *lightning.SendOptions) (id string, token string, err error) {
	webhookData, ok := options.ChannelData.(map[string]any)
	if !ok {
		return "", "", lightning.LogError(
			errors.New("invalid webhook data for Discord channel"),
			"Failed to use webhook for Discord",
			map[string]any{"channel": options.ChannelID},
			lightning.ChannelDisabled{Read: false, Write: true},
		)
	}

	id, _ = webhookData["id"].(string)
	token, _ = webhookData["token"].(string)
	return id, token, nil
}

func getOutgoingMessage(session *discordgo.Session, message lightning.Message, opts *lightning.SendOptions, button bool) *discordOutgoingMessage {
	msg := discordOutgoingMessage{
		AllowedMentions: getOutgoingMention(opts),
		AvatarURL:       getOutgoingProfile(message),
		Components:      getOutgoingComponents(session, message, button),
		Content:         getOutgoingContent(message),
		Embeds:          getOutgoingEmbeds(message),
		Files:           getOutgoingFiles(session, message),
		Reference:       getOutgoingReference(message, button),
		Username:        message.Author.Nickname,
	}

	if msg.Content == "" && len(msg.Embeds) == 0 && len(msg.Files) == 0 {
		msg.Content = "_ _"
	}

	return &msg
}

func getOutgoingMention(opts *lightning.SendOptions) *discordgo.MessageAllowedMentions {
	if opts == nil || opts.AllowEveryonePings {
		return nil
	}
	return &discordgo.MessageAllowedMentions{
		Parse: []discordgo.AllowedMentionType{
			discordgo.AllowedMentionTypeRoles,
			discordgo.AllowedMentionTypeUsers,
		},
	}
}

func getOutgoingProfile(message lightning.Message) string {
	if message.Author.ProfilePicture != nil {
		return *message.Author.ProfilePicture
	}
	return discordgo.EndpointDefaultUserAvatar(1)
}

func getOutgoingContent(message lightning.Message) string {
	if len(message.Content) > maxContentLength {
		return string([]rune(message.Content)[:maxContentLength-3]) + "..."
	}
	return message.Content
}

func getOutgoingComponents(session *discordgo.Session, message lightning.Message, button bool) []discordgo.MessageComponent {
	if !button || message.RepliedTo == nil || len(message.RepliedTo) == 0 {
		return nil
	}

	var buttons []discordgo.MessageComponent

	for i, replyID := range message.RepliedTo {
		if i >= maxButtonReplies || replyID == "" {
			continue
		}

		replyMsg, err := session.State.Message(message.ChannelID, replyID)
		if err != nil {
			continue
		}

		channel, err := session.State.Channel(replyMsg.ChannelID)
		if err != nil {
			continue
		}

		displayName := replyMsg.Author.DisplayName()
		if displayName == "" {
			displayName = "unknown user"
		}

		btn := discordgo.Button{
			Label: "reply to " + displayName,
			Style: discordgo.LinkButton,
			URL: "https://discord.com/channels/" + channel.GuildID + "/" +
				replyMsg.ChannelID + "/" + replyMsg.ID,
		}
		buttons = append(buttons, btn)
	}

	if len(buttons) == 0 {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: buttons},
	}
}

func getOutgoingEmbeds(message lightning.Message) []*discordgo.MessageEmbed {
	if len(message.Embeds) == 0 {
		return nil
	}

	var embeds []*discordgo.MessageEmbed

	for _, embed := range message.Embeds {
		discordEmbed := &discordgo.MessageEmbed{}

		if embed.Title != nil {
			discordEmbed.Title = *embed.Title
		}
		if embed.Timestamp != nil {
			discordEmbed.Timestamp = strconv.FormatInt(*embed.Timestamp, 10)
		}
		if embed.URL != nil {
			discordEmbed.URL = *embed.URL
		}
		if embed.Color != nil {
			discordEmbed.Color = *embed.Color
		}

		if embed.Footer != nil {
			footer := &discordgo.MessageEmbedFooter{Text: embed.Footer.Text}
			if embed.Footer.IconURL != nil {
				footer.IconURL = *embed.Footer.IconURL
			}
			discordEmbed.Footer = footer
		}

		if embed.Image != nil && embed.Image.URL != "" {
			discordEmbed.Image = &discordgo.MessageEmbedImage{URL: embed.Image.URL}
		}

		if embed.Thumbnail != nil && embed.Thumbnail.URL != "" {
			discordEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: embed.Thumbnail.URL}
		}

		if embed.Author != nil {
			author := &discordgo.MessageEmbedAuthor{Name: embed.Author.Name}
			if embed.Author.URL != nil {
				author.URL = *embed.Author.URL
			}
			if embed.Author.IconURL != nil {
				author.IconURL = *embed.Author.IconURL
			}
			discordEmbed.Author = author
		}

		embeds = append(embeds, discordEmbed)
	}

	return embeds
}

func getOutgoingFiles(session *discordgo.Session, message lightning.Message) []*discordgo.File {
	if len(message.Attachments) == 0 {
		return nil
	}

	maxFileSizeMiB := defaultMaxFileMiB

	if ch, err := session.State.Channel(message.ChannelID); err == nil && ch.GuildID != "" {
		if guild, err := session.State.Guild(ch.GuildID); err == nil {
			switch guild.PremiumTier {
			case 2:
				maxFileSizeMiB = boostTier2FileMax
			case 3:
				maxFileSizeMiB = boostTier3FileMax
			}
		}
	}

	var files []*discordgo.File

	for _, attachment := range message.Attachments {
		if attachment.Size > maxFileSizeMiB {
			continue
		}

		name := attachment.Name
		if name == "" {
			parts := strings.Split(attachment.URL, "/")
			name = parts[len(parts)-1]
		}

		ctx, cancel := context.WithTimeout(context.Background(), fileDownloadTimeout)
		req, err := http.NewRequestWithContext(ctx, "GET", attachment.URL, nil)
		if err != nil {
			cancel()
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			continue
		}

		files = append(files, &discordgo.File{
			Name:        name,
			ContentType: resp.Header.Get("Content-Type"),
			Reader:      &cancelableReadCloser{resp.Body, cancel},
		})
	}

	return files
}

type cancelableReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelableReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

func getOutgoingReference(message lightning.Message, button bool) *discordgo.MessageReference {
	if button || message.RepliedTo == nil || len(message.RepliedTo) == 0 {
		return nil
	}

	return &discordgo.MessageReference{
		Type:      discordgo.MessageReferenceTypeDefault,
		MessageID: message.RepliedTo[0],
		ChannelID: message.ChannelID,
	}
}
