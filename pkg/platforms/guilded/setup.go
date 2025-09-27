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

	body, err := json.Marshal(map[string]string{"channelId": channel, "name": channel})
	if err != nil {
		return nil, fmt.Errorf("guilded: failed to marshal webhook creation body: %w\n\tchannel: %s", err, channel)
	}

	var reader io.Reader = bytes.NewReader(body)

	resp, err := guildedMakeRequest(p.token, "POST", "/servers/"+channelData.ServerID+"/webhooks", reader)
	if err != nil {
		return nil, fmt.Errorf("guilded: failed to create webhook for channel: %w\n\tchannel: %s", err, channel)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			body = []byte(string(body) + "\n\tdata: " + string(bodyBytes))
		}

		if resp.Body.Close() != nil {
			slog.Warn("guilded: failed to close request body when making webhook")
		}

		return nil, &guildedStatusError{"failed to create webhook", string(body), resp.StatusCode, false}
	}

	var webhookData guildedWebhookResponse

	if err := json.NewDecoder(resp.Body).Decode(&webhookData); err != nil {
		return nil, fmt.Errorf("guilded: failed to decode webhook data: %w\n\tchannel: %s", err, channel)
	}

	if webhookData.Webhook.Token == nil {
		return nil, &guildedWebhookTokenNilError{channel}
	}

	p.webhookIDsCache.Set(webhookData.Webhook.ID, true)

	return map[string]string{"id": webhookData.Webhook.ID, "token": *webhookData.Webhook.Token}, nil
}

func getChannel(token, channel string) (*guildedServerChannel, error) {
	resp, err := guildedMakeRequest(token, "GET", "/channels/"+channel, nil)
	if err != nil {
		return nil, fmt.Errorf("guilded: failed to get channel for setup: %w\n\tchannel: %s", err, channel)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("guilded: failed to read body when getting channel: %w\n\tchannel: %s", err, channel)
	}

	if resp.Body.Close() != nil {
		slog.Warn("guilded: failed to close request body when getting channel")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, &guildedStatusError{"failed to get channel", string(bodyBytes), resp.StatusCode, false}
	}

	var channelData guildedServerChannelResponse
	if err := json.Unmarshal(bodyBytes, &channelData); err != nil {
		return nil, fmt.Errorf("guilded: failed to unmarshal channel data: %w\n\tbody: %s", err, bodyBytes)
	}

	return &channelData.Channel, nil
}
