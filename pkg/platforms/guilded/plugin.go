package guilded

import (
	"github.com/williamhorning/lightning/pkg/lightning"
)

func init() {
	lightning.Plugins.RegisterType("guilded", newGuildedPlugin)
}

func newGuildedPlugin(config any) (lightning.Plugin, error) {
	if cfg, ok := config.(map[string]any); !ok {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Invalid config for Guilded plugin",
			nil,
			lightning.ChannelDisabled{},
		)
	} else {
		token := cfg["token"].(string)

		socket := guildedNewSocketManager(token)
		plugin := &guildedPlugin{token, socket}

		socket.OnReady(func(msg *guildedWelcomeMessage) {
			lightning.Log.Info().Str("plugin", "guilded").Str("username", msg.User.Name).Msg("ready!")
		})

		if err := socket.Connect(); err != nil {
			return nil, lightning.LogError(
				err,
				"Failed to connect to Guilded socket",
				nil,
				lightning.ChannelDisabled{},
			)
		}

		return plugin, nil
	}
}

type guildedPlugin struct {
	token  string
	socket *guildedSocketManager
}

func (p *guildedPlugin) Name() string {
	return "bolt-guilded"
}

func (p *guildedPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.SendOptions) error {
	return nil
}

func (p *guildedPlugin) DeleteMessage(ids []string, opts *lightning.SendOptions) error {
	for _, id := range ids {
		_, err := guildedMakeRequest(p.token, "DELETE", "/channels/"+opts.ChannelID+"/messages/"+id, nil)

		if err != nil {
			return lightning.LogError(
				err,
				"Failed to delete message",
				map[string]any{"messageID": id, "channelID": opts.ChannelID},
				lightning.ChannelDisabled{},
			)
		}
	}

	return nil
}

func (p *guildedPlugin) SetupCommands(command map[string]lightning.Command) error {
	return nil
}

func (p *guildedPlugin) ListenMessages() <-chan lightning.Message {
	ch := make(chan lightning.Message, 1000)

	p.socket.OnMessageCreated(func(msg *guildedChatMessageCreated) {
		message := getIncomingMessage(p.token, &msg.Message)
		if message != nil {
			ch <- *message
		}
	})

	return ch
}

func (p *guildedPlugin) ListenEdits() <-chan lightning.Message {
	ch := make(chan lightning.Message, 1000)

	p.socket.OnMessageUpdated(func(msg *guildedChatMessageUpdated) {
		message := getIncomingMessage(p.token, &msg.Message)
		if message != nil {
			ch <- *message
		}
	})

	return ch
}

func (p *guildedPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	ch := make(chan lightning.BaseMessage, 1000)

	p.socket.OnMessageDeleted(func(msg *guildedChatMessageDeleted) {
		ch <- lightning.BaseMessage{
			EventID:   msg.Message.ID,
			ChannelID: msg.Message.ChannelID,
			Plugin:    "bolt-guilded",
			Time:      msg.DeletedAt,
		}
	})

	return ch
}

func (p *guildedPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return make(chan lightning.CommandEvent)
}
