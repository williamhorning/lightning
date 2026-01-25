package telegram

import (
	"strconv"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/PaulSonOfLars/gotgbot/v2"
)

type channelIDError struct {
	channelID string
}

func (channelIDError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e channelIDError) Error() string {
	return "invalid channel ID: " + e.channelID
}

type notAdminError struct{}

func (notAdminError) Error() string {
	return "to setup a channel, you must be an administrator or owner"
}

type entityContentPair struct {
	content  string
	entities []gotgbot.MessageEntity
}

func lightningToTelegramMessage(message *lightning.Message) []entityContentPair {
	content := ""

	if message.Author != nil {
		content += message.Author.Username + " » "
	}

	content += message.Content + "\n"

	for idx := range message.Embeds {
		content += "\n\n" + message.Embeds[idx].ToMarkdown()
	}

	return markdownToTelegram(content)
}

func getSendOptions(message *lightning.Message, entities []gotgbot.MessageEntity) *gotgbot.SendMessageOpts {
	sendOpts := &gotgbot.SendMessageOpts{Entities: entities}

	if len(message.RepliedTo) != 0 {
		replyID, err := strconv.ParseInt(message.RepliedTo[0], 10, 64)
		if err == nil && replyID > 0 {
			sendOpts.ReplyParameters = &gotgbot.ReplyParameters{
				MessageId:                replyID,
				AllowSendingWithoutReply: true,
			}
		}
	}

	return sendOpts
}
