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

	"github.com/williamhorning/lightning/pkg/lightning"
)

var attachmentRegex = regexp.MustCompile(`!\[.*?\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)`)
var emojiRegex = regexp.MustCompile(`<(:\w+:)\d+>`)

func extractURLFromMarkdown(markdown string) string {
	startIdx := strings.LastIndex(markdown, "(")
	endIdx := strings.LastIndex(markdown, ")")

	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		return markdown[startIdx+1 : endIdx]
	}
	return ""
}

type signatureResponse struct {
	URLSignatures []guildedUrlSignature `json:"urlSignatures"`
}

func getIncomingAttachments(token string, markdownURLs []string) []lightning.Attachment {
	var attachments []lightning.Attachment

	for _, markdownURL := range markdownURLs {
		url := extractURLFromMarkdown(markdownURL)
		if url == "" {
			continue
		}

		if cached, exists := cache.Assets.Get(url); exists {
			attachments = append(attachments, cached)
			continue
		}

		requestBody := map[string][]string{
			"urls": {url},
		}
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			continue
		}

		var bodyReader io.Reader = bytes.NewReader(jsonBody)
		resp, err := guildedMakeRequest(token, http.MethodPost, "/url-signatures", &bodyReader)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var signatureResp signatureResponse
		if err := json.Unmarshal(body, &signatureResp); err != nil {
			continue
		}

		if len(signatureResp.URLSignatures) == 0 {
			continue
		}

		signed := signatureResp.URLSignatures[0]
		if signed.RetryAfter == nil || *signed.RetryAfter > 0 || signed.Signature == nil {
			continue
		}

		filename := path.Base(*signed.Signature)
		if idx := strings.Index(filename, "?"); idx > 0 {
			filename = filename[:idx]
		}
		if filename == "" {
			filename = "unknown"
		}

		headResp, err := http.Head(*signed.Signature)
		if err != nil {
			continue
		}

		contentLength := headResp.Header.Get("Content-Length")
		size := 0.0
		if contentLength != "" {
			if sizeBytes, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				size = float64(sizeBytes) / 1048576
			}
		}
		headResp.Body.Close()

		attachment := lightning.Attachment{
			Name: filename,
			URL:  *signed.Signature,
			Size: size,
		}

		cache.Assets.Set(url, attachment)

		attachments = append(attachments, attachment)
	}

	return attachments
}

func getIncomingMessage(token string, msg *guildedChatMessage) *lightning.Message {
	if msg.ServerId == nil {
		return nil
	}

	timestamp := msg.CreatedAt

	if msg.UpdatedAt != nil {
		timestamp = *msg.UpdatedAt
	}

	content := ""

	if msg.Content != nil {
		content = *msg.Content
	}

	urls := attachmentRegex.FindAllString(content, -1)

	content = attachmentRegex.ReplaceAllString(content, "")
	content = emojiRegex.ReplaceAllString(content, "$1")

	var repliedTo []string
	if msg.ReplyMessageIds != nil {
		repliedTo = *msg.ReplyMessageIds
	}

	return &lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   msg.Id,
			ChannelID: msg.ChannelId,
			Plugin:    "bolt-guilded",
			Time:      timestamp,
		},
		Attachments: getIncomingAttachments(token, urls),
		Author:      getIncomingAuthor(token, msg),
		Content:     content,
		Embeds:      getIncomingEmbeds(msg.Embeds),
		RepliedTo:   repliedTo,
	}
}

func getIncomingAuthor(token string, msg *guildedChatMessage) lightning.MessageAuthor {
	defaultAuthor := lightning.MessageAuthor{
		Nickname: "Guilded User",
		Username: "GuildedUser",
		ID:       msg.CreatedBy,
	}

	if defaultAuthor.ID == "" {
		defaultAuthor.ID = msg.CreatedBy
	}

	try := func() (lightning.MessageAuthor, error) {
		if msg.CreatedByWebhookId == nil {
			key := *msg.ServerId + "/" + msg.CreatedBy

			if cached, exists := cache.Members.Get(key); exists {
				return lightning.MessageAuthor{
					Nickname:       getNickname(cached),
					Username:       cached.User.Name,
					ID:             msg.CreatedBy,
					ProfilePicture: cached.User.Avatar,
				}, nil
			}

			endpoint := fmt.Sprintf("/servers/%s/members/%s", *msg.ServerId, msg.CreatedBy)
			resp, err := guildedMakeRequest(token, http.MethodGet, endpoint, nil)
			if err != nil {
				return lightning.MessageAuthor{}, err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return lightning.MessageAuthor{}, err
			}

			var memberResp struct {
				Member guildedServerMember `json:"member"`
			}

			if err := json.Unmarshal(body, &memberResp); err != nil {
				return lightning.MessageAuthor{}, err
			}

			cache.Members.Set(key, memberResp.Member)

			author := memberResp.Member
			return lightning.MessageAuthor{
				Nickname:       getNickname(author),
				Username:       author.User.Name,
				ID:             msg.CreatedBy,
				ProfilePicture: author.User.Avatar,
			}, nil

		} else {
			key := *msg.ServerId + "/" + *msg.CreatedByWebhookId

			if cached, exists := cache.Webhooks.Get(key); exists {
				return lightning.MessageAuthor{
					Nickname:       cached.Name,
					Username:       cached.Name,
					ID:             cached.Id,
					ProfilePicture: cached.Avatar,
				}, nil
			}

			endpoint := fmt.Sprintf("/servers/%s/webhooks/%s", *msg.ServerId, *msg.CreatedByWebhookId)
			resp, err := guildedMakeRequest(token, http.MethodGet, endpoint, nil)
			if err != nil {
				return lightning.MessageAuthor{}, err
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return lightning.MessageAuthor{}, err
			}

			var webhookResp struct {
				Webhook guildedWebhook `json:"webhook"`
			}

			if err := json.Unmarshal(body, &webhookResp); err != nil {
				return lightning.MessageAuthor{}, err
			}

			cache.Webhooks.Set(key, webhookResp.Webhook)

			webhook := webhookResp.Webhook
			return lightning.MessageAuthor{
				Nickname:       webhook.Name,
				Username:       webhook.Name,
				ID:             webhook.Id,
				ProfilePicture: webhook.Avatar,
			}, nil
		}
	}

	author, err := try()
	if err != nil {
		return defaultAuthor
	}
	return author
}

func getNickname(member guildedServerMember) string {
	if member.Nickname != nil {
		return *member.Nickname
	}
	return member.User.Name
}

func getIncomingEmbeds(embeds *[]guildedChatEmbed) []lightning.Embed {
	if embeds == nil {
		return nil
	}

	incomingEmbeds := make([]lightning.Embed, 0)

	for _, embed := range *embeds {
		var author *lightning.EmbedAuthor
		if embed.Author != nil {
			author = &lightning.EmbedAuthor{
				Name: "",
				URL:  embed.Author.Url,
			}
			if embed.Author.Name != nil {
				author.Name = *embed.Author.Name
			}
			if embed.Author.IconUrl != nil {
				author.IconURL = embed.Author.IconUrl
			}
		}

		var footer *lightning.EmbedFooter
		if embed.Footer != nil {
			footer = &lightning.EmbedFooter{
				Text: embed.Footer.Text,
			}
			if embed.Footer.IconUrl != nil {
				footer.IconURL = embed.Footer.IconUrl
			}
		}

		var image *lightning.Media
		if embed.Image != nil && embed.Image.Url != nil {
			image = &lightning.Media{
				URL: *embed.Image.Url,
			}
		}

		var thumbnail *lightning.Media
		if embed.Thumbnail != nil && embed.Thumbnail.Url != nil {
			thumbnail = &lightning.Media{
				URL: *embed.Thumbnail.Url,
			}
		}

		var fields []lightning.EmbedField
		if embed.Fields != nil {
			fields = make([]lightning.EmbedField, len(*embed.Fields))
			for i, field := range *embed.Fields {
				fields[i] = lightning.EmbedField{
					Name:   field.Name,
					Value:  field.Value,
					Inline: field.Inline != nil && *field.Inline,
				}
			}
		}

		var timestamp *int64
		if embed.Timestamp != nil {
			ts := embed.Timestamp.Unix()
			timestamp = &ts
		}

		incomingEmbeds = append(incomingEmbeds, lightning.Embed{
			Title:       embed.Title,
			Description: embed.Description,
			URL:         embed.Url,
			Color:       embed.Color,
			Author:      author,
			Fields:      fields,
			Footer:      footer,
			Image:       image,
			Thumbnail:   thumbnail,
			Timestamp:   timestamp,
		})
	}

	return incomingEmbeds
}
