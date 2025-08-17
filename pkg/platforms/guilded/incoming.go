package guilded

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	assetCacheTTL   = 24 * time.Hour
	defaultCacheTTL = 30 * time.Second
)

var (
	attachmentRegex = regexp.MustCompile(`!\[.*?\]\(https:\/\/cdn\.gldcdn\.com\/ContentMedia(GenericFiles)?\/.*\)`)
	emojiRegex      = regexp.MustCompile(`<(:\w+:)\d+>`)
)

func extractURLFromMarkdown(markdown string) string {
	startIDx := strings.LastIndex(markdown, "(")
	endIDx := strings.LastIndex(markdown, ")")

	if startIDx != -1 && endIDx != -1 && startIDx < endIDx {
		return markdown[startIDx+1 : endIDx]
	}

	return ""
}

func (p *guildedPlugin) getIncomingAttachments(markdownURLs []string) []lightning.Attachment {
	attachments := make([]lightning.Attachment, 0)

	for _, markdownURL := range markdownURLs {
		url := extractURLFromMarkdown(markdownURL)
		if url == "" {
			continue
		}

		if cached, exists := p.assetsCache.Get(url); exists {
			attachments = append(attachments, cached)

			continue
		}

		signatureResp := getSignature(url, p.token)

		if signatureResp == nil || len(signatureResp.URLSignatures) == 0 {
			continue
		}

		signed := signatureResp.URLSignatures[0]
		if signed.RetryAfter != nil || signed.Signature == nil {
			continue
		}

		attachment := lightning.Attachment{
			Name: getFilename(*signed.Signature),
			URL:  *signed.Signature,
			Size: getContentLength(*signed.Signature),
		}

		p.assetsCache.Set(url, attachment)
		attachments = append(attachments, attachment)
	}

	return attachments
}

func getFilename(url string) string {
	filename := path.Base(url)
	if idx := strings.Index(filename, "?"); idx > 0 {
		filename = filename[:idx]
	}

	return filename
}

func getContentLength(url string) int64 {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodHead, url, nil)
	if err != nil {
		return 0.0
	}

	headResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0.0
	}

	contentLength := headResp.Header.Get("Content-Length")
	size := int64(0)

	if contentLength != "" {
		var sizeBytes int64
		if sizeBytes, err = strconv.ParseInt(contentLength, 10, 64); err == nil {
			size = sizeBytes
		}
	}

	if err = headResp.Body.Close(); err != nil {
		slog.Warn("guilded: failed to close request body when getting content length")
	}

	return size
}

func getSignature(url, token string) *guildedURLSignatureResponse {
	jsonBody, err := json.Marshal(map[string][]string{"urls": {url}})
	if err != nil {
		return nil
	}

	var reader io.Reader = bytes.NewReader(jsonBody)

	resp, err := guildedMakeRequest(token, http.MethodPost, "/url-signatures", reader)
	if err != nil {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	err = resp.Body.Close()
	if err != nil {
		slog.Warn("guilded: failed to close request body when getting signature")
	}

	var signatureResp guildedURLSignatureResponse
	if err := json.Unmarshal(body, &signatureResp); err != nil {
		return nil
	}

	return &signatureResp
}

func (p *guildedPlugin) getIncomingMessage(msg *guildedChatMessage) *lightning.Message {
	if msg.ServerID == nil {
		return nil
	}

	if msg.CreatedByWebhookID != nil {
		if exists, _ := p.webhookIDsCache.Get(*msg.CreatedByWebhookID); exists {
			return nil
		}
	}

	content := ""

	if msg.Content != nil {
		content = *msg.Content
	}

	urls := attachmentRegex.FindAllString(content, -1)

	content = attachmentRegex.ReplaceAllString(content, "")
	content = emojiRegex.ReplaceAllString(content, "$1")

	var repliedTo []string
	if msg.ReplyMessageIDs != nil {
		repliedTo = *msg.ReplyMessageIDs
	}

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   msg.ID,
			ChannelID: msg.ChannelID,
			Time:      msg.CreatedAt,
		},
		Attachments: p.getIncomingAttachments(urls),
		Author:      p.getIncomingAuthor(msg),
		Content:     content,
		Embeds:      getIncomingEmbeds(msg.Embeds),
		RepliedTo:   repliedTo,
	}
}

func (p *guildedPlugin) getIncomingAuthor(msg *guildedChatMessage) lightning.MessageAuthor {
	defaultAuthor := lightning.MessageAuthor{
		Nickname: "Guilded User",
		Username: "GuildedUser",
		ID:       msg.CreatedBy,
	}

	if defaultAuthor.ID == "" {
		defaultAuthor.ID = msg.CreatedBy
	}

	author, err := p.getAuthor(msg)
	if err != nil {
		return defaultAuthor
	}

	return author
}

func (p *guildedPlugin) getAuthor(msg *guildedChatMessage) (lightning.MessageAuthor, error) {
	if msg.CreatedByWebhookID == nil {
		return p.getMemberAuthor(msg)
	}

	return p.getWebhookAuthor(msg)
}

func (p *guildedPlugin) getMemberAuthor(msg *guildedChatMessage) (lightning.MessageAuthor, error) {
	key := *msg.ServerID + "/" + msg.CreatedBy

	if cached, exists := p.membersCache.Get(key); exists {
		return cached.toAuthor(), nil
	}

	endpoint := "/servers/" + *msg.ServerID + "/members/" + msg.CreatedBy

	resp, err := guildedMakeRequest(p.token, http.MethodGet, endpoint, nil)
	if err != nil {
		return lightning.MessageAuthor{}, err
	}

	var memberResp guildedServerMemberResponse
	if err := parseResponse(resp, &memberResp); err != nil {
		return lightning.MessageAuthor{}, err
	}

	p.membersCache.Set(key, memberResp.Member)

	return memberResp.Member.toAuthor(), nil
}

func (p *guildedPlugin) getWebhookAuthor(msg *guildedChatMessage) (lightning.MessageAuthor, error) {
	key := *msg.ServerID + "/" + *msg.CreatedByWebhookID

	if cached, exists := p.webhooksCache.Get(key); exists {
		return cached.toAuthor(), nil
	}

	endpoint := "/servers/" + *msg.ServerID + "/webhooks/" + *msg.CreatedByWebhookID

	resp, err := guildedMakeRequest(p.token, http.MethodGet, endpoint, nil)
	if err != nil {
		return lightning.MessageAuthor{}, err
	}

	var webhookResp guildedWebhookResponse
	if err := parseResponse(resp, &webhookResp); err != nil {
		return lightning.MessageAuthor{}, err
	}

	p.webhooksCache.Set(key, webhookResp.Webhook)

	return webhookResp.Webhook.toAuthor(), nil
}

func parseResponse(resp *http.Response, result any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("guilded: failed to read response body", "error", err, "status", resp.StatusCode)

		return fmt.Errorf("guilded: failed to read response body: %w", err)
	}

	if resp.Body.Close() != nil {
		slog.Warn("guilded: failed to close request body")
	}

	if err := json.Unmarshal(body, result); err != nil {
		slog.Error("guilded: failed to unmarshal response body", "error", err, "body", string(body))

		return fmt.Errorf("guilded: failed to unmarshal response body: %w", err)
	}

	return nil
}

func processEmbedAuthor(author *guildedChatEmbedAuthor) *lightning.EmbedAuthor {
	if author == nil {
		return nil
	}

	result := &lightning.EmbedAuthor{
		Name: "",
		URL:  author.URL,
	}

	if author.Name != nil {
		result.Name = *author.Name
	}

	if author.IconURL != nil {
		result.IconURL = author.IconURL
	}

	return result
}

func processEmbedFooter(footer *guildedChatEmbedFooter) *lightning.EmbedFooter {
	if footer == nil {
		return nil
	}

	result := &lightning.EmbedFooter{
		Text: footer.Text,
	}

	if footer.IconURL != nil {
		result.IconURL = footer.IconURL
	}

	return result
}

func processEmbedMedia(media *guildedChatEmbedMedia) *lightning.Media {
	if media == nil || media.URL == nil {
		return nil
	}

	return &lightning.Media{
		URL: *media.URL,
	}
}

func processEmbedFields(fields *[]guildedChatEmbedField) []lightning.EmbedField {
	if fields == nil {
		return nil
	}

	result := make([]lightning.EmbedField, len(*fields))
	for i, field := range *fields {
		result[i] = lightning.EmbedField{
			Name:   field.Name,
			Value:  field.Value,
			Inline: field.Inline != nil && *field.Inline,
		}
	}

	return result
}

func getIncomingEmbeds(embeds *[]guildedChatEmbed) []lightning.Embed {
	if embeds == nil {
		return nil
	}

	incomingEmbeds := make([]lightning.Embed, 0)

	for _, embed := range *embeds {
		incomingEmbeds = append(incomingEmbeds, lightning.Embed{
			Title:       embed.Title,
			Description: embed.Description,
			URL:         embed.URL,
			Color:       embed.Color,
			Author:      processEmbedAuthor(embed.Author),
			Fields:      processEmbedFields(embed.Fields),
			Footer:      processEmbedFooter(embed.Footer),
			Image:       processEmbedMedia(embed.Image),
			Thumbnail:   processEmbedMedia(embed.Thumbnail),
			Timestamp:   embed.Timestamp,
		})
	}

	return incomingEmbeds
}
