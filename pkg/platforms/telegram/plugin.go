// Package telegram provides a [lightning.Plugin] implementation for Telegram.
// It additionally provides a file proxy to proxy Telegram attachments to other
// platforms, as Telegram files require a token to fetch, and that shouldn't be
// exposed to other platforms.
//
// To use Telegram support with lightning, see [New]
//
//	bot := lightning.NewBot(lightning.BotOptions{
//		// ...
//	}
//
//	bot.AddPluginType("telegram", telegram.New)
//
//	bot.UsePluginType("telegram", map[string]any{
//		// ...
//	})
package telegram

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// New creates a new [lightning.Plugin] that provides Telegram support for Lightning
//
// It only takes in a map with the following structure:
//
//	map[string]any{
//		"token": "", // a string with your Telegram bot token. You can get this from BotFather
//		"proxy_port": 0, // the port to use for the built-in Telegram file proxy
//		"proxy_url": "", // the publicly accessible url of the Telegram file proxy
//	}
//
// Note that you must have a working file proxy at `proxy_url`, otherwise files will not
// work with other plugins.
func New(config any) (lightning.Plugin, error) {
	cfg, err := getTelegramConfig(config)
	if err != nil {
		return nil, err
	}

	telegram, err := gotgbot.NewBot(cfg.token, &gotgbot.BotOpts{BotClient: newRetrier()})
	if err != nil {
		return nil, lightning.LogError(err, "Failed to setup Telegram bot", nil, nil)
	}

	commands := make(chan lightning.CommandEvent, 1000)
	messages := make(chan lightning.Message, 1000)
	edits := make(chan lightning.EditedMessage, 1000)

	dispatch := ext.NewDispatcher(nil)

	dispatch.AddHandler(handlers.Message{
		AllowEdited:   true,
		AllowChannel:  true,
		AllowBusiness: true,
		Filter: func(_ *gotgbot.Message) bool {
			return true
		},
		Response: func(b *gotgbot.Bot, ctx *ext.Context) error {
			msg := getMessage(b, ctx, cfg.proxyURL)
			if ctx.EditedMessage != nil {
				edits <- lightning.EditedMessage{
					Message: msg,
					Edited:  time.UnixMilli(ctx.EditedMessage.GetDate() * 1000),
				}
			} else {
				messages <- msg
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

	slog.Info("telegram: ready! invite me at https://t.me/" + telegram.Username)

	plugin := &telegramPlugin{commands, messages, edits, dispatch, &cfg, telegram, updater}

	go plugin.startProxy()

	return plugin, nil
}

type telegramPlugin struct {
	commandChannel chan lightning.CommandEvent
	messageChannel chan lightning.Message
	editChannel    chan lightning.EditedMessage
	dispatch       *ext.Dispatcher
	cfg            *telegramConfig
	telegram       *gotgbot.Bot
	updater        *ext.Updater
}

func (*telegramPlugin) SetupChannel(_ string) (any, error) {
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *telegramPlugin) SendMessage(message lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return []string{}, lightning.LogError(err, "Failed to parse channel ID",
			map[string]any{"channel_id": message.ChannelID}, &lightning.ChannelDisabled{Read: false, Write: true})
	}

	content := parseContent(message, opts)

	sendOpts := &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeMarkdownV2,
		RequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Second * 10,
		},
	}

	if len(message.RepliedTo) > 0 {
		var replyID int64

		replyID, err = strconv.ParseInt(message.RepliedTo[0], 10, 64)
		if err == nil && replyID > 0 {
			sendOpts.ReplyParameters = &gotgbot.ReplyParameters{
				MessageId:                replyID,
				AllowSendingWithoutReply: true,
			}
		}
	}

	msg, err := p.telegram.SendMessage(channel, content, sendOpts)
	if err != nil {
		return []string{}, lightning.LogError(err, "failed to send telegram message",
			map[string]any{"channel": message.ChannelID, "content": content, "reply": sendOpts.ReplyParameters}, nil)
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
	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return lightning.LogError(err, "Failed to parse channel ID", map[string]any{"channel_id": message.ChannelID},
			&lightning.ChannelDisabled{Read: false, Write: true})
	}

	msgID, err := strconv.ParseInt(ids[0], 10, 64)
	if err != nil {
		return lightning.LogError(err, "Failed to parse message ID", map[string]any{"id": ids[0]}, nil)
	}

	content := parseContent(message, opts)

	_, _, err = p.telegram.EditMessageText(
		content,
		&gotgbot.EditMessageTextOpts{ChatId: channel, MessageId: msgID, ParseMode: gotgbot.ParseModeMarkdownV2},
	)
	if err != nil &&
		strings.Contains(err.Error(), "specified new message content and reply markup are exactly the same") {
		return nil
	}

	if err == nil {
		return nil
	}

	return lightning.LogError(err, "Failed to edit message on Telegram",
		map[string]any{"channel_id": message.ChannelID, "content": content}, nil)
}

func (p *telegramPlugin) DeleteMessage(channelID string, ids []string) error {
	channel, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return lightning.LogError(err, "Failed to parse channel ID", map[string]any{"channel_id": channelID},
			&lightning.ChannelDisabled{Read: false, Write: true})
	}

	messageIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		var msgID int64

		msgID, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return lightning.LogError(err, "Failed to parse message ID", map[string]any{"id": id}, nil)
		}

		messageIDs = append(messageIDs, msgID)
	}

	_, err = p.telegram.DeleteMessages(channel, messageIDs, nil)
	if err == nil {
		return nil
	}

	return lightning.LogError(err, "Failed to delete message on Telegram", map[string]any{"channel_id": channelID}, nil)
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

	return lightning.LogError(err, "Failed to setup commands on Telegram", nil, nil)
}

func (p *telegramPlugin) ListenMessages() <-chan lightning.Message {
	return p.messageChannel
}

func (p *telegramPlugin) ListenEdits() <-chan lightning.EditedMessage {
	return p.editChannel
}

func (*telegramPlugin) ListenDeletes() <-chan lightning.BaseMessage {
	return nil
}

func (p *telegramPlugin) ListenCommands() <-chan lightning.CommandEvent {
	return p.commandChannel
}
