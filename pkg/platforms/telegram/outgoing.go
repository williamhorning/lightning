package telegram

import (
	"math"
	"slices"

	"github.com/sshturbo/GoTeleMD/pkg/formatter"
	"github.com/sshturbo/GoTeleMD/pkg/types"
	"github.com/williamhorning/lightning/pkg/lightning"
)

var specialChars = []string{"[", "]", "(", ")", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!", "\\."}

func parseContent(message lightning.Message, opts *lightning.SendOptions) string {
	content := ""

	if opts != nil {
		content += getMarkdownV2(message.Author.Nickname) + " » "
	}

	mdV2 := getMarkdownV2(message.Content)

	if len(mdV2) > 0 && slices.Contains(specialChars, mdV2[:1]) {
		content += "\n"
	}

	content += mdV2

	if len(message.Embeds) > 0 {
		content += "\n_this message has embeds_"
	}

	return content
}

var config = &types.Config{
	SafetyLevel:          1,
	AlignTableColumns:    true,
	IgnoreTableSeparator: false,
	MaxMessageLength:     math.MaxInt,
	EnableDebugLogs:      false,
	PreserveEmptyLines:   false,
	StrictLineBreaks:     true,
	NumWorkers:           4,
	WorkerQueueSize:      32,
	MaxConcurrentParts:   8,
}

func getMarkdownV2(str string) string {
	resp, _ := formatter.ConvertMarkdown(str, config)

	return resp
}
