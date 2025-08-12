package guilded

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

func (p *guildedPlugin) SetupChannel(channel string) (any, error) {
	channelData, err := getChannel(p.token, channel)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]string{"channelId": channel, "name": "Lightning Bridges"})
	if err != nil {
		slog.Error("guilded: failed to marshal webhook creation body", "error", err, "channel", channel)

		return nil, fmt.Errorf("guilded: failed to marshal webhook creation body: %w", err)
	}

	var reader io.Reader = bytes.NewReader(body)

	resp, err := guildedMakeRequest(p.token, "POST", "/servers/"+channelData.ServerID+"/webhooks", reader)
	if err != nil {
		slog.Error("guilded: failed to create webhook for channel", "error", err, "channel", channel)

		return nil, fmt.Errorf("guilded: failed to create webhook for channel %s: %w", channel, err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		extra := map[string]any{"status": resp.StatusCode, "body": string(body)}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			extra["resp"] = string(bodyBytes)
		}

		if resp.Body.Close() != nil {
			slog.Warn("guilded: failed to close request body when making webhook")
		}

		return nil, guildedStatusError{"failed to create webhook", resp.StatusCode, false}
	}

	var webhookData guildedWebhookResponse

	if err := json.NewDecoder(resp.Body).Decode(&webhookData); err != nil {
		slog.Error("guilded: failed to decode webhook data", "error", err, "channel", channel)

		return nil, fmt.Errorf("guilded: failed to decode webhook data: %w", err)
	}

	if webhookData.Webhook.Token == nil {
		slog.Error("guilded: webhook token is nil", "channel", channel, "data", webhookData)

		return nil, guildedWebhookTokenNilError{channel}
	}

	return map[string]string{"id": webhookData.Webhook.ID, "token": *webhookData.Webhook.Token}, nil
}

func getChannel(token, channel string) (*guildedServerChannel, error) {
	resp, err := guildedMakeRequest(token, "GET", "/channels/"+channel, nil)
	if err != nil {
		slog.Error("guilded: failed to get channel for setup", "error", err, "channel", channel)

		return nil, fmt.Errorf("guilded: failed to get channel %s for setup: %w", channel, err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("guilded: failed to read response body when getting channel", "error", err, "channel", channel)

		return nil, fmt.Errorf("guilded: failed to read response body when getting channel %s: %w", channel, err)
	}

	if resp.Body.Close() != nil {
		slog.Warn("guilded: failed to close request body when getting channel")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		slog.Error("guilded: unexpected status code when getting channel",
			"status", resp.StatusCode, "body", string(bodyBytes))

		return nil, guildedStatusError{"failed to get channel", resp.StatusCode, false}
	}

	var channelData guildedServerChannelResponse
	if err := json.Unmarshal(bodyBytes, &channelData); err != nil {
		slog.Error("guilded: failed to unmarshal channel data", "error", err, "body", string(bodyBytes))

		return nil, fmt.Errorf("guilded: failed to unmarshal channel data: %w", err)
	}

	return &channelData.Channel, nil
}
