package guilded

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *guildedPlugin) SetupChannel(channel string) (any, error) {
	resp, err := guildedMakeRequest(p.token, "GET", "/channels/"+channel, nil)

	if err != nil {
		return nil, lightning.LogError(err, "Failed to get channel for setup", nil, lightning.ReadWriteDisabled{})
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to read response body", nil, lightning.ReadWriteDisabled{})
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, lightning.LogError(
			errors.New("Failed to get channel: "+strconv.Itoa(resp.StatusCode)),
			"Failed to get channel for setup",
			map[string]any{"status": resp.StatusCode, "body": string(bodyBytes)},
			lightning.ReadWriteDisabled{},
		)
	}

	var channelData struct {
		Channel guildedServerChannel `json:"channel"`
	}

	if err := json.Unmarshal(bodyBytes, &channelData); err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to unmarshal channel data",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	body, _ := json.Marshal(map[string]string{"channelId": channel, "name": "Lightning Bridges"})
	var reader io.Reader = bytes.NewReader(body)

	resp, err = guildedMakeRequest(p.token, "POST", "/servers/"+channelData.Channel.ServerId+"/webhooks", &reader)

	if err != nil {
		return nil, lightning.LogError(err, "Failed to create webhook for channel", nil, lightning.ReadWriteDisabled{})
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		extra := map[string]any{"status": resp.StatusCode, "body": string(body)}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			extra["resp"] = string(bodyBytes)
		}

		return nil, lightning.LogError(
			errors.New("Failed to create webhook: "+strconv.Itoa(resp.StatusCode)),
			"Failed to create webhook for channel",
			extra,
			lightning.ReadWriteDisabled{},
		)
	}

	var webhookData struct {
		Webhook guildedWebhook `json:"webhook"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&webhookData); err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to decode webhook data",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	if webhookData.Webhook.Token == nil {
		return nil, lightning.LogError(
			errors.New("webhook token is nil"),
			"Failed to create webhook for channel",
			map[string]any{"channelID": channel, "webhook": webhookData},
			lightning.ReadWriteDisabled{},
		)
	}

	return map[string]string{"id": webhookData.Webhook.Id, "token": *webhookData.Webhook.Token}, nil
}
