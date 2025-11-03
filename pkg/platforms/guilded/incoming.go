package guilded

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

const assetCacheTTL = 24 * time.Hour

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

	contentLength := headResp.Header["Content-Length"][0]
	size := int64(0)

	if contentLength != "" {
		var sizeBytes int64
		if sizeBytes, err = strconv.ParseInt(contentLength, 10, 64); err == nil {
			size = sizeBytes
		}
	}

	if err = headResp.Body.Close(); err != nil {
		log.Println("guilded: failed to close request body when getting content length")
	}

	return size
}

func getSignature(url, token string) *guildedURLSignatureResponse {
	jsonBody, err := json.Marshal(map[string][]string{"urls": {url}})
	if err != nil {
		return nil
	}

	resp, err := guildedMakeRequest(token, http.MethodPost, "/url-signatures", bytes.NewReader(jsonBody))
	if err != nil {
		return nil
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Println("guilded: failed to close request body when getting signature")
		}
	}()

	var signatureResp guildedURLSignatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&signatureResp); err != nil {
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

	urls := attachmentRegex.FindAllString(msg.Content, -1)

	msg.Content = attachmentRegex.ReplaceAllString(msg.Content, "")
	msg.Content = emojiRegex.ReplaceAllString(msg.Content, "$1")

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
		Content:     msg.Content,
		Embeds:      msg.Embeds,
		RepliedTo:   repliedTo,
	}
}

func (p *guildedPlugin) getIncomingAuthor(msg *guildedChatMessage) *lightning.MessageAuthor {
	defaultAuthor := lightning.MessageAuthor{
		Nickname: "Guilded User",
		Username: "GuildedUser",
		ID:       msg.CreatedBy,
		Color:    "#f8d64c",
	}

	if defaultAuthor.ID == "" {
		defaultAuthor.ID = msg.CreatedBy
	}

	author, err := p.getAuthor(msg)
	if err != nil {
		return &defaultAuthor
	}

	return &author
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
		return getMemberAuthorData(&cached), nil
	}

	endpoint := "/servers/" + *msg.ServerID + "/members/" + msg.CreatedBy

	resp, err := guildedMakeRequest(p.token, http.MethodGet, endpoint, nil)
	if err != nil {
		return lightning.MessageAuthor{}, err
	}

	var memberResp struct {
		Member guildedServerMember `json:"member"`
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("guilded: failed to close request body")
		}
	}()

	if err := json.NewDecoder(resp.Body).Decode(&memberResp); err != nil {
		return lightning.MessageAuthor{}, fmt.Errorf("guilded: failed to unmarshal response body: %w", err)
	}

	p.membersCache.Set(key, memberResp.Member)

	return getMemberAuthorData(&memberResp.Member), nil
}

func getMemberAuthorData(member *guildedServerMember) lightning.MessageAuthor {
	nickname := member.User.Name

	if member.Nickname != nil {
		nickname = *member.Nickname
	}

	return lightning.MessageAuthor{
		Nickname:       nickname,
		Username:       member.User.Name,
		ID:             member.User.ID,
		ProfilePicture: member.User.Avatar,
		Color:          "#f8d64c",
	}
}

func (p *guildedPlugin) getWebhookAuthor(msg *guildedChatMessage) (lightning.MessageAuthor, error) {
	key := *msg.ServerID + "/" + *msg.CreatedByWebhookID

	if cached, exists := p.webhooksCache.Get(key); exists {
		return getWebhookAuthorData(&cached), nil
	}

	endpoint := "/servers/" + *msg.ServerID + "/webhooks/" + *msg.CreatedByWebhookID

	resp, err := guildedMakeRequest(p.token, http.MethodGet, endpoint, nil)
	if err != nil {
		return lightning.MessageAuthor{}, err
	}

	var webhookResp struct {
		Webhook guildedWebhook `json:"webhook"`
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("guilded: failed to close request body")
		}
	}()

	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		return lightning.MessageAuthor{}, fmt.Errorf("guilded: failed to unmarshal response body: %w", err)
	}

	p.webhooksCache.Set(key, webhookResp.Webhook)

	return getWebhookAuthorData(&webhookResp.Webhook), nil
}

func getWebhookAuthorData(wh *guildedWebhook) lightning.MessageAuthor {
	return lightning.MessageAuthor{
		Nickname:       wh.Name,
		Username:       wh.Name,
		ID:             wh.ID,
		ProfilePicture: wh.Avatar,
	}
}
