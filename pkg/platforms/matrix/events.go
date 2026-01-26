package matrix

import (
	"context"
	"log"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
	"maunium.net/go/mautrix/id"
)

func (p *matrixPlugin) listenForEvents(
	onEvent func(eventType event.Type, callback mautrix.EventHandler),
	joinRoom func(ctx context.Context, roomID id.RoomID) (resp *mautrix.RespJoinRoom, err error),
	client *mautrix.Client,
	regex string,
	msgChannel chan *lightning.Message,
	editChannel chan *lightning.EditedMessage,
	deleteChannel chan *lightning.BaseMessage,
) {
	onEvent(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.Content.AsMember().Membership == event.MembershipInvite {
			if _, err := joinRoom(ctx, evt.RoomID); err != nil {
				log.Printf("matrix: failed to join room %q: %v\n", evt.RoomID, err)
			}
		}
	})

	onEvent(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		msg := p.matrixToLightningMessage(ctx, evt, client, regex)
		if msg == nil {
			return
		}

		edit := evt.Content.AsMessage().NewContent
		if edit == nil {
			msgChannel <- msg

			return
		}

		if edit.FormattedBody == "" {
			edit.FormattedBody = edit.Body
		}

		newContent, _ := format.HTMLToMarkdownFull(nil, edit.FormattedBody)
		msg.Content = newContent

		editChannel <- &lightning.EditedMessage{Edited: time.UnixMilli(evt.Timestamp), Message: msg}
	})

	onEvent(event.EventRedaction, func(_ context.Context, evt *event.Event) {
		deleteChannel <- &lightning.BaseMessage{
			Time:      time.UnixMilli(evt.Timestamp),
			EventID:   string(evt.Content.AsRedaction().Redacts),
			ChannelID: string(evt.RoomID),
		}
	})
}
