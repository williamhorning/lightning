package guilded

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *guildedPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := p.getOutgoingMessage(message, opts)

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return nil, lightning.LogError(err, "failed to marshal message", map[string]any{"message": message}, nil)
	}

	reader := bytes.NewReader(jsonMsg)

	if opts == nil {
		return p.apiSendMessage(message, reader)
	}

	return p.sendWebhookMessage(message, opts, reader)
}

func (p *guildedPlugin) apiSendMessage(message lightning.Message, reader io.Reader) ([]string, error) {
	resp, err := guildedMakeRequest(p.token, "POST", "/channels/"+message.ChannelID+"/messages", reader)
	if err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to send message",
			map[string]any{"message": message, "channelID": message.ChannelID},
			nil,
		)
	}

	if err := checkStatusCode(resp, message.ChannelID); err != nil {
		return nil, err
	}

	var msg guildedChatMessageResponse
	if err := readResponse(resp, &msg, message.ChannelID); err != nil {
		return nil, err
	}

	if resp.Body.Close() != nil {
		slog.Warn("guilded: failed to close request body when sending message")
	}

	return []string{msg.Message.ID}, nil
}

func getWebhookInfo(data any) (string, string, error) {
	webhookData, ok := data.(map[string]any)
	if !ok {
		return "", "", lightning.LogError(
			guildedWebhookDataError{},
			"Failed to use webhook for Guilded",
			nil, &lightning.ChannelDisabled{Read: false, Write: true},
		)
	}

	whID, idOk := webhookData["id"].(string)
	token, tokenOk := webhookData["token"].(string)

	if !idOk || !tokenOk {
		return "", "", lightning.LogError(
			guildedWebhookDataError{},
			"Failed to use webhook for Guilded",
			nil, &lightning.ChannelDisabled{Read: false, Write: true},
		)
	}

	return whID, token, nil
}

func (p *guildedPlugin) sendWebhookMessage(
	message lightning.Message,
	opts *lightning.SendOptions,
	reader io.Reader,
) ([]string, error) {
	whID, token, err := getWebhookInfo(opts.ChannelData)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://media.guilded.gg/webhooks/%s/%s", whID, token)

	p.webhookIDsCache.Set(whID, true)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, reader)
	if err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to send message request",
			map[string]any{"message": message, "channelID": message.ChannelID},
			nil,
		)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to send message",
			map[string]any{"message": message, "channelID": message.ChannelID},
			nil,
		)
	}

	if err := checkStatusCode(resp, message.ChannelID); err != nil {
		return nil, err
	}

	var response guildedWebhookExecuteResponse
	if err := readResponse(resp, &response, message.ChannelID); err != nil {
		return nil, err
	}

	if resp.Body.Close() != nil {
		slog.Warn("guilded: failed to close request body when sending webhook message")
	}

	return []string{response.ID}, nil
}

func checkStatusCode(resp *http.Response, channelID string) error {
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	var (
		errMsg  string
		disable bool
	)

	switch resp.StatusCode {
	case http.StatusNotFound:
		errMsg = "not found! this might be a Guilded problem"
		disable = true
	case http.StatusForbidden:
		errMsg = "the bot lacks some permissions, please check them"
		disable = true
	default:
		errMsg = "unexpected status code: " + resp.Status
		disable = false
	}

	return lightning.LogError(
		guildedStatusError{errMsg, resp.StatusCode},
		"Failed to send message",
		map[string]any{"channelID": channelID},
		&lightning.ChannelDisabled{Read: false, Write: disable},
	)
}

func readResponse(resp *http.Response, target any, channelID string) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return lightning.LogError(err, "Failed to read response body", map[string]any{"channelID": channelID}, nil)
	}

	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return lightning.LogError(err, "Failed to unmarshal response", map[string]any{"channelID": channelID}, nil)
	}

	return nil
}
