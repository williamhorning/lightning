package guilded

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *guildedPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := getOutgoingMessage(message, opts, p.token)
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to marshal outgoing message", map[string]any{"message": message}, nil)
	}

	reader := bytes.NewReader(jsonMsg)

	if opts == nil {
		return p.sendMessage(message, reader)
	}
	return p.sendWebhookMessage(message, opts, reader)
}

func (p *guildedPlugin) sendMessage(message lightning.Message, reader io.Reader) ([]string, error) {
	resp, err := guildedMakeRequest(p.token, "POST", "/channels/"+message.ChannelID+"/messages", &reader)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to send message", map[string]any{"message": message, "channelID": message.ChannelID}, nil)
	}
	defer resp.Body.Close()

	if err := p.checkStatusCode(resp, message.ChannelID); err != nil {
		return nil, err
	}

	var msg guildedChatMessageResponse
	if err := p.readResponse(resp, &msg, message.ChannelID); err != nil {
		return nil, err
	}

	return []string{msg.Message.ID}, nil
}

func (p *guildedPlugin) sendWebhookMessage(message lightning.Message, opts *lightning.SendOptions, reader io.Reader) ([]string, error) {
	webhookData, ok := opts.ChannelData.(map[string]any)
	if !ok {
		return nil, lightning.LogError(errors.New("invalid webhook data for Guilded channel"), "Failed to use webhook for Guilded", map[string]any{"channel": opts.ChannelID}, &lightning.ChannelDisabled{Read: false, Write: true})
	}

	id, _ := webhookData["id"].(string)
	token, _ := webhookData["token"].(string)
	url := fmt.Sprintf("https://media.guilded.gg/webhooks/%s/%s", id, token)

	webhookIDsCache.Set(id, true)

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to send message request", map[string]any{"message": message, "channelID": opts.ChannelID}, nil)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to send message", map[string]any{"message": message, "channelID": opts.ChannelID}, nil)
	}
	defer resp.Body.Close()

	if err := p.checkStatusCode(resp, opts.ChannelID); err != nil {
		return nil, err
	}

	var response guildedWebhookExecuteResponse
	if err := p.readResponse(resp, &response, opts.ChannelID); err != nil {
		return nil, err
	}

	return []string{response.ID}, nil
}

func (p *guildedPlugin) checkStatusCode(resp *http.Response, channelID string) error {
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		return nil
	}

	var errMsg string
	var disable bool
	switch resp.StatusCode {
	case 404:
		errMsg = "not found! this might be a Guilded problem"
		disable = true
	case 403:
		errMsg = "the bot lacks some permissions, please check them"
		disable = true
	default:
		errMsg = "unexpected status code: " + resp.Status
		disable = false
	}

	return lightning.LogError(errors.New(errMsg), "Failed to send message", map[string]any{"channelID": channelID}, &lightning.ChannelDisabled{Read: false, Write: disable})
}

func (p *guildedPlugin) readResponse(resp *http.Response, target any, channelID string) error {
	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return lightning.LogError(err, "Failed to read response body", map[string]any{"channelID": channelID}, nil)
	}

	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return lightning.LogError(err, "Failed to unmarshal response", map[string]any{"channelID": channelID}, nil)
	}

	return nil
}
