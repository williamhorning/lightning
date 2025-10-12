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
//	bot.UsePluginType("telegram", "", map[string]string{
//		// ...
//	})
package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
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
//	map[string]string{
//		"token": "", // a string with your Telegram bot token. You can get this from BotFather
//		"proxy_port": "0", // the port to use for the built-in Telegram file proxy
//		"proxy_url": "", // the publicly accessible url of the Telegram file proxy
//	}
//
// Note that you must have a working file proxy at `proxy_url`, otherwise files will not
// work with other plugins.
func New(config map[string]string) (lightning.Plugin, error) {
	telegram, err := gotgbot.NewBot(config["token"], &gotgbot.BotOpts{BotClient: newRetrier()})
	if err != nil {
		return nil, fmt.Errorf("telegram: failed to create bot: %w", err)
	}

	messages := make(chan *lightning.Message, 1000)
	edits := make(chan *lightning.EditedMessage, 1000)

	dispatch := ext.NewDispatcher(nil)

	dispatch.AddHandler(handlers.Message{
		AllowEdited:   true,
		AllowChannel:  true,
		AllowBusiness: true,
		Filter: func(_ *gotgbot.Message) bool {
			return true
		},
		Response: func(b *gotgbot.Bot, ctx *ext.Context) error {
			msg := getMessage(b, ctx, config["proxy_url"])
			if ctx.EditedMessage != nil {
				time := time.UnixMilli(ctx.EditedMessage.GetDate() * 1000)
				edits <- &lightning.EditedMessage{
					Message: &msg,
					Edited:  &time,
				}
			} else {
				messages <- &msg
			}

			return nil
		},
	})

	updater := ext.NewUpdater(dispatch, &ext.UpdaterOpts{
		UnhandledErrFunc: func(err error) {
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				log.Printf("telegram: unhandled error in dispatcher: %v\n", err)
			}
		},
	})
	if err := updater.StartPolling(telegram, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts:     &gotgbot.GetUpdatesOpts{Timeout: int64(defaultTimeout.Seconds())},
	}); err != nil {
		return nil, fmt.Errorf("telegram: failed to start polling: %w", err)
	}

	log.Printf("telegram: ready! invite me at https://t.me/%s\n", telegram.Username)

	plugin := &telegramPlugin{messages, edits, dispatch, telegram, updater}

	go startProxy(config)

	return plugin, nil
}

type telegramPlugin struct {
	messageChannel chan *lightning.Message
	editChannel    chan *lightning.EditedMessage
	dispatch       *ext.Dispatcher
	telegram       *gotgbot.Bot
	updater        *ext.Updater
}

func (*telegramPlugin) SetupChannel(_ string) (any, error) {
	return nil, nil //nolint:nilnil // we don't need a value for ChannelData later
}

func (p *telegramPlugin) SendCommandResponse(
	message *lightning.Message,
	opts *lightning.SendOptions,
	user string,
) ([]string, error) {
	message.ChannelID = user

	return p.SendMessage(message, opts)
}

func (p *telegramPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return nil, &channelIDError{message.ChannelID}
	}

	content := parseContent(message, opts)

	sendOpts := &gotgbot.SendMessageOpts{
		ParseMode: gotgbot.ParseModeMarkdownV2, RequestOpts: &gotgbot.RequestOpts{Timeout: defaultTimeout},
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
	if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
		return []string{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("telegram: failed to send message: %w\n\tchannel: %s\n\tcontent: %s\n\treply: %#+v",
			err, message.ChannelID, content, sendOpts.ReplyParameters)
	}

	ids := []string{strconv.FormatInt(msg.MessageId, 10)}

	for _, attachment := range message.Attachments {
		if msg, err := p.telegram.SendDocument(channel, gotgbot.InputFileByURL(attachment.URL), nil); err == nil {
			ids = append(ids, strconv.FormatInt(msg.MessageId, 10))
		}
	}

	return ids, nil
}

func (p *telegramPlugin) EditMessage(message *lightning.Message, ids []string, opts *lightning.SendOptions) error {
	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return &channelIDError{message.ChannelID}
	}

	msgID, err := strconv.ParseInt(ids[0], 10, 64)
	if err != nil {
		return fmt.Errorf("telegram: failed to parse message ID: %w\n\tchannel: %s\n\tmessage: %s",
			err, message.ChannelID, ids[0])
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

	return fmt.Errorf("telegram: failed to edit message: %w\n\tchannel: %s\n\tmessage: %s",
		err, message.ChannelID, ids[0])
}

func (p *telegramPlugin) DeleteMessage(channelID string, ids []string) error {
	channel, err := strconv.ParseInt(channelID, 10, 64)
	if err != nil {
		return &channelIDError{channelID}
	}

	messageIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		var msgID int64

		msgID, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return fmt.Errorf("telegram: failed to parse message ID: %w\n\tchannel: %s\n\tmessage: %d",
				err, channelID, msgID)
		}

		messageIDs = append(messageIDs, msgID)
	}

	_, err = p.telegram.DeleteMessages(channel, messageIDs, nil)
	if err == nil {
		return nil
	}

	return fmt.Errorf("telegram: failed to delete message: %w\n\tchannel: %s\n\tmessage: %#+v", err, channelID, ids)
}

func (*telegramPlugin) SetupCommands(_ map[string]*lightning.Command) error {
	return nil
}

func (p *telegramPlugin) ListenMessages() <-chan *lightning.Message {
	return p.messageChannel
}

func (p *telegramPlugin) ListenEdits() <-chan *lightning.EditedMessage {
	return p.editChannel
}

func (*telegramPlugin) ListenDeletes() <-chan *lightning.BaseMessage {
	return nil
}

func (*telegramPlugin) ListenCommands() <-chan *lightning.CommandEvent {
	return nil
}
