package discord

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	maxContentLength    = 2000
	maxButtonReplies    = 5
	defaultMaxFileSize  = int64(10485760)
	boostTier2FileMax   = 52428800
	boostTier3FileMax   = 104857600
	fileDownloadTimeout = 10 * time.Second
)

type discordOutgoingMessage struct {
	AllowedMentions *discordgo.MessageAllowedMentions
	Reference       *discordgo.MessageReference
	AvatarURL       string
	Content         string
	Username        string
	Components      []discordgo.MessageComponent
	Embeds          []*discordgo.MessageEmbed
	Files           []*discordgo.File
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

type discordWebhook struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

func (p *discordPlugin) getWebhookFromChannel(channel string, options *lightning.SendOptions) (discordWebhook, error) {
	webhookData, ok := options.ChannelData.(map[string]any)
	if !ok {
		return discordWebhook{}, discordInvalidWebhookError{channel}
	}

	webhookID, okID := webhookData["id"].(string)
	webhookToken, okToken := webhookData["token"].(string)

	if !okID || !okToken || webhookID == "" || webhookToken == "" {
		return discordWebhook{}, discordInvalidWebhookError{channel}
	}

	p.webhookCache.Set(webhookID, true)

	return discordWebhook{webhookID, webhookToken}, nil
}

func getOutgoingMessage(
	session *discordgo.Session,
	message lightning.Message,
	opts *lightning.SendOptions,
) *discordOutgoingMessage {
	msg := discordOutgoingMessage{
		AllowedMentions: getOutgoingMention(opts),
		AvatarURL:       getOutgoingProfile(message),
		Content:         getOutgoingContent(message),
		Embeds:          getOutgoingEmbeds(message),
		Files:           getOutgoingFiles(session, message),
		Username:        message.Author.Nickname,
	}

	if opts != nil {
		msg.Components = getOutgoingComponents(session, message)
	} else {
		msg.Reference = getOutgoingReference(message)
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

func getOutgoingComponents(
	session *discordgo.Session,
	message lightning.Message,
) []discordgo.MessageComponent {
	if len(message.RepliedTo) == 0 {
		return nil
	}

	buttons := make([]discordgo.MessageComponent, 0)

	for i, replyID := range message.RepliedTo {
		if i >= maxButtonReplies || replyID == "" {
			continue
		}

		replyMsg, err := session.ChannelMessage(message.ChannelID, replyID)
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

	if len(buttons) < 1 {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: buttons},
	}
}

func getOutgoingEmbeds(message lightning.Message) []*discordgo.MessageEmbed {
	embeds := make([]*discordgo.MessageEmbed, 0)

	for _, embed := range message.Embeds {
		discordEmbed := &discordgo.MessageEmbed{}
		setEmbedBasicProperties(discordEmbed, embed)
		setEmbedFooter(discordEmbed, embed)
		setEmbedMedia(discordEmbed, embed)
		setEmbedAuthor(discordEmbed, embed)
		embeds = append(embeds, discordEmbed)
	}

	return embeds
}

func setEmbedBasicProperties(discordEmbed *discordgo.MessageEmbed, embed lightning.Embed) {
	if embed.Title != nil {
		discordEmbed.Title = *embed.Title
	}

	if embed.Timestamp != nil {
		discordEmbed.Timestamp = embed.Timestamp.Format(time.RFC3339)
	}

	if embed.URL != nil {
		discordEmbed.URL = *embed.URL
	}

	if embed.Color != nil {
		discordEmbed.Color = *embed.Color
	}
}

func setEmbedFooter(discordEmbed *discordgo.MessageEmbed, embed lightning.Embed) {
	if embed.Footer != nil {
		footer := &discordgo.MessageEmbedFooter{Text: embed.Footer.Text}
		if embed.Footer.IconURL != nil {
			footer.IconURL = *embed.Footer.IconURL
		}

		discordEmbed.Footer = footer
	}
}

func setEmbedMedia(discordEmbed *discordgo.MessageEmbed, embed lightning.Embed) {
	if embed.Image != nil && embed.Image.URL != "" {
		discordEmbed.Image = &discordgo.MessageEmbedImage{URL: embed.Image.URL}
	}

	if embed.Thumbnail != nil && embed.Thumbnail.URL != "" {
		discordEmbed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: embed.Thumbnail.URL}
	}
}

func setEmbedAuthor(discordEmbed *discordgo.MessageEmbed, embed lightning.Embed) {
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
}

func getOutgoingFiles(session *discordgo.Session, message lightning.Message) []*discordgo.File {
	if len(message.Attachments) == 0 {
		return nil
	}

	maxFileSize := getMaxFileSize(session, message)

	files := make([]*discordgo.File, 0)

	for _, attachment := range message.Attachments {
		file := getFile(&attachment, maxFileSize)

		if file == nil {
			continue
		}

		files = append(files, file)
	}

	return files
}

func getMaxFileSize(session *discordgo.Session, message lightning.Message) int64 {
	maxFileSize := defaultMaxFileSize

	if ch, err := session.State.Channel(message.ChannelID); err == nil && ch.GuildID != "" {
		if guild, err := session.State.Guild(ch.GuildID); err == nil {
			switch guild.PremiumTier {
			case discordgo.PremiumTier2:
				maxFileSize = boostTier2FileMax
			case discordgo.PremiumTier3:
				maxFileSize = boostTier3FileMax
			case discordgo.PremiumTier1, discordgo.PremiumTierNone:
			default:
			}
		}
	}

	return maxFileSize
}

func getFile(attachment *lightning.Attachment, maxFileSize int64) *discordgo.File {
	if attachment.Size > maxFileSize {
		return nil
	}

	if attachment.Name == "" {
		parts := strings.Split(attachment.URL, "/")
		attachment.Name = parts[len(parts)-1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), fileDownloadTimeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachment.URL, nil)
	if err != nil {
		cancel()

		return nil
	}

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose // see cancelableReadCloser
	if err != nil {
		cancel()

		return nil
	}

	return &discordgo.File{
		Name:        attachment.Name,
		ContentType: resp.Header.Get("Content-Type"),
		Reader:      &cancelableReadCloser{resp.Body, cancel},
	}
}

type cancelableReadCloser struct {
	io.ReadCloser

	cancel context.CancelFunc
}

func (c *cancelableReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()

	if err != nil {
		slog.Error("discord: failed closing cancelable read closer", "error", err)

		return fmt.Errorf("discord: failed closing cancelable read closer: %w", err)
	}

	return nil
}

func getOutgoingReference(message lightning.Message) *discordgo.MessageReference {
	if len(message.RepliedTo) == 0 {
		return nil
	}

	return &discordgo.MessageReference{
		Type:      discordgo.MessageReferenceTypeDefault,
		MessageID: message.RepliedTo[0],
		ChannelID: message.ChannelID,
	}
}
