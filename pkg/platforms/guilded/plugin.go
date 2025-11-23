// Package guilded provides a [lightning.Plugin] implementation for Guilded.
// To use Guilded support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("guilded", guilded.New)
//
//	bot.UsePluginType("guilded", "", map[string]string{
//		// ...
//	})
package guilded

import (
	"fmt"
	"log"

	"github.com/williamhorning/lightning/internal/cache"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Guilded support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]string{
//		"token": "", // a string with your Guilded bot token
//	}
func New(cfg map[string]string) (lightning.Plugin, error) {
	plugin := &guildedPlugin{socket: &session{
		messageDeleted: make(chan *guildedChatMessageDeleted, 1000),
		messageCreated: make(chan *guildedChatMessageWrapper, 1000),
		messageUpdated: make(chan *guildedChatMessageWrapper, 1000),
		token:          cfg["token"],
	}, token: cfg["token"]}

	plugin.assetsCache.TTL = assetCacheTTL

	if err := plugin.socket.connect(); err != nil {
		return nil, fmt.Errorf("guilded: failed to connect to socket: %w", err)
	}

	log.Println("guilded: ready")

	return plugin, nil
}

type guildedPlugin struct {
	socket          *session
	token           string
	assetsCache     cache.Expiring[string, lightning.Attachment]
	membersCache    cache.Expiring[string, guildedServerMember]
	webhookIDsCache cache.Expiring[string, bool]
}

func (*guildedPlugin) SetupChannel(_ string) (map[string]string, error) {
	return nil, &guildedShuttingDownError{}
}

func (p *guildedPlugin) SendCommandResponse(
	message *lightning.Message, opts *lightning.SendOptions, _ string,
) ([]string, error) {
	return p.SendMessage(message, opts)
}

func (*guildedPlugin) EditMessage(_ *lightning.Message, _ []string, _ *lightning.SendOptions) error {
	return nil
}

func (p *guildedPlugin) DeleteMessage(channel string, ids []string) error {
	for _, msgID := range ids {
		resp, err := guildedMakeRequest(p.token, "DELETE", "/channels/"+channel+"/messages/"+msgID, nil)
		if err != nil {
			return fmt.Errorf("guilded: failed to delete message: %w\n\tchannel %s\n\tmessage: %s", err, channel, msgID)
		}

		_ = resp.Body.Close()
	}

	return nil
}

func (*guildedPlugin) SetupCommands(_ map[string]*lightning.Command) error {
	return nil
}

func (p *guildedPlugin) ListenMessages() <-chan *lightning.Message {
	channel := make(chan *lightning.Message, 1000)

	go func() {
		for msg := range p.socket.messageCreated {
			if message := guildedToLightning(p, &msg.Message); message != nil {
				channel <- message
			}
		}
	}()

	return channel
}

func (p *guildedPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	channel := make(chan *lightning.EditedMessage, 1000)

	go func() {
		for msg := range p.socket.messageUpdated {
			if message := guildedToLightning(p, &msg.Message); message != nil {
				channel <- &lightning.EditedMessage{Message: message, Edited: msg.Message.UpdatedAt}
			}
		}
	}()

	return channel
}

func (p *guildedPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	channel := make(chan *lightning.BaseMessage, 1000)

	go func() {
		for msg := range p.socket.messageDeleted {
			channel <- &lightning.BaseMessage{
				EventID: msg.Message.ID, ChannelID: msg.Message.ChannelID, Time: msg.DeletedAt,
			}
		}
	}()

	return channel
}

func (*guildedPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
