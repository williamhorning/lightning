// Package revolt provides a [lightning.Plugin] implementation for Revolt.
// To use Revolt support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("revolt", revolt.New)
//
//	bot.UsePluginType("revolt", "", map[string]string{
//		// ...
//	})
package revolt

import (
	"fmt"
	"log"
	"time"

	"github.com/williamhorning/lightning/internal/rvapi"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Revolt support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"token": "", // a string with your Revolt bot token
//	}
func New(cfg map[string]string) (lightning.Plugin, error) {
	plugin := &revoltPlugin{session: &rvapi.Session{
		MessageDeleted: make(chan *rvapi.MessageDeleteEvent, 1000),
		MessageCreated: make(chan *rvapi.MessageEvent, 1000),
		MessageUpdated: make(chan *rvapi.MessageUpdateEvent, 1000),
		Ready:          make(chan *rvapi.ReadyEvent, 100),
		Token:          cfg["token"],
	}}
	plugin.self = plugin.session.User("@me")

	if plugin.self == nil {
		return nil, lightning.PluginConfigError{Plugin: "revolt", Message: "failed to get self user"}
	}

	go func() {
		for ready := range plugin.session.Ready {
			log.Printf("revolt: ready as %s in %d servers!\n", plugin.self.Username, len(ready.Servers))
			log.Printf("revolt: https://app.revolt.chat/invite/%s\n", plugin.self.ID)
		}
	}()

	if err := plugin.session.Connect(); err != nil {
		return nil, fmt.Errorf("revolt: failed to connect to socket: %w", err)
	}

	return plugin, nil
}

type revoltPlugin struct {
	self    *rvapi.User
	session *rvapi.Session
}

const correctPermissionValue = rvapi.PermissionManageCustomization | rvapi.PermissionChangeNickname |
	rvapi.PermissionChangeAvatar | rvapi.PermissionViewChannel | rvapi.PermissionReadMessageHistory |
	rvapi.PermissionSendMessage | rvapi.PermissionManageMessages | rvapi.PermissionInviteOthers |
	rvapi.PermissionSendEmbeds | rvapi.PermissionUploadFiles | rvapi.PermissionMasquerade |
	rvapi.PermissionReact

func (p *revoltPlugin) SetupChannel(channel string) (any, error) {
	channelData := p.session.Channel(channel)
	needed := correctPermissionValue

	if channelData.ChannelType == rvapi.ChannelTypeGroup {
		needed &= ^rvapi.PermissionManageCustomization
		needed &= ^rvapi.PermissionChangeNickname
		needed &= ^rvapi.PermissionChangeAvatar
	}

	permissions := p.session.GetPermissions(p.self, channelData)

	if permissions&needed == needed {
		return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
	}

	return nil, &revoltPermissionsError{permissions, needed}
}

func (p *revoltPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	user string,
) ([]string, error) {
	var channel rvapi.Channel

	if err := rvapi.Get(p.session, "/users/"+user+"/dm", &channel); err != nil {
		return nil, fmt.Errorf("revolt: failed to get dm channel: %w", err)
	}

	message.ChannelID = channel.ID

	return p.SendMessage(message, opts)
}

func (p *revoltPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	msg := p.getOutgoing(message, opts)
	leftover := make([]string, 0)

	if len(msg.Attachments) > 5 {
		leftover = msg.Attachments[5:]
		msg.Attachments = msg.Attachments[:5]
	}

	res, err := p.revoltSendMessage(message.ChannelID, msg)
	if err != nil {
		return nil, err
	}

	ids := []string{res}

	if len(leftover) > 0 {
		res, err := p.revoltSendMessage(message.ChannelID, rvapi.DataMessageSend{
			Attachments: leftover,
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

func (*revoltPlugin) SetupCommands(_ map[string]*lightning.Command) error {
	return nil
}

func (p *revoltPlugin) ListenMessages() <-chan *lightning.Message {
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

func (p *revoltPlugin) ListenEdits() <-chan *lightning.EditedMessage {
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

func (p *revoltPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
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

func (*revoltPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
