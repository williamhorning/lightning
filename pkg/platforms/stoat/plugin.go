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
	"math"
	"time"

	"github.com/williamhorning/lightning/internal/rvapi"
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
	plugin := &stoatPlugin{session: &rvapi.Session{
		MessageDeleted: make(chan *rvapi.MessageDeleteEvent, 1000),
		MessageCreated: make(chan *rvapi.MessageEvent, 1000),
		MessageUpdated: make(chan *rvapi.MessageUpdateEvent, 1000),
		Ready:          make(chan *rvapi.ReadyEvent, 100),
		Token:          cfg["token"],
	}}
	plugin.self = plugin.session.User("@me")

	if plugin.self == nil {
		return nil, lightning.PluginConfigError{Plugin: "stoat", Message: "failed to get self user"}
	}

	go func() {
		for ready := range plugin.session.Ready {
			log.Printf("stoat: ready as %s in %d servers!\n", plugin.self.Username, len(ready.Servers))
			log.Printf("stoat: https://app.stoat.chat/invite/%s\n", plugin.self.ID)
		}
	}()

	if err := plugin.session.Connect(); err != nil {
		return nil, fmt.Errorf("stoat: failed to connect to socket: %w", err)
	}

	return plugin, nil
}

type stoatPlugin struct {
	self    *rvapi.User
	session *rvapi.Session
}

const correctPermissionValue = rvapi.PermissionManageCustomization | rvapi.PermissionManageRole |
	rvapi.PermissionChangeNickname | rvapi.PermissionChangeAvatar | rvapi.PermissionViewChannel |
	rvapi.PermissionReadMessageHistory | rvapi.PermissionSendMessage | rvapi.PermissionManageMessages |
	rvapi.PermissionInviteOthers | rvapi.PermissionSendEmbeds | rvapi.PermissionUploadFiles |
	rvapi.PermissionMasquerade | rvapi.PermissionReact

func (p *stoatPlugin) SetupChannel(channel string) (any, error) {
	channelData := p.session.Channel(channel)
	needed := correctPermissionValue

	if channelData.ChannelType == rvapi.ChannelTypeGroup {
		needed &= ^rvapi.PermissionManageCustomization
		needed &= ^rvapi.PermissionManageRole
		needed &= ^rvapi.PermissionChangeNickname
		needed &= ^rvapi.PermissionChangeAvatar
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
	var channel rvapi.Channel

	if err := rvapi.Get(p.session, "/users/"+user+"/dm", &channel); err != nil {
		return nil, fmt.Errorf("stoat: failed to get dm channel: %w", err)
	}

	message.ChannelID = channel.ID

	return p.SendMessage(message, opts)
}

func (p *stoatPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := p.getOutgoing(message, opts)
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

	chunks := make([][]string, 0, int(math.Ceil(float64(len(leftover))/5)))

	for i := 0; i < len(leftover); i += 5 {
		end := min(i+5, len(leftover))

		chunks = append(chunks, leftover[i:end])
	}

	for _, chunk := range chunks {
		res, err := p.stoatSendMessage(message.ChannelID, rvapi.DataMessageSend{
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
			if msg := p.getIncomingMessage(m.Message); msg != nil {
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
			if msg := p.getIncomingMessage(m.Data); msg != nil {
				channel <- &lightning.EditedMessage{
					Message: msg,
					Edited:  &m.Data.Edited,
				}
			}
		}
	}()

	return channel
}

func (p *stoatPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	go func() {
		for m := range p.session.MessageDeleted {
			timestamp := time.Now()
			channel <- &lightning.BaseMessage{
				EventID:   m.ID,
				ChannelID: m.Channel,
				Time:      &timestamp,
			}
		}
	}()

	return channel
}

func (*stoatPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
