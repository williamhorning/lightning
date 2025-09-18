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
	"time"

	"github.com/williamhorning/lightning/internal/cache"
	"github.com/williamhorning/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// New creates a new [lightning.Plugin] that provides Matrix support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"access_token": "", // a string with your Matrix bot's token.
//						    // note: this should be set after initial login
//		"device_id": "", // a string with your Matrix bot's device ID.
//					     // note: this should be set after initial login
//		"homeserver": "",  // a string with your Matrix homeserver URL.
//						   // note: this MUST be set
//		"mxid": "",  // a string with your Matrix homeserver URL.
//					 // note: this should be set after initial login
//		"password": "", // a string with your Matrix bot password
//					    // note: this MUST be set
//		"random": "", // a random encryption key which MUST be set
//		"recovery_key": "", // a string with your Matrix bot recovery key
//					        // note: this MUST be set
//		"username": "", // a string with your Matrix bot username
//					    // note: this MUST be set
//	}
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(matrixConfig)
	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "matrix", Message: "invalid config"}
	}

	client, err := setupClient(cfg)
	if err != nil {
		return nil, err
	}

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		client.Syncer = mautrix.NewDefaultSyncer()
	}

	msgChannel := make(chan *lightning.Message, 1000)
	editChannel := make(chan *lightning.EditedMessage, 1000)

	setupEvents(syncer, client, msgChannel, editChannel)

	return &matrixPlugin{
		client, syncer, cache.New[string, id.ContentURIString](cache.DefaultTTL),
		msgChannel, editChannel,
	}, nil
}

type matrixPlugin struct {
	client *mautrix.Client
	syncer *mautrix.DefaultSyncer

	mxcCache *cache.Expiring[string, id.ContentURIString]

	msgChannel  chan *lightning.Message
	editChannel chan *lightning.EditedMessage
}

func (*matrixPlugin) SetupChannel(_ string) (any, error) {
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *matrixPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	_ string,
) ([]string, error) {
	return p.SendMessage(message, opts)
}

func (p *matrixPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	ids := make([]string, 0, len(message.Attachments)+1)

	for _, msg := range p.getOutgoing(message, nil, opts) {
		resp, err := p.client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return nil, handleError(err, "failed to send matrix message",
				map[string]any{"channel": message.ChannelID, "content": message.Content})
		}

		ids = append(ids, string(resp.EventID))
	}

	return ids, nil
}

func (p *matrixPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	for idx, msg := range p.getOutgoing(message, ids, opts) {
		msg.RelatesTo.SetReplace(id.EventID(ids[idx]))

		_, err := p.client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return handleError(err, "failed to edit matrix message",
				map[string]any{"channel": message.ChannelID, "content": message.Content})
		}
	}

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

func (*matrixPlugin) SetupCommands(_ map[string]*lightning.Command) error {
	return nil
}

func (p *matrixPlugin) ListenMessages() <-chan *lightning.Message {
	return p.msgChannel
}

func (p *matrixPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	return p.editChannel
}

func (p *matrixPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	p.syncer.OnEventType(event.EventRedaction, func(_ context.Context, evt *event.Event) {
		timestamp := time.UnixMilli(evt.Timestamp)

		channel <- &lightning.BaseMessage{
			Time:      &timestamp,
			EventID:   evt.Content.AsRedaction().Redacts.String(),
			ChannelID: evt.RoomID.String(),
		}
	})

	return channel
}

func (*matrixPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
