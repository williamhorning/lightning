package telegram

import (
	"slices"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type channelIDError struct {
	channelID string
}

// Disable implements the lightning.ChannelDisabler interface for channelIDError.
func (channelIDError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (e channelIDError) Error() string {
	return "telegram: invalid channel ID: " + e.channelID
}

func parseContent(message lightning.Message, opts *lightning.SendOptions) string {
	content := ""

	if opts != nil {
		content += getMarkdownV2(message.Author.Nickname) + " » "
	}

	mdV2 := getMarkdownV2(message.Content)

	if len(mdV2) > 0 &&
		slices.Contains(
			[]string{"[", "]", "(", ")", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!", "\\."}, mdV2[:1],
		) {
		content += "\n"
	}

	content += mdV2

	for _, embed := range message.Embeds {
		content += getMarkdownV2(embed.ToMarkdown())
	}

	return content
}
