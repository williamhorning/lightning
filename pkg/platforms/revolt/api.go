package revolt

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *revoltPlugin) getChannel(channelID string) *revoltChannel {
	if channel, ok := p.channelCache.Get(channelID); ok {
		return &channel
	}

	resp, err := revoltMakeRequest(p.token, "GET", "/channels/"+channelID, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close body", "err", err)
		}
	}()

	var channel revoltChannel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil
	}

	p.channelCache.Set(channelID, channel)

	return &channel
}

func (p *revoltPlugin) getEmoji(emojiID string) *revoltEmoji {
	if emoji, ok := p.emojiCache.Get(emojiID); ok {
		return &emoji
	}

	resp, err := revoltMakeRequest(p.token, "GET", "/custom/emoji/"+emojiID, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close body", "err", err)
		}
	}()

	var emoji revoltEmoji
	if err := json.NewDecoder(resp.Body).Decode(&emoji); err != nil {
		return nil
	}

	p.emojiCache.Set(emojiID, emoji)

	return &emoji
}

func (p *revoltPlugin) getMember(serverID, userID string) *revoltServerMember {
	cacheKey := serverID + "-" + userID
	if member, ok := p.memberCache.Get(cacheKey); ok {
		return &member
	}

	resp, err := revoltMakeRequest(p.token, "GET", "/servers/"+serverID+"/members/"+userID, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close body", "err", err)
		}
	}()

	var member revoltServerMember
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil
	}

	p.memberCache.Set(cacheKey, member)

	return &member
}

func (p *revoltPlugin) getServer(serverID string) *revoltServer {
	if server, ok := p.serverCache.Get(serverID); ok {
		return &server
	}

	resp, err := revoltMakeRequest(p.token, "GET", "/servers/"+serverID, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close body", "err", err)
		}
	}()

	var server revoltServer
	if err := json.NewDecoder(resp.Body).Decode(&server); err != nil {
		return nil
	}

	p.serverCache.Set(serverID, server)

	return &server
}

func (p *revoltPlugin) getUser(userID string) *revoltUser {
	if user, ok := p.userCache.Get(userID); ok {
		return &user
	}

	resp, err := revoltMakeRequest(p.token, "GET", "/users/"+userID, nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close body", "err", err)
		}
	}()

	var user revoltUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil
	}

	p.userCache.Set(userID, user)

	return &user
}

func revoltMakeRequest(token, method, endpoint string, body io.Reader) (*http.Response, error) {
	url := "https://api.revolt.chat/0.8" + endpoint

	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		return nil, lightning.LogError(err, "revolt: failed to create request", nil, nil)
	}

	req.Header.Set("X-Bot-Token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "rvapi/0.0.5")

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		return resp, nil
	}

	return nil, lightning.LogError(err, "revolt: failed to make api request", nil, nil)
}

func sendRevoltMessage(token, channel string, message revoltMessageSend) (string, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return "", lightning.LogError(err, "revolt: failed to marshal message", nil, nil)
	}

	resp, err := revoltMakeRequest(
		token,
		http.MethodPost,
		"/channels/"+channel+"/messages",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close send body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", revoltStatusError{"failed to send revolt message", resp.StatusCode}
	}

	var response revoltMessage
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", lightning.LogError(err, "revolt: failed to decode response", nil, nil)
	}

	return response.ID, nil
}

func editRevoltMessage(token, channel, messageID string, message revoltMessageEditData) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return lightning.LogError(err, "revolt: failed to marshal message", nil, nil)
	}

	resp, err := revoltMakeRequest(
		token,
		http.MethodPatch,
		"/channels/"+channel+"/messages/"+messageID,
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close edit body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return revoltStatusError{"failed to edit revolt message", resp.StatusCode}
	}

	return nil
}

func bulkDeleteRevoltMessages(token, channel string, body revoltChannelMessageBulkDeleteData) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return lightning.LogError(err, "revolt: failed to marshal deletion", nil, nil)
	}

	resp, err := revoltMakeRequest(
		token,
		http.MethodDelete,
		"/channels/"+channel+"/bulk",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("revolt: failed to close bulk delete body", "err", err)
		}
	}()

	if resp.StatusCode != http.StatusNoContent {
		return revoltStatusError{"failed to delete revolt message", resp.StatusCode}
	}

	return nil
}
