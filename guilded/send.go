package guilded

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/williamhorning/lightning"
)

func (p *guildedPlugin) SendMessage(message lightning.Message, opts *lightning.BridgeMessageOptions) ([]string, error) {
	msg := getOutgoingMessage(message, opts, p.token)
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to marshal outgoing message",
			map[string]any{"message": message}, lightning.ReadWriteDisabled{})
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
		return nil, lightning.LogError(err, "Failed to send message",
			map[string]any{"message": message, "channelID": message.ChannelID}, lightning.ReadWriteDisabled{})
	}
	defer resp.Body.Close()

	if err := p.checkStatusCode(resp, message.ChannelID); err != nil {
		return nil, err
	}

	var msg struct {
		Message guildedChatMessage `json:"message"`
	}
	if err := p.readResponse(resp, &msg, message.ChannelID); err != nil {
		return nil, err
	}

	return []string{msg.Message.Id}, nil
}

func (p *guildedPlugin) sendWebhookMessage(message lightning.Message, opts *lightning.BridgeMessageOptions, reader io.Reader) ([]string, error) {
	webhookData, ok := opts.Channel.Data.(map[string]any)
	if !ok {
		return nil, lightning.LogError(errors.New("invalid webhook data for Guilded channel"),
			"Failed to use webhook for Guilded", map[string]any{"channel": opts.Channel.ID},
			lightning.ReadWriteDisabled{Read: false, Write: true})
	}

	id, _ := webhookData["id"].(string)
	token, _ := webhookData["token"].(string)
	url := fmt.Sprintf("https://media.guilded.gg/webhooks/%s/%s", id, token)

	req, err := http.NewRequest("POST", url, reader)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to send message request",
			map[string]any{"message": message, "channelID": opts.Channel.ID}, lightning.ReadWriteDisabled{})
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to send message",
			map[string]any{"message": message, "channelID": opts.Channel.ID}, lightning.ReadWriteDisabled{})
	}
	defer resp.Body.Close()

	if err := p.checkStatusCode(resp, opts.Channel.ID); err != nil {
		return nil, err
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := p.readResponse(resp, &response, opts.Channel.ID); err != nil {
		return nil, err
	}

	return []string{response.ID}, nil
}

func (p *guildedPlugin) checkStatusCode(resp *http.Response, channelID string) error {
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		return nil
	}

	var errMsg string
	switch resp.StatusCode {
	case 404:
		errMsg = "not found! this might be a Guilded problem"
	case 403:
		errMsg = "the bot lacks some permissions, please check them"
	default:
		errMsg = "unexpected status code: " + resp.Status
	}

	return lightning.LogError(errors.New(errMsg), "Failed to send message",
		map[string]any{"channelID": channelID},
		lightning.ReadWriteDisabled{Read: false, Write: true})
}

func (p *guildedPlugin) readResponse(resp *http.Response, target any, channelID string) error {
	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return lightning.LogError(err, "Failed to read response body",
			map[string]any{"channelID": channelID}, lightning.ReadWriteDisabled{})
	}

	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return lightning.LogError(err, "Failed to unmarshal response",
			map[string]any{"channelID": channelID}, lightning.ReadWriteDisabled{})
	}

	return nil
}
