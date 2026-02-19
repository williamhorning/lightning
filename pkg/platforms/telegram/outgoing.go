package telegram

import (
	"strconv"
	"strings"

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

type entityContentPair struct {
	content  string
	entities []gotgbot.MessageEntity
}

func lightningToTelegramMessage(message *lightning.Message) []entityContentPair {
	var content strings.Builder

	if message.Author != nil {
		content.WriteString(message.Author.Username + " » ")
	}

	content.WriteString(message.Content + "\n")

	for idx := range message.Embeds {
		content.WriteString("\n\n" + message.Embeds[idx].ToMarkdown())
	}

	return markdownToTelegram(content.String())
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
