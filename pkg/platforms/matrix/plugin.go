// Package matrix provides a [lightning.Plugin] implementation for Matrix.
// Note that this implementation may not be
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
	"fmt"
	"log"

	"codeberg.org/jersey/lightning/internal/cache"
	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/appservice"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

// New creates a new [lightning.Plugin] that provides Matrix support for Lightning
//
// It only takes in a map with the following structure:
//
//		map[string]string{
//			"access_token": "", // a string with your Matrix bot token, sets up a bot using MSC4144 if specified
//	 		"as_url": "",       // a string with the URL your appservice is at
//	 		"as_token": "",     // a string with the random token your appservice will use
//	 		"as_local": "",     // a string with the localpart your appservice will use
//	 		"hs_token": "",     // a string with the random token your homeserver will use
//			"homeserver": "",   // a string with your Matrix homeserver URL (always required)
//			"proxy_port": "",   // a string with your proxy port for files (always required)
//	 		"proxy_url": "",    // a string with your proxy url for files (always required)
//			"mxid": "",         // a string with your Matrix bot ID
//		}
func New(config map[string]string) (lightning.Plugin, error) {
	if config["access_token"] != "" {
		return getBot(config)
	}

	return getAppsvc(config)
}

type matrixPlugin struct {
	proxy         string
	appsvc        *appservice.AppService
	client        *mautrix.Client
	msgChannel    chan *lightning.Message
	editChannel   chan *lightning.EditedMessage
	deleteChannel chan *lightning.BaseMessage
	mxcCache      cache.Expiring[string, id.ContentURIString]
}

func (p *matrixPlugin) IsAdmin(user, channel string) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("matrix: panic on IsAdmin: %v", r)
		}
	}()

	levels, err := p.client.StateStore.GetPowerLevels(context.Background(), id.RoomID(channel))
	if err != nil || levels == nil || levels.CreateEvent == nil {
		return false, fmt.Errorf("state not synced yet, failed to get power levels: %w", err)
	} else if levels.GetUserLevel(id.UserID(user)) >= 60 {
		return true, nil
	}

	return false, nil
}

func (*matrixPlugin) SetupChannel(_ string) (map[string]string, error) {
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *matrixPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	ids := make([]string, 0, len(message.Attachments)+1)
	client, fallback := p.getClient(message)

	for _, msg := range p.lightningToMatrixMessage(client, message, opts, fallback) {
		resp, err := client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, msg, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return nil, handleError(err, "send")
		}

		ids = append(ids, string(resp.EventID))
	}

	return ids, nil
}

func (p *matrixPlugin) EditMessage(
	message *lightning.Message, ids []string, opts *lightning.SendOptions,
) ([]string, error) {
	message.Attachments = nil
	client, fallback := p.getClient(message)

	for idx, msg := range p.lightningToMatrixMessage(client, message, opts, fallback) {
		sendable := event.MessageEventContent{MsgType: event.MsgText}
		sendable.RelatesTo = &event.RelatesTo{Type: "m.replace", EventID: id.EventID(ids[idx])}
		sendable.NewContent = msg

		_, err := client.SendMessageEvent(
			context.Background(), id.RoomID(message.ChannelID), event.EventMessage, sendable, mautrix.ReqSendEvent{},
		)
		if err != nil {
			return nil, handleError(err, "edit")
		}
	}

	return ids, nil
}

func (p *matrixPlugin) DeleteMessage(channel string, ids []string) error {
	client, _ := p.getClient(nil)

	for _, msgID := range ids {
		if _, err := client.RedactEvent(
			context.Background(), id.RoomID(channel), id.EventID(msgID), mautrix.ReqRedact{Reason: "deleted in bridge"},
		); err != nil {
			return handleError(err, "redact")
		}
	}

	return nil
}

func (*matrixPlugin) SetupCommands(_ map[string]lightning.Command) {}

func (p *matrixPlugin) ListenMessages() <-chan *lightning.Message {
	return p.msgChannel
}

func (p *matrixPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	return p.editChannel
}

func (p *matrixPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	return p.deleteChannel
}

func (*matrixPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
