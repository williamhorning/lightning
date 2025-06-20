package telegram

import (
	"strconv"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/williamhorning/lightning"
)

func init() {
	lightning.RegisterPluginType("bolt-telegram", newTelegramPlugin)
}

func newTelegramPlugin(config any) (lightning.Plugin, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Invalid config for Telegram plugin",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	token, ok := cfg["token"].(string)
	if !ok || token == "" {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Missing or invalid token in Telegram plugin config",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	proxyPort, ok := cfg["proxy_port"].(int64)
	if !ok || proxyPort < 0 {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Missing or invalid proxy port in Telegram plugin config",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	proxyURL, ok := cfg["proxy_url"].(string)
	if !ok || proxyURL == "" {
		return nil, lightning.LogError(
			lightning.ErrPluginConfigInvalid,
			"Missing or invalid proxy URL in Telegram plugin config",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	telegram, err := gotgbot.NewBot(token, nil)
	if err != nil {
		return nil, lightning.LogError(
			err,
			"Failed to setup Telegram bot",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	commandChannel := make(chan lightning.CommandEvent, 100)
	messageChannel := make(chan lightning.Message, 100)
	editChannel := make(chan lightning.Message, 100)

	dispatch := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			lightning.LogError(err, "Error in Telegram plugin", nil, lightning.ReadWriteDisabled{})
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
		return nil, lightning.LogError(
			err,
			"Failed to start Telegram updater",
			nil,
			lightning.ReadWriteDisabled{},
		)
	}

	lightning.Log.Info().Str("plugin", "telegram").Str("username", telegram.Username).Msg("ready!")
	lightning.Log.Info().Str("plugin", "telegram").Msg("invite me at https://t.me/" + telegram.Username)

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

func (p *telegramPlugin) SendMessage(message lightning.Message, opts *lightning.BridgeMessageOptions) ([]string, error) {
	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return []string{}, lightning.LogError(
			err,
			"Failed to parse channel ID",
			map[string]any{"channel_id": message.ChannelID},
			lightning.ReadWriteDisabled{Read: false, Write: true},
		)
	}

	content := parseContent(message, opts)

	// Setup message options with proper reply handling
	sendOpts := &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeMarkdownV2,
	}

	if len(message.RepliedTo) > 0 {
		replyID, err := strconv.ParseInt(message.RepliedTo[0], 10, 64)
		if err == nil {
			sendOpts.ReplyParameters = &gotgbot.ReplyParameters{
				MessageId:                replyID,
				AllowSendingWithoutReply: true,
			}
		}
	}

	msg, err := p.telegram.SendMessage(channel, content, sendOpts)

	if err != nil {
		return []string{}, lightning.LogError(
			err,
			"Failed to send message to Telegram",
			map[string]any{"channel_id": opts.Channel.ID, "content": content},
			lightning.ReadWriteDisabled{},
		)
	}

	ids := []string{strconv.FormatInt(msg.MessageId, 10)}

	docOpts := &gotgbot.SendDocumentOpts{}
	if len(message.RepliedTo) > 0 {
		replyID, err := strconv.ParseInt(message.RepliedTo[0], 10, 64)
		if err == nil {
			docOpts.ReplyParameters = &gotgbot.ReplyParameters{
				MessageId:                replyID,
				AllowSendingWithoutReply: true,
			}
		}
	}

	for _, attachment := range message.Attachments {
		if msg, err := p.telegram.SendDocument(channel, gotgbot.InputFileByURL(attachment.URL), docOpts); err == nil {
			ids = append(ids, strconv.FormatInt(msg.MessageId, 10))
		}
	}

	return ids, nil
}

func (p *telegramPlugin) EditMessage(message lightning.Message, ids []string, opts *lightning.BridgeMessageOptions) error {
	channel, err := strconv.ParseInt(opts.Channel.ID, 10, 64)
	if err != nil {
		return lightning.LogError(
			err,
			"Failed to parse channel ID",
			map[string]any{"channel_id": opts.Channel.ID},
			lightning.ReadWriteDisabled{Read: false, Write: true},
		)
	}

	msgID, err := strconv.ParseInt(ids[0], 10, 64)
	if err != nil {
		return lightning.LogError(
			err,
			"Failed to parse message ID",
			map[string]any{"id": ids[0]},
			lightning.ReadWriteDisabled{},
		)
	}

	content := parseContent(message, opts)
	_, _, err = p.telegram.EditMessageText(content, &gotgbot.EditMessageTextOpts{
		ChatId:    channel,
		MessageId: msgID,
		ParseMode: gotgbot.ParseModeMarkdownV2,
	})

	return err
}

func (p *telegramPlugin) DeleteMessage(ids []string, opts *lightning.BridgeMessageOptions) error {
	channel, err := strconv.ParseInt(opts.Channel.ID, 10, 64)
	if err != nil {
		return lightning.LogError(
			err,
			"Failed to parse channel ID",
			map[string]any{"channel_id": opts.Channel.ID},
			lightning.ReadWriteDisabled{Read: false, Write: true},
		)
	}

	messageIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		if msgID, err := strconv.ParseInt(id, 10, 64); err == nil {
			messageIDs = append(messageIDs, msgID)
		} else {
			return lightning.LogError(
				err,
				"Failed to parse message ID",
				map[string]any{"id": id},
				lightning.ReadWriteDisabled{},
			)
		}
	}

	_, err = p.telegram.DeleteMessages(channel, messageIDs, nil)
	return err
}

func (p *telegramPlugin) SetupCommands(commands []lightning.Command) error {
	start := lightning.HelpCommand()
	start.Name = "start"
	commands = append(commands, start)

	cmds := make([]gotgbot.BotCommand, 0, len(commands))

	for _, cmd := range commands {
		cmds = append(cmds, gotgbot.BotCommand{
			Command:     cmd.Name,
			Description: cmd.Description,
		})

		handler := handlers.NewCommand(cmd.Name, func(b *gotgbot.Bot, ctx *ext.Context) error {
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
