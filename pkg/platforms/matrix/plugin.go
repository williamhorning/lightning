// Package matrix provides a [lightning.Plugin] implementation for Matrix.
// To use Matrix support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("matrix", matrix.New)
//
//	bot.UsePluginType("matrix", map[string]string{
//		// ...
//	})
package matrix

import (
	"context"
	"log"
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
//	map[string]string{
//		"homeserver": "",  // a string with your Matrix homeserver URL
//		"password": "", // a string with your Matrix bot password
//		"recovery_key": "", // a string with your Matrix bot recovery key
//		"username": "", // a string with your Matrix bot username
//	}
func New(config map[string]string) (lightning.Plugin, error) {
	client, err := setupClient(config)
	if err != nil {
		return nil, err
	}

	syncer, ok := client.Syncer.(*mautrix.DefaultSyncer)
	if !ok {
		syncer = mautrix.NewDefaultSyncer()
		client.Syncer = syncer
	}

	msgChannel := make(chan *lightning.Message, 1000)
	editChannel := make(chan *lightning.EditedMessage, 1000)

	listenForEvents(syncer, client, msgChannel, editChannel)

	go func() {
		for {
			if err := client.Sync(); err != nil {
				log.Printf("matrix: sync stopped: %v, retrying...", err)
			}
		}
	}()

	return &matrixPlugin{client: client, syncer: syncer, msgChannel: msgChannel, editChannel: editChannel}, nil
}

type matrixPlugin struct {
	client      *mautrix.Client
	syncer      *mautrix.DefaultSyncer
	msgChannel  chan *lightning.Message
	editChannel chan *lightning.EditedMessage
	mxcCache    cache.Expiring[string, id.ContentURIString]
}

func (*matrixPlugin) SetupChannel(_ string) (map[string]string, error) {
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

	for _, msg := range p.lightningToMatrixMessage(message, nil, opts) {
		resp, err := p.client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return nil, handleError(err, "failed to send matrix message")
		}

		ids = append(ids, string(resp.EventID))
	}

	return ids, nil
}

func (p *matrixPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	for idx, msg := range p.lightningToMatrixMessage(message, ids, opts) {
		msg.RelatesTo.Type = "m.replace"
		msg.RelatesTo.EventID = id.EventID(ids[idx])

		_, err := p.client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return handleError(err, "failed to edit matrix message")
		}
	}

	return nil
}

func (p *matrixPlugin) DeleteMessage(channel string, ids []string) error {
	for _, msgID := range ids {
		if _, err := p.client.RedactEvent(
			context.Background(), id.RoomID(channel), id.EventID(msgID), mautrix.ReqRedact{Reason: "deleted in bridge"},
		); err != nil {
			return handleError(err, "Failed to redact Matrix message")
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
		channel <- &lightning.BaseMessage{
			Time:      time.UnixMilli(evt.Timestamp),
			EventID:   string(evt.Content.AsRedaction().Redacts),
			ChannelID: string(evt.RoomID),
		}
	})

	return channel
}

func (*matrixPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
