// Package matrix provides a [lightning.Plugin] implementation for Matrix.
// To use Matrix support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("matrix", matrix.New)
//
//	bot.UsePluginType("matrix", map[string]any{
//		// ...
//	})
package matrix

import (
	"context"
	"fmt"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

// New creates a new [lightning.Plugin] that provides Matrix support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"access_token": "", // a string with your Matrix bot token
//		"homeserver": "",  // a string with your Matrix homeserver URL
//		"mxid": "",        // a string with your Matrix bot's user ID
//	}
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid config"}
	}

	accessToken, ok := cfg["access_token"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid token"}
	}

	homeserver, ok := cfg["homeserver"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid homeserver"}
	}

	mxid, ok := cfg["mxid"].(string)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid mxid"}
	}

	client, err := mautrix.NewClient(homeserver, id.UserID(mxid), accessToken)
	if err != nil {
		return nil, fmt.Errorf("matrix: failed to create client: %w", err)
	}

	client.UserAgent = "lightning/" + lightning.VERSION

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		client.Syncer = mautrix.NewDefaultSyncer()
	}

	msgChannel := make(chan lightning.Message, 1000)
	editChannel := make(chan lightning.EditedMessage, 1000)

	setupEvents(syncer, client, msgChannel, editChannel)

	return &matrixPlugin{client, syncer, msgChannel, editChannel}, nil
}

type matrixPlugin struct {
	client *mautrix.Client
	syncer *mautrix.DefaultSyncer

	msgChannel  chan lightning.Message
	editChannel chan lightning.EditedMessage
}

func (*matrixPlugin) SetupChannel(_ string) (any, error) {
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (*matrixPlugin) SendCommandResponse(_ lightning.Message, _ *lightning.SendOptions, _ string) ([]string, error) {
	return nil, nil //nolint:nilnil // placeholder
}

func (p *matrixPlugin) SendMessage(message lightning.Message, _ *lightning.SendOptions) ([]string, error) {
	msg := format.RenderMarkdown(message.Content, true, false)

	msg.BeeperPerMessageProfile = &event.BeeperPerMessageProfile{
		ID:          message.Author.ID,
		Displayname: message.Author.Nickname,
		// TODO: avatar URL, cache, sendoptions
		HasFallback: false,
	}

	msg.AddPerMessageProfileFallback()

	resp, err := p.client.SendMessageEvent(
		context.Background(),
		id.RoomID(message.ChannelID),
		event.EventMessage,
		msg,
		mautrix.ReqSendEvent{},
	)
	if err != nil {
		return nil, handleError(err, "failed to send matrix message",
			map[string]any{"channel": message.ChannelID, "content": message.Content})
	}

	return []string{resp.EventID.String()}, nil
}

func (*matrixPlugin) EditMessage(_ lightning.Message, _ []string, _ *lightning.SendOptions) error {
	return nil
}

func (p *matrixPlugin) DeleteMessage(channel string, ids []string) error {
	for _, msgID := range ids {
		if _, err := p.client.RedactEvent(
			context.Background(), id.RoomID(channel), id.EventID(msgID), mautrix.ReqRedact{Reason: "deleted in bridge"},
		); err != nil {
			return handleError(err, "Failed to redact Matrix message",
				map[string]any{"channel": channel, "message_id": msgID})
		}
	}

	return nil
}

func (*matrixPlugin) SetupCommands(_ map[string]lightning.Command) error {
	return nil
}

func (p *matrixPlugin) ListenMessages() <-chan lightning.Message {
	return p.msgChannel
}

func (p *matrixPlugin) ListenEdits() <-chan lightning.EditedMessage {
	return p.editChannel
}

func (p *matrixPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	channel := make(chan lightning.BaseMessage, 1000)

	p.syncer.OnEventType(event.EventRedaction, func(_ context.Context, evt *event.Event) {
		channel <- lightning.BaseMessage{
			Time:      time.UnixMilli(evt.Timestamp),
			EventID:   evt.Content.AsRedaction().Redacts.String(),
			ChannelID: evt.RoomID.String(),
		}
	})

	return channel
}

func (*matrixPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return nil
}
