package stoat

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/williamhorning/lightning/internal/v2/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func (p *stoatPlugin) stoatSendMessage(channel string, message stoat.DataMessageSend) (string, error) {
	p.clearMasqueradeColour(channel, &message)

	resp, code, err := p.session.Fetch("POST", "/channels/"+channel+"/messages", message, nil,
		map[string][]string{"Content-Type": {"application/json"}})
	if err != nil {
		return "", fmt.Errorf("stoat: error making send message request: %w", err)
	}

	defer func() {
		if err = resp.Close(); err != nil {
			log.Printf("stoat: failed to close send body: %v\n", err)
		}
	}()

	if code != 200 {
		return "", &stoatStatusError{"failed to send stoat message", nil, code, true}
	}

	var response stoat.Message
	if err := json.NewDecoder(resp).Decode(&response); err != nil {
		return "", fmt.Errorf("stoat: failed to decode %d response: %w", code, err)
	}

	return response.ID, nil
}

func (p *stoatPlugin) clearMasqueradeColour(channel string, msg *stoat.DataMessageSend) {
	ch := stoat.Get(p.session, "/channels/"+channel, channel, &p.session.ChannelCache)

	if ch != nil && msg.Masquerade != nil &&
		(ch.ChannelType != stoat.ChannelTypeText && ch.ChannelType != stoat.ChannelTypeVoice) {
		msg.Masquerade.Colour = ""
	}
}

func (p *stoatPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	message.Attachments = nil
	outgoing := p.getOutgoing(message, opts)
	data := stoat.DataEditMessage{Content: outgoing.Content, Embeds: outgoing.Embeds}

	resp, code, err := p.session.Fetch("PATCH", "/channels/"+message.ChannelID+"/messages/"+ids[0], data, nil,
		map[string][]string{"Content-Type": {"application/json"}})
	if err != nil {
		return fmt.Errorf("stoat: error making edit request: %w", err)
	}

	if err := resp.Close(); err != nil {
		log.Printf("stoat: failed to close edit body: %v\n", err)
	}

	if code != 200 {
		return &stoatStatusError{"failed to edit stoat message", nil, code, true}
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

	if err := resp.Close(); err != nil {
		log.Printf("stoat: failed to close deletion body: %v\n", err)
	}

	if code != 204 {
		return &stoatStatusError{"failed to delete stoat messages", nil, code, true}
	}

	return nil
}
