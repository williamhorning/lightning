package guilded

import (
	"github.com/williamhorning/lightning"
)

func init() {
	lightning.RegisterPluginType("guilded", newGuildedPlugin)
}

func newGuildedPlugin(config any) (lightning.Plugin, error) {
	if cfg, ok := config.(map[string]any); !ok {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Invalid config for Guilded plugin",
			nil,
			lightning.ReadWriteDisabled{},
		)
	} else {
		token := cfg["token"].(string)

		socket := guildedNewSocketManager(token)

		if err := socket.Connect(); err != nil {
			return nil, lightning.LogError(
				err,
				"Failed to connect to Guilded socket",
				nil,
				lightning.ReadWriteDisabled{},
			)
		}

		socket.On("ready", func(msg *guildedWelcomeMessage) {
			lightning.Log.Info().Str("plugin", "guilded").Str("username", msg.User.Name).Msg("ready!")
		})

		return &guildedPlugin{token, socket}, nil
	}
}

type guildedPlugin struct {
	token  string
	socket *guildedSocketManager
}

func (p *guildedPlugin) Name() string {
	return "bolt-guilded"
}

func (p *guildedPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.BridgeMessageOptions) error {
	return nil
}

func (p *guildedPlugin) DeleteMessage(ids []string, opts *lightning.BridgeMessageOptions) error {
	for _, id := range ids {
		_, err := guildedMakeRequest(p.token, "DELETE", "/channels/"+opts.Channel.ID+"/messages/"+id, nil)

		if err != nil {
			return lightning.LogError(
				err,
				"Failed to delete message",
				map[string]any{"messageID": id, "channelID": opts.Channel.ID},
				lightning.ReadWriteDisabled{},
			)
		}
	}

	return nil
}

func (p *guildedPlugin) SetupCommands(command []lightning.Command) error {
	return nil
}

func (p *guildedPlugin) ListenMessages() <-chan lightning.Message {
	ch := make(chan lightning.Message)

	p.socket.On("ChatMessageCreated", func(msg *guildedChatMessageCreated) {
		ch <- *getIncomingMessage(p.token, &msg.Message)
	})

	return ch
}

func (p *guildedPlugin) ListenEdits() <-chan lightning.Message {
	ch := make(chan lightning.Message)

	p.socket.On("ChatMessageUpdated", func(msg *guildedChatMessageUpdated) {
		ch <- *getIncomingMessage(p.token, &msg.Message)
	})

	return ch
}

func (p *guildedPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	ch := make(chan lightning.BaseMessage)

	p.socket.On("ChatMessageDeleted", func(msg *guildedChatMessageDeleted) {
		ch <- lightning.BaseMessage{
			EventID:   msg.Message.Id,
			ChannelID: msg.Message.ChannelId,
			Plugin:    "bolt-guilded",
			Time:      msg.DeletedAt,
		}
	})

	return ch
}

func (p *guildedPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return make(chan lightning.CommandEvent)
}
