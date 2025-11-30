package guilded

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_ ()-]{1,25}$`)

func (p *guildedPlugin) SendMessage(msg *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	payload := guildedPayload{Content: msg.Content, ReplyMessageIDs: msg.RepliedTo, Embeds: msg.Embeds}

	if msg.Author != nil {
		payload.AvatarURL = msg.Author.ProfilePicture
		switch {
		case usernameRegex.MatchString(msg.Author.Nickname):
			payload.Username = msg.Author.Nickname
		case usernameRegex.MatchString(msg.Author.Username):
			payload.Username = msg.Author.Username
		default:
			payload.Username = msg.Author.ID
		}
	}

	if payload.Content == "" && len(payload.Embeds) == 0 {
		payload.Content = "\u2800"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("guilded: marshal message: %w", err)
	}

	if opts == nil {
		return p.sendAPI(msg, bytes.NewReader(body))
	}

	return p.sendWebhook(opts, bytes.NewReader(body))
}

func (p *guildedPlugin) sendAPI(msg *lightning.Message, body io.Reader) ([]string, error) {
	resp, err := guildedMakeRequest(p.token, "POST", "/channels/"+msg.ChannelID+"/messages", body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, &guildedStatusError{resp.StatusCode}
	}

	var r guildedChatMessageWrapper
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("failed to decode send: %w", err)
	}

	return []string{r.Message.ID}, nil
}

func (p *guildedPlugin) sendWebhook(opts *lightning.SendOptions, payload io.Reader) ([]string, error) {
	id, token := opts.ChannelData["id"], opts.ChannelData["token"]
	p.webhookIDsCache.Set(id, true)
	url := "https://media.guilded.gg/webhooks/" + id + "/" + token

	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to make webhook request: %w", err)
	}

	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send webhook request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, &guildedStatusError{resp.StatusCode}
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode webhook send: %w", err)
	}

	return []string{body.ID}, nil
}
