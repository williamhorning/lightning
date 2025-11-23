package stoat

import (
	"encoding/json"
	"fmt"

	"github.com/williamhorning/lightning/internal/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *stoatPlugin) stoatSendMessage(channel string, message stoat.DataMessageSend) (string, error) {
	ch, err := stoat.Get(p.session, "/channels/"+channel, channel, &p.session.ChannelCache)

	if err == nil && message.Masquerade != nil &&
		(ch.ChannelType != stoat.ChannelTypeText && ch.ChannelType != stoat.ChannelTypeVoice) {
		message.Masquerade.Colour = ""
	}

	resp, code, err := p.session.Fetch("POST", "/channels/"+channel+"/messages", message, nil,
		map[string][]string{"Content-Type": {"application/json"}})
	if err != nil {
		return "", fmt.Errorf("stoat: error making send message request: %w", err)
	}

	defer resp.Close()

	if code != 200 {
		return "", &stoatStatusError{"failed to send stoat message", code, true}
	}

	var response stoat.Message
	if err := json.NewDecoder(resp).Decode(&response); err != nil {
		return "", fmt.Errorf("stoat: failed to decode %d response: %w", code, err)
	}

	return response.ID, nil
}

func (p *stoatPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	message.Attachments = nil
	outgoing := lightningToStoatMessage(p.session, message, opts)
	data := stoat.DataEditMessage{Content: outgoing.Content, Embeds: outgoing.Embeds}

	resp, code, err := p.session.Fetch("PATCH", "/channels/"+message.ChannelID+"/messages/"+ids[0], data, nil,
		map[string][]string{"Content-Type": {"application/json"}})
	if err != nil {
		return fmt.Errorf("stoat: error making edit request: %w", err)
	}

	defer resp.Close()

	if code != 200 {
		return &stoatStatusError{"failed to edit stoat message", code, true}
	}

	return nil
}

func (p *stoatPlugin) DeleteMessage(channel string, ids []string) error {
	resp, code, err := p.session.Fetch(
		"DELETE", "/channels/"+channel+"/messages/bulk",
		stoat.OptionsBulkDelete{IDs: ids},
		nil, map[string][]string{"Content-Type": {"application/json"}},
	)
	if err != nil {
		return fmt.Errorf("stoat: error making deletion request: %w", err)
	}

	defer resp.Close()

	if code != 204 {
		return &stoatStatusError{"failed to delete stoat messages", code, true}
	}

	return nil
}
