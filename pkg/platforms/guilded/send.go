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

func (p *guildedPlugin) SendCommandResponse(
	message lightning.Message,
	opts *lightning.SendOptions,
	_ string,
) ([]string, error) {
	return p.SendMessage(message, opts)
}

func (p *guildedPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := p.getOutgoingMessage(message, opts)

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		slog.Error("guilded: failed to marshal message", "error", err, "message", message)

		return nil, fmt.Errorf("guilded: failed to marshal message: %w", err)
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
		slog.Error("guilded: failed to send message", "error", err, "message", message)

		return nil, fmt.Errorf("guilded: failed to send message: %w", err)
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

func getWebhookInfo(data any) (guildedWebhook, error) {
	webhookData, ok := data.(map[string]any)
	if !ok {
		return guildedWebhook{}, guildedWebhookDataError{}
	}

	whID, idOk := webhookData["id"].(string)
	token, tokenOk := webhookData["token"].(string)

	if !idOk || !tokenOk {
		return guildedWebhook{}, guildedWebhookDataError{}
	}

	return guildedWebhook{ID: whID, Token: &token}, nil
}

func (p *guildedPlugin) sendWebhookMessage(
	message lightning.Message,
	opts *lightning.SendOptions,
	reader io.Reader,
) ([]string, error) {
	webhook, err := getWebhookInfo(opts.ChannelData)
	if err != nil {
		return nil, err
	}

	url := "https://media.guilded.gg/webhooks/" + webhook.ID + "/" + *webhook.Token

	p.webhookIDsCache.Set(webhook.ID, true)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, reader)
	if err != nil {
		return nil, fmt.Errorf("guilded: failed to create webhook message in channel ID %s (message %+v): %w",
			message.ChannelID, message, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("guilded: failed to send webhook message", "error", err, "message", message)

		return nil, fmt.Errorf("guilded: failed to send webhook message in channel ID %s (message %+v): %w",
			message.ChannelID, message, err)
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

	slog.Error("guilded: failed to send message: ", "status", resp.StatusCode, "channelID", channelID)

	return guildedStatusError{"failed to send message: " + errMsg, resp.StatusCode, disable}
}

func readResponse(resp *http.Response, target any, channelID string) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("guilded: failed to read response body", "error", err, "status", resp.StatusCode,
			"channelID", channelID)

		return fmt.Errorf("guilded: failed to read response body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, target); err != nil {
		slog.Error("guilded: failed to unmarshal response body", "error", err, "body", string(bodyBytes),
			"channelID", channelID)

		return fmt.Errorf("guilded: failed to unmarshal response body: %w", err)
	}

	return nil
}
