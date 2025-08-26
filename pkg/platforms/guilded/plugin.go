// Package guilded provides a [lightning.Plugin] implementation for Guilded.
// To use Guilded support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("guilded", guilded.New)
//
//	bot.UsePluginType("guilded", "", map[string]any{
//		// ...
//	})
package guilded

import (
	"fmt"
	"log/slog"

	"github.com/williamhorning/lightning/internal/cache"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Guilded support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"token": "", // a string with your Guilded bot token
//	}
func New(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)

	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "guilded", Message: "invalid config"}
	}

	token, ok := cfg["token"].(string)

	if !ok {
		return nil, lightning.PluginConfigError{Plugin: "guilded", Message: "invalid token"}
	}

	socket := guildedNewSocketManager(token)
	plugin := &guildedPlugin{
		socket, cache.New[string, lightning.Attachment](assetCacheTTL),
		cache.New[string, guildedServerMember](defaultCacheTTL),
		cache.New[string, guildedWebhook](defaultCacheTTL),
		cache.New[string, bool](defaultCacheTTL), token,
	}

	socket.OnReady(func(msg *guildedWelcomeMessage) {
		slog.Info("guilded: ready!", "username", msg.User.Name)
	})

	if err := socket.Connect(); err != nil {
		return nil, fmt.Errorf("guilded: failed to connect to socket: %w", err)
	}

	return plugin, nil
}

type guildedPlugin struct {
	socket          *guildedSocketManager
	assetsCache     *cache.Expiring[string, lightning.Attachment]
	membersCache    *cache.Expiring[string, guildedServerMember]
	webhooksCache   *cache.Expiring[string, guildedWebhook]
	webhookIDsCache *cache.Expiring[string, bool]
	token           string
}

func (*guildedPlugin) EditMessage(_ lightning.Message, _ []string, _ *lightning.SendOptions) error {
	return nil
}

func (p *guildedPlugin) DeleteMessage(channel string, ids []string) error {
	for _, msgID := range ids {
		resp, err := guildedMakeRequest(p.token, "DELETE", "/channels/"+channel+"/messages/"+msgID, nil)

		if resp.Body.Close() != nil {
			slog.Warn("guilded: failed to close request body when deleting message")
		}

		if err != nil {
			slog.Error("guilded: failed to delete message", "error", err, "messageID", msgID, "channelID", channel)

			return fmt.Errorf("guilded: failed to delete message %s in channel %s: %w", msgID, channel, err)
		}
	}

	return nil
}

func (*guildedPlugin) SetupCommands(_ map[string]lightning.Command) error {
	return nil
}

func (p *guildedPlugin) ListenMessages() <-chan lightning.Message {
	channel := make(chan lightning.Message, 1000)

	p.socket.OnMessageCreated(func(msg *guildedChatMessageCreated) {
		message := p.getIncomingMessage(&msg.Message)
		if message != nil {
			channel <- *message
		}
	})

	return channel
}

func (p *guildedPlugin) ListenEdits() <-chan lightning.EditedMessage {
	channel := make(chan lightning.EditedMessage, 1000)

	p.socket.OnMessageUpdated(func(msg *guildedChatMessageUpdated) {
		message := p.getIncomingMessage(&msg.Message)
		if message != nil {
			channel <- lightning.EditedMessage{
				Message: *message,
				Edited:  *msg.Message.UpdatedAt,
			}
		}
	})

	return channel
}

func (p *guildedPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	channel := make(chan lightning.BaseMessage, 1000)

	p.socket.OnMessageDeleted(func(msg *guildedChatMessageDeleted) {
		channel <- lightning.BaseMessage{
			EventID:   msg.Message.ID,
			ChannelID: msg.Message.ChannelID,
			Time:      msg.DeletedAt,
		}
	})

	return channel
}

func (*guildedPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return nil
}
