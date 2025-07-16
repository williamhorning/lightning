package guilded

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *guildedPlugin) SetupChannel(channel string) (any, error) {
	channelData, err := getChannel(p.token, channel)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(map[string]string{"channelId": channel, "name": "Lightning Bridges"})
	if err != nil {
		return nil, lightning.LogError(err, "Failed to marshal webhook creation body", nil, nil)
	}

	var reader io.Reader = bytes.NewReader(body)

	resp, err := guildedMakeRequest(p.token, "POST", "/servers/"+channelData.ServerID+"/webhooks", reader)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to create webhook for channel", nil, nil)
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

		return nil, lightning.LogError(
			guildedStatusError{"failed to create webhook", resp.StatusCode},
			"Failed to create webhook for channel",
			extra,
			nil,
		)
	}

	var webhookData guildedWebhookResponse

	if err := json.NewDecoder(resp.Body).Decode(&webhookData); err != nil {
		return nil, lightning.LogError(err, "Failed to decode webhook data", nil, nil)
	}

	if webhookData.Webhook.Token == nil {
		return nil, lightning.LogError(
			guildedWebhookTokenNilError{},
			"Failed to create webhook for channel",
			map[string]any{"channelID": channel, "webhook": webhookData},
			nil,
		)
	}

	return map[string]string{"id": webhookData.Webhook.ID, "token": *webhookData.Webhook.Token}, nil
}

func getChannel(token, channel string) (*guildedServerChannel, error) {
	resp, err := guildedMakeRequest(token, "GET", "/channels/"+channel, nil)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to get channel for setup", nil, nil)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to read response body", nil, nil)
	}

	if resp.Body.Close() != nil {
		slog.Warn("guilded: failed to close request body when getting channel")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, lightning.LogError(
			guildedStatusError{"failed to get channel", resp.StatusCode},
			"Failed to get channel for setup",
			map[string]any{"status": resp.StatusCode, "body": string(bodyBytes)},
			nil,
		)
	}

	var channelData guildedServerChannelResponse
	if err := json.Unmarshal(bodyBytes, &channelData); err != nil {
		return nil, lightning.LogError(err, "Failed to unmarshal channel data", nil, nil)
	}

	return &channelData.Channel, nil
}
