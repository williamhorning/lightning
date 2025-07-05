package telegram

import (
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func init() {
	lightning.Plugins.RegisterType("telegram", newTelegramPlugin)
}

func newTelegramPlugin(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Invalid config for Telegram plugin", nil, nil)
	}

	token, ok := cfg["token"].(string)
	if !ok || token == "" {
		return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Missing or invalid token in Telegram plugin config", nil, nil)
	}

	proxyPort, ok := cfg["proxy_port"].(int64)
	if !ok || proxyPort < 0 {
		return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Missing or invalid proxy port in Telegram plugin config", nil, nil)
	}

	proxyURL, ok := cfg["proxy_url"].(string)
	if !ok || proxyURL == "" {
		return nil, lightning.LogError(lightning.ErrPluginConfigInvalid, "Missing or invalid proxy URL in Telegram plugin config", nil, nil)
	}

	telegram, err := gotgbot.NewBot(token, nil)
	if err != nil {
		return nil, lightning.LogError(err, "Failed to setup Telegram bot", nil, nil)
	}

	commandChannel := make(chan lightning.CommandEvent, 1000)
	messageChannel := make(chan lightning.Message, 1000)
	editChannel := make(chan lightning.Message, 1000)

	dispatch := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			lightning.LogError(err, "Error in Telegram plugin", nil, nil)
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	dispatch.AddHandler(handlers.Message{
		AllowEdited:   true,
		AllowChannel:  true,
		AllowBusiness: true,
		Filter:        message.All,
		Response: func(b *gotgbot.Bot, ctx *ext.Context) error {
			msg := getMessage(b, ctx, proxyURL)
			if ctx.EditedMessage != nil {
				editChannel <- msg
			} else {
				messageChannel <- msg
			}
			return nil
		},
	})

	updater := ext.NewUpdater(dispatch, nil)
	if err := updater.StartPolling(telegram, &ext.PollingOpts{
		DropPendingUpdates: true,
	}); err != nil {
		return nil, lightning.LogError(err, "Failed to start Telegram updater", nil, nil)
	}

	lightning.Log.With("plugin", "telegram").Info("ready! invite me at https://t.me/" + telegram.Username)

	plugin := &telegramPlugin{commandChannel, messageChannel, editChannel, dispatch, proxyURL, proxyPort, telegram, updater}

	go plugin.startProxy()

	return plugin, nil
}

type telegramPlugin struct {
	commandChannel chan lightning.CommandEvent
	messageChannel chan lightning.Message
	editChannel    chan lightning.Message
	dispatch       *ext.Dispatcher
	proxyURL       string
	proxyPort      int64
	telegram       *gotgbot.Bot
	updater        *ext.Updater
}

func (p *telegramPlugin) Name() string {
	return "bolt-telegram"
}

func (p *telegramPlugin) SetupChannel(channel string) (any, error) {
	return channel, nil
}

func (p *telegramPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return []string{}, lightning.LogError(err, "Failed to parse channel ID", map[string]any{"channel_id": message.ChannelID}, &lightning.ChannelDisabled{Read: false, Write: true})
	}

	content := parseContent(message, opts)

	sendOpts := &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeMarkdownV2,
	}

	if len(message.RepliedTo) > 0 {
		replyID, err := strconv.ParseInt(message.RepliedTo[0], 10, 64)
		if err == nil && replyID > 0 {
			sendOpts.ReplyParameters = &gotgbot.ReplyParameters{
				MessageId:                replyID,
				AllowSendingWithoutReply: true,
			}
		}
	}

	msg, err := p.telegram.SendMessage(channel, content, sendOpts)

	if err != nil {
		return []string{}, lightning.LogError(err, "Failed to send message to Telegram", map[string]any{"channel_id": opts.ChannelID, "content": content, "reply_opts": sendOpts.ReplyParameters}, nil)
	}

	ids := []string{strconv.FormatInt(msg.MessageId, 10)}

	for _, attachment := range message.Attachments {
		if msg, err := p.telegram.SendDocument(channel, gotgbot.InputFileByURL(attachment.URL), nil); err == nil {
			ids = append(ids, strconv.FormatInt(msg.MessageId, 10))
		}
	}

	return ids, nil
}

func (p *telegramPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.SendOptions) error {
	channel, err := strconv.ParseInt(opts.ChannelID, 10, 64)
	if err != nil {
		return lightning.LogError(err, "Failed to parse channel ID", map[string]any{"channel_id": opts.ChannelID}, &lightning.ChannelDisabled{Read: false, Write: true})
	}

	msgID, err := strconv.ParseInt(ids[0], 10, 64)
	if err != nil {
		return lightning.LogError(err, "Failed to parse message ID", map[string]any{"id": ids[0]}, nil)
	}

	content := parseContent(message, opts)
	_, _, err = p.telegram.EditMessageText(content, &gotgbot.EditMessageTextOpts{ChatId: channel, MessageId: msgID, ParseMode: gotgbot.ParseModeMarkdownV2})

	if err != nil && strings.Contains("message is not modified: specified new message content and reply markup are exactly the same as a current content and reply markup of the message", err.Error()) {
		return nil
	}

	return err
}

func (p *telegramPlugin) DeleteMessage(ids []string, opts *lightning.SendOptions) error {
	channel, err := strconv.ParseInt(opts.ChannelID, 10, 64)
	if err != nil {
		return lightning.LogError(err, "Failed to parse channel ID", map[string]any{"channel_id": opts.ChannelID}, &lightning.ChannelDisabled{Read: false, Write: true})
	}

	messageIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		if msgID, err := strconv.ParseInt(id, 10, 64); err == nil {
			messageIDs = append(messageIDs, msgID)
		} else {
			return lightning.LogError(err, "Failed to parse message ID", map[string]any{"id": id}, nil)
		}
	}

	_, err = p.telegram.DeleteMessages(channel, messageIDs, nil)
	return err
}

func (p *telegramPlugin) SetupCommands(commands map[string]lightning.Command) error {
	if help, exists := commands["help"]; exists {
		commands["start"] = help
	}

	cmds := make([]gotgbot.BotCommand, 0, len(commands))

	for telegramName, cmd := range commands {
		cmds = append(cmds, gotgbot.BotCommand{
			Command:     telegramName,
			Description: cmd.Description,
		})

		handler := handlers.NewCommand(telegramName, func(b *gotgbot.Bot, ctx *ext.Context) error {
			p.commandChannel <- getCommand(cmd.Name, b, ctx)
			return nil
		})
		handler.SetAllowChannel(true)
		p.dispatch.AddHandler(handler)
	}

	_, err := p.telegram.SetMyCommands(cmds, nil)
	return err
}

func (p *telegramPlugin) ListenMessages() <-chan lightning.Message {
	return p.messageChannel
}

func (p *telegramPlugin) ListenEdits() <-chan lightning.Message {
	return p.editChannel
}

func (p *telegramPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	return make(chan lightning.BaseMessage)
}

func (p *telegramPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return p.commandChannel
}
