package matrix

import (
	"context"
	"log"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/format"
)

func listenForEvents(
	syncer *mautrix.DefaultSyncer,
	client *mautrix.Client,
	msgChannel chan *lightning.Message,
	editChannel chan *lightning.EditedMessage,
) {
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.Content.AsMember().Membership == event.MembershipInvite {
			if _, err := client.JoinRoomByID(ctx, evt.RoomID); err != nil {
				log.Printf("matrix: failed to join room: %v\n", err)
			}
		}
	})

	syncer.OnSync(func(ctx context.Context, resp *mautrix.RespSync, since string) bool {
		return since != "" || client.DontProcessOldEvents(ctx, resp, since)
	})

	syncer.OnEventType(event.EventMessage, mautrix.EventHandler(func(ctx context.Context, evt *event.Event) {
		msg := matrixToLightningMessage(ctx, evt, client)

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
	}))
}
