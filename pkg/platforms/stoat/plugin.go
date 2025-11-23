// Package stoat provides a [lightning.Plugin] implementation for Stoat.
// To use Stoat support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("stoat", stoat.New)
//
//	bot.UsePluginType("stoat", "", map[string]string{
//		// ...
//	})
package stoat

import (
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/williamhorning/lightning/internal/stoat"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Stoat support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"token": "", // a string with your Stoat bot token
//	}
func New(cfg map[string]string) (lightning.Plugin, error) {
	plugin := &stoatPlugin{session: &stoat.Session{
		MessageDeleted: make(chan *stoat.MessageDeleteEvent, 1000),
		MessageCreated: make(chan *stoat.Message, 1000),
		MessageUpdated: make(chan *stoat.MessageUpdateEvent, 1000),
		Ready:          make(chan *stoat.ReadyEvent, 100),
		Token:          cfg["token"],
	}}

	var err error

	plugin.self, err = stoat.Get(plugin.session, "/users/@me", "@me", &plugin.session.UserCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get self: %w", err)
	}

	go func() {
		for ready := range plugin.session.Ready {
			log.Printf("stoat: ready as %s in %d servers!\n", plugin.self.Username, len(ready.Servers))
		}
	}()

	if err = plugin.session.Connect(); err != nil {
		return nil, fmt.Errorf("stoat: failed to connect to socket: %w", err)
	}

	return plugin, nil
}

type stoatPlugin struct {
	self    *stoat.User
	session *stoat.Session
}

const correctPermissionValue = stoat.PermissionManageCustomization | stoat.PermissionManageRole |
	stoat.PermissionChangeNickname | stoat.PermissionChangeAvatar | stoat.PermissionViewChannel |
	stoat.PermissionReadMessageHistory | stoat.PermissionSendMessage | stoat.PermissionManageMessages |
	stoat.PermissionInviteOthers | stoat.PermissionSendEmbeds | stoat.PermissionUploadFiles |
	stoat.PermissionMasquerade | stoat.PermissionReact

func (p *stoatPlugin) SetupChannel(channel string) (map[string]string, error) {
	channelData, err := stoat.Get(p.session, "/channels/"+channel, channel, &p.session.ChannelCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	needed := correctPermissionValue

	if channelData.ChannelType == stoat.ChannelTypeGroup {
		needed &= ^stoat.PermissionManageCustomization
		needed &= ^stoat.PermissionManageRole
		needed &= ^stoat.PermissionChangeNickname
		needed &= ^stoat.PermissionChangeAvatar
	}

	permissions := p.session.GetPermissions(p.self, channelData)

	if permissions&needed == needed {
		return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
	}

	return nil, &stoatPermissionsError{permissions, needed}
}

func (p *stoatPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	user string,
) ([]string, error) {
	channel, err := stoat.Get(p.session, "/users/"+user+"/dm", "", &p.session.ChannelCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get dm channel: %w", err)
	}

	message.ChannelID = channel.ID

	return p.SendMessage(message, opts)
}

func (p *stoatPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := lightningToStoatMessage(p.session, message, opts)
	leftover := make([]string, 0)

	if len(msg.Attachments) > 5 {
		leftover = msg.Attachments[5:]
		msg.Attachments = msg.Attachments[:5]
	}

	res, err := p.stoatSendMessage(message.ChannelID, msg)
	if err != nil {
		return nil, err
	}

	ids := []string{res}

	if len(leftover) == 0 {
		return ids, nil
	}

	for chunk := range slices.Chunk(leftover, 5) {
		res, err := p.stoatSendMessage(message.ChannelID, stoat.DataMessageSend{
			Attachments: chunk,
			Masquerade:  msg.Masquerade,
			Replies:     msg.Replies,
		})
		if err != nil {
			log.Printf("failed to send leftover attachments: %v\n\tleftover: %#+v\n", err, leftover)
		} else {
			ids = append(ids, res)
		}
	}

	return ids, nil
}

func (*stoatPlugin) SetupCommands(_ map[string]*lightning.Command) error {
	return nil
}

func (p *stoatPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	go func() {
		for m := range p.session.MessageCreated {
			if msg := stoatToLightningMessage(p.session, p.self.ID, m); msg != nil {
				channel <- msg
			}
		}
	}()

	return channel
}

func (p *stoatPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	go func() {
		for m := range p.session.MessageUpdated {
			if msg := stoatToLightningMessage(p.session, p.self.ID, &m.Data); msg != nil {
				channel <- &lightning.EditedMessage{Message: msg, Edited: m.Data.Edited}
			}
		}
	}()

	return channel
}

func (p *stoatPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	go func() {
		for m := range p.session.MessageDeleted {
			channel <- &lightning.BaseMessage{EventID: m.ID, ChannelID: m.Channel, Time: time.Now()}
		}
	}()

	return channel
}

func (*stoatPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
