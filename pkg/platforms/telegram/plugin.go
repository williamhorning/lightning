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

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
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
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	messages := make(chan *lightning.Message, 2048)
	edits := make(chan *lightning.EditedMessage, 2048)

	dispatch := ext.NewDispatcher(nil)

	dispatch.AddHandler(handlers.Message{
		AllowEdited:   true,
		AllowChannel:  true,
		AllowBusiness: true,
		Filter: func(_ *gotgbot.Message) bool {
			return true
		},
		Response: func(b *gotgbot.Bot, ctx *ext.Context) error {
			msg := telegramToLightningMessage(b, ctx, config["proxy_url"])
			if ctx.EditedMessage != nil {
				edits <- &lightning.EditedMessage{
					Message: &msg, Edited: time.UnixMilli(ctx.EditedMessage.GetDate() * 1000),
				}
			} else {
				messages <- &msg
			}

			return nil
		},
	})

	updater := ext.NewUpdater(dispatch, &ext.UpdaterOpts{
		UnhandledErrFunc: func(err error) {
			if err != nil && !errors.Is(err, context.DeadlineExceeded) &&
				!strings.Contains(err.Error(), "connection reset") {
				log.Printf("telegram: unhandled error in dispatcher: %v\n", err)
			}
		},
	})
	if err := updater.StartPolling(telegram, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts:     &gotgbot.GetUpdatesOpts{Timeout: int64(defaultTimeout.Seconds())},
	}); err != nil {
		return nil, fmt.Errorf("failed to start polling: %w", err)
	}

	log.Println("telegram: ready as @" + telegram.Username + "! invite me at https://t.me/" + telegram.Username)

	plugin := &telegramPlugin{messages, edits, dispatch, telegram, updater}

	if err := startProxy(config); err != nil {
		return nil, err
	}

	return plugin, nil
}

type telegramPlugin struct {
	messageChannel chan *lightning.Message
	editChannel    chan *lightning.EditedMessage
	dispatch       *ext.Dispatcher
	telegram       *gotgbot.Bot
	updater        *ext.Updater
}

func (p *telegramPlugin) IsAdmin(user, channel string) (bool, error) {
	chID, err := strconv.ParseInt(channel, 10, 64)
	if err != nil {
		return false, &channelIDError{channel}
	}

	userID, err := strconv.ParseInt(user, 10, 64)
	if err != nil {
		return false, &channelIDError{channel}
	}

	member, err := p.telegram.GetChatMember(chID, userID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}

	switch member.GetStatus() {
	case "creator", "administrator":
		return true, nil
	default:
		return false, nil
	}
}

func (*telegramPlugin) SetupChannel(_ string) (map[string]string, error) {
	return nil, nil //nolint:nilnil
}

func (p *telegramPlugin) SendMessage(message *lightning.Message, opts *lightning.SendOptions) ([]string, error) {
	if opts.CommandResponse {
		message.ChannelID = opts.CommandUser
	}

	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return nil, &channelIDError{message.ChannelID}
	}

	chunks := lightningToTelegramMessage(message)

	res := make([]string, 0, len(message.Attachments)+len(chunks))

	for _, data := range chunks {
		msg, err := p.telegram.SendMessage(channel, data.content, getSendOptions(message, data.entities))
		if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
			return res, fmt.Errorf("failed to send message: %w", err)
		}

		res = append(res, strconv.FormatInt(msg.MessageId, 10))
	}

	for _, attachment := range message.Attachments {
		msg, err := p.telegram.SendDocument(channel, gotgbot.InputFileByURL(attachment.URL), nil)
		if err != nil {
			log.Printf("telegram: failed to send attachment: %v\n", err)
		} else {
			res = append(res, strconv.FormatInt(msg.MessageId, 10))
		}
	}

	return res, nil
}

func (p *telegramPlugin) EditMessage(
	message *lightning.Message, ids []string, opts *lightning.SendOptions,
) ([]string, error) {
	if opts.CommandResponse {
		message.ChannelID = opts.CommandUser
	}

	channel, err := strconv.ParseInt(message.ChannelID, 10, 64)
	if err != nil {
		return nil, &channelIDError{message.ChannelID}
	}

	for idx, data := range lightningToTelegramMessage(message) {
		telegramID, err := strconv.ParseInt(ids[idx], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message ID on edit: %w", err)
		}

		_, _, err = p.telegram.EditMessageText(
			data.content,
			&gotgbot.EditMessageTextOpts{
				ChatId:    channel,
				MessageId: telegramID,
				Entities:  data.entities,
			},
		)
		if err != nil && !strings.Contains(err.Error(),
			"specified new message content and reply markup are exactly the same") {
			return nil, fmt.Errorf("failed to edit message: %w", err)
		}
	}

	return ids, nil
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
			return fmt.Errorf("failed to parse message ID on delete: %w", err)
		}

		messageIDs = append(messageIDs, msgID)
	}

	_, err = p.telegram.DeleteMessages(channel, messageIDs, nil)
	if err == nil {
		return nil
	}

	return fmt.Errorf("failed to delete message: %w", err)
}

func (*telegramPlugin) SetupCommands(_ map[string]lightning.Command) {}

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
