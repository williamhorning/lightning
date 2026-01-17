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
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Stoat support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"token": "", // a string with your Stoat bot token
//	}
func New(cfg map[string]string) (lightning.Plugin, error) {
	plugin := &stoatPlugin{session: &session{
		messageDeleted: make(chan *stMessageDeleteEvent, 1000),
		messageCreated: make(chan *stMessage, 1000),
		messageUpdated: make(chan *stMessageUpdateEvent, 1000),
		ready:          make(chan *stReadyEvent, 100),
		token:          cfg["token"],
	}}

	var err error

	plugin.self, err = get(plugin.session, "/users/@me", "@me", &plugin.session.userCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get own user: %w", err)
	}

	go func() {
		for ready := range plugin.session.ready {
			log.Printf("stoat: ready as %s in %d servers! https://stoat.chat/bot/%s\n",
				plugin.self.Username, len(ready.Servers), plugin.self.ID)
		}
	}()

	if err = plugin.session.connect(); err != nil {
		return nil, fmt.Errorf("stoat: failed to connect to websocket: %w", err)
	}

	return plugin, nil
}

type stoatPlugin struct {
	self    *stUser
	session *session
}

const correctPermissionValue = stPermissionManageCustomization | stPermissionManageRole |
	stPermissionChangeNickname | stPermissionChangeAvatar | stPermissionViewChannel |
	stPermissionReadMessageHistory | stPermissionSendMessage | stPermissionManageMessages |
	stPermissionInviteOthers | stPermissionSendEmbeds | stPermissionUploadFiles |
	stPermissionMasquerade | stPermissionReact

func (p *stoatPlugin) SetupChannel(channel string) (map[string]string, error) {
	channelData, err := get(p.session, "/channels/"+channel, channel, &p.session.channelCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get current channel: %w", err)
	}

	needed := correctPermissionValue

	if channelData.ChannelType == "Group" {
		needed &= ^stPermissionManageCustomization
		needed &= ^stPermissionManageRole
		needed &= ^stPermissionChangeNickname
		needed &= ^stPermissionChangeAvatar
	}

	permissions := p.session.getPermissions(p.self, channelData)

	if permissions&needed == needed {
		return nil, nil //nolint:nilnil
	}

	return nil, &stoatPermissionsError{permissions, needed}
}

func (p *stoatPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	if opts.CommandResponse {
		channel, err := get(p.session, "/users/"+opts.CommandUser+"/dm", opts.CommandUser, &p.session.dmChannelCache)
		if err != nil {
			return nil, fmt.Errorf("failed to make dm channel for command response: %w", err)
		}

		message.ChannelID = channel.ID
	}

	chunks := lightningToStoatMessage(p.session, message, opts)
	ids := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		res, err := p.session.sendMessage(message.ChannelID, &chunk)
		if err != nil {
			return ids, err
		}

		ids = append(ids, res)
	}

	return ids, nil
}

func (p *stoatPlugin) EditMessage(
	message *lightning.Message, ids []string, opts *lightning.SendOptions,
) ([]string, error) {
	if opts.CommandResponse {
		channel, err := get(p.session, "/users/"+opts.CommandUser+"/dm", opts.CommandUser, &p.session.dmChannelCache)
		if err != nil {
			return nil, fmt.Errorf("failed to make dm channel for command response: %w", err)
		}

		message.ChannelID = channel.ID
	}

	message.Attachments = nil

	chunks := lightningToStoatMessage(p.session, message, opts)

	for idx, chunk := range chunks {
		if _, err := fetch[any](p.session, "PATCH", "https://api.stoat.chat/0.8/channels/"+message.ChannelID+
			"/messages/"+ids[idx], "application/json",
			stDataEditMessage{Content: chunk.Content, Embeds: chunk.Embeds}); err != nil {
			return ids, fmt.Errorf("failed to edit message: %w", err)
		}
	}

	return ids, nil
}

func (p *stoatPlugin) DeleteMessage(channel string, ids []string) error {
	if _, err := fetch[any](p.session, "DELETE", "https://api.stoat.chat/0.8/channels/"+channel+"/messages/bulk",
		"application/json", map[string][]string{"ids": ids}); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}

func (*stoatPlugin) SetupCommands(_ map[string]lightning.Command) {}

func (p *stoatPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	go func() {
		for m := range p.session.messageCreated {
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
		for m := range p.session.messageUpdated {
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
		for m := range p.session.messageDeleted {
			channel <- &lightning.BaseMessage{EventID: m.ID, ChannelID: m.Channel, Time: time.Now()}
		}
	}()

	return channel
}

func (*stoatPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
