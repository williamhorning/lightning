package revolt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/williamhorning/lightning/internal/rvapi"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// SendMessage returns a message ID after sending a message on Revolt.
func (p *revoltPlugin) revoltSendMessage(channel string, message rvapi.DataMessageSend) (string, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("rvapi: failed to marshal send: %w\n\tbody: %#+v", err, message)
	}

	resp, code, err := p.session.Fetch(http.MethodPost, "/channels/"+channel+"/messages", bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("revolt: error making send message request: %w", err)
	}

	defer func() {
		if err := resp.Close(); err != nil {
			log.Printf("revolt: failed to close send body: %v", err)
		}
	}()

	if code != http.StatusOK {
		return "", &revoltStatusError{"failed to send revolt message", code, false}
	}

	var response rvapi.Message
	if err := json.NewDecoder(resp).Decode(&response); err != nil {
		return "", fmt.Errorf("revolt: failed to decode %d response: %w", code, err)
	}

	return response.ID, nil
}

func (p *revoltPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	message.Attachments = nil
	outgoing := p.getOutgoing(message, opts)
	data := rvapi.DataEditMessage{Content: outgoing.Content, Embeds: outgoing.Embeds}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("revolt: failed to marshal edit: %w\n\tbody: %#+v", err, data)
	}

	resp, code, err := p.session.Fetch(
		http.MethodPatch, "/channels/"+message.ChannelID+"/messages/"+ids[0], bytes.NewBuffer(payload),
	)
	if err != nil {
		return fmt.Errorf("revolt: error making edit request: %w", err)
	}

	if err := resp.Close(); err != nil {
		log.Printf("revolt: failed to close edit body: %v", err)
	}

	if code != http.StatusOK {
		return &revoltStatusError{"failed to edit revolt message", code, true}
	}

	return nil
}

func (p *revoltPlugin) DeleteMessage(channel string, ids []string) error {
	payload, err := json.Marshal(&rvapi.OptionsBulkDelete{IDs: ids})
	if err != nil {
		return fmt.Errorf("revolt: failed to marshal deletion: %w", err)
	}

	resp, code, err := p.session.Fetch(
		http.MethodDelete, "/channels/"+channel+"/messages/bulk", bytes.NewBuffer(payload),
	)
	if err != nil {
		return fmt.Errorf("revolt: error making deletion request: %w", err)
	}

	defer func() {
		if err := resp.Close(); err != nil {
			log.Printf("revolt: failed to close deletion body: %v", err)
		}
	}()

	if code != http.StatusNoContent {
		return &revoltStatusError{"failed to delete revolt messages\n\tbody: " + string(payload), code, true}
	}

	return nil
}
