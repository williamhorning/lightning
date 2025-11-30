package guilded

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

const assetCacheTTL = 24 * time.Hour

var attachmentRegex = regexp.MustCompile(
	`!\[.*?\]\(https:\/\/cdn\.gldcdn\.com\/ContentMedia(?:GenericFiles)?\/.*\)`,
)

func guildedToLightning(plugin *guildedPlugin, msg *guildedChatMessage) *lightning.Message {
	if msg.ServerID == nil {
		return nil
	}

	if found, _ := plugin.webhookIDsCache.Get(msg.CreatedByWebhookID); found {
		return nil
	}

	attachmentMarkdown := attachmentRegex.FindAllString(msg.Content, -1)

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   msg.ID,
			ChannelID: msg.ChannelID,
			Time:      msg.CreatedAt,
		},
		Author:      guildedToLightningAuthor(plugin, msg),
		Content:     attachmentRegex.ReplaceAllString(msg.Content, ""),
		Embeds:      msg.Embeds,
		Attachments: guildedToLightningAttachments(plugin, attachmentMarkdown),
		RepliedTo:   msg.ReplyMessageIDs,
	}
}

func guildedToLightningAuthor(plugin *guildedPlugin, msg *guildedChatMessage) *lightning.MessageAuthor {
	var cacheKey string

	var endpoint string

	if msg.CreatedByWebhookID == "" {
		cacheKey = *msg.ServerID + "/" + msg.CreatedBy
		endpoint = "/servers/" + *msg.ServerID + "/members/" + msg.CreatedBy
	} else {
		cacheKey = *msg.ServerID + "/" + msg.CreatedByWebhookID
		endpoint = "/servers/" + *msg.ServerID + "/webhooks/" + msg.CreatedByWebhookID
	}

	if cachedMember, ok := plugin.membersCache.Get(cacheKey); ok {
		return toAuthor(cachedMember.User, cachedMember.Nickname)
	}

	var responseBody struct {
		Member  *guildedServerMember `json:"member,omitempty"`
		Webhook *guildedUser         `json:"webhook,omitempty"`
	}

	if err := doJSONRequest(plugin.token, http.MethodGet, endpoint, nil, &responseBody); err != nil {
		return &lightning.MessageAuthor{
			Nickname: "Guilded User",
			Username: "GuildedUser",
			ID:       msg.CreatedBy,
			Color:    "#f8d64c",
		}
	}

	if responseBody.Member != nil {
		plugin.membersCache.Set(cacheKey, *responseBody.Member)

		return toAuthor(responseBody.Member.User, responseBody.Member.Nickname)
	}

	plugin.membersCache.Set(cacheKey, guildedServerMember{User: *responseBody.Webhook})

	return toAuthor(*responseBody.Webhook, nil)
}

func toAuthor(user guildedUser, nickname *string) *lightning.MessageAuthor {
	displayName := user.Name
	if nickname != nil {
		displayName = *nickname
	}

	return &lightning.MessageAuthor{
		Nickname:       displayName,
		Username:       user.Name,
		ID:             user.ID,
		ProfilePicture: user.Avatar,
		Color:          "#f8d64c",
	}
}

func guildedToLightningAttachments(plugin *guildedPlugin, markdownLinks []string) []lightning.Attachment {
	attachments := make([]lightning.Attachment, 0, len(markdownLinks))

	for _, mdLink := range markdownLinks {
		url := extractURLFromMarkdown(mdLink)
		if url == "" {
			continue
		}

		if cachedAttachment, ok := plugin.assetsCache.Get(url); ok {
			attachments = append(attachments, cachedAttachment)

			continue
		}

		sig := getSignature(url, plugin.token)
		if sig == nil || len(sig.URLSignatures) == 0 || sig.URLSignatures[0].Signature == nil {
			continue
		}

		urlSignature := *sig.URLSignatures[0].Signature
		attachment := lightning.Attachment{
			Name: path.Base(strings.SplitN(urlSignature, "?", 2)[0]),
			URL:  urlSignature,
			Size: getContentLength(urlSignature),
		}

		plugin.assetsCache.Set(url, attachment)
		attachments = append(attachments, attachment)
	}

	return attachments
}

func extractURLFromMarkdown(markdown string) string {
	start, end := strings.LastIndex(markdown, "("), strings.LastIndex(markdown, ")")
	if start >= 0 && end > start {
		return markdown[start+1 : end]
	}

	return ""
}

func getContentLength(url string) int64 {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return 0
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	contentLength, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	return contentLength
}

func getSignature(url, token string) *guildedURLSignatureResponse {
	var signatureResponse guildedURLSignatureResponse
	if err := doJSONRequest(token, http.MethodPost, "/url-signatures",
		map[string][]string{"urls": {url}}, &signatureResponse); err != nil {
		return nil
	}

	return &signatureResponse
}

func doJSONRequest(token, method, endpoint string, body, out any) error {
	var buf io.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal: %w", err)
		}

		buf = bytes.NewReader(b)
	}

	resp, err := guildedMakeRequest(token, method, endpoint, buf)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	return nil
}
