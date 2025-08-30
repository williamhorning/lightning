package revolt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

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
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
		}
	}()

	var channel revoltChannel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil
	}

	p.channelCache.Set(channelID, channel)

	return &channel
}

func (p *revoltPlugin) getDMChannel(user string) *revoltChannel {
	if channel, ok := p.dmChannelCache.Get(user); ok {
		return &channel
	}

	resp, err := revoltMakeRequest(p.token, "GET", "/users/"+user+"/dm", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
		}
	}()

	var channel revoltChannel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil
	}

	p.dmChannelCache.Set(user, channel)

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
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
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
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
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
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
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
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
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
		return nil, fmt.Errorf("revolt: failed to create request: %w\n\tendpoint: %s\n\tmethod: %s",
			err, endpoint, method)
	}

	req.Header.Set("X-Bot-Token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "lightning/"+lightning.VERSION)

	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		return resp, nil
	}

	return nil, fmt.Errorf("revolt: failed to make request: %w\n\tendpoint: %s\n\tmethod: %s",
		err, endpoint, method)
}

func ratelimitRetry[T any](
	resp *http.Response,
	token, channel string,
	data T,
	self func(string, string, T) (string, error),
) (string, error) {
	retryAfter := resp.Header.Get("X-Ratelimit-Retry-After")

	if retryAfter == "" {
		retryAfter = "1000"
	}

	retryAfterDuration, err := time.ParseDuration(retryAfter + "ms")
	if err != nil {
		retryAfterDuration = time.Second
	}

	time.Sleep(retryAfterDuration)

	return self(token, channel, data)
}

func sendRevoltMessage(token, channel string, message revoltMessageSend) (string, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("revolt: failed to marshal message: %w\n\tmessage: %#+v", err, message)
	}

	resp, err := revoltMakeRequest(token, http.MethodPost, "/channels/"+channel+"/messages",
		bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		return ratelimitRetry(resp, token, channel, message, sendRevoltMessage)
	}

	if resp.StatusCode != http.StatusOK {
		return "", &revoltStatusError{"failed to send revolt message", resp.StatusCode, false}
	}

	var response revoltMessage
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("revolt: failed to decode %d response: %w", resp.StatusCode, err)
	}

	return response.ID, nil
}

func editRevoltMessage(token, channel, messageID string, message revoltMessageEditData) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("revolt: failed to marshal message: %w\n\tmessage: %#+v", err, message)
	}

	resp, err := revoltMakeRequest(token, http.MethodPatch, "/channels/"+channel+"/messages/"+messageID,
		bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		_, err := ratelimitRetry(resp, token, channel, message,
			func(_, _ string, _ revoltMessageEditData) (string, error) {
				return "", editRevoltMessage(token, channel, messageID, message)
			})

		return err
	}

	if resp.StatusCode != http.StatusOK {
		return &revoltStatusError{"failed to edit revolt message", resp.StatusCode, true}
	}

	return nil
}

func bulkDeleteRevoltMessages(token, channel string, body revoltChannelMessageBulkDeleteData) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("revolt: failed to marshal deletion: %w\n\body: %#+v", err, body)
	}

	resp, err := revoltMakeRequest(token, http.MethodDelete, "/channels/"+channel+"/messages/bulk",
		bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn(fmt.Errorf("revolt: failed to close body: %w", err).Error())
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		_, err := ratelimitRetry(resp, token, channel, body,
			func(_, _ string, _ revoltChannelMessageBulkDeleteData) (string, error) {
				return "", bulkDeleteRevoltMessages(token, channel, body)
			})

		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return &revoltStatusError{"failed to delete revolt message", resp.StatusCode, true}
	}

	return nil
}
