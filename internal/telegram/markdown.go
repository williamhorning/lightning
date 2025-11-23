// Package telegram handles Telegram-specific transformations
package telegram

import (
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// GetMarkdownV2 takes in CommonMark-like syntax and turns it into Telegram's MarkdownV2.
func GetMarkdownV2(input string) string {
	reader := text.NewReader([]byte(input))
	root := goldmark.DefaultParser().Parse(reader)

	return strings.TrimSpace(nodeToTelegram(root, reader.Source()))
}

func nodeToTelegram(astNode ast.Node, source []byte) string {
	switch node := astNode.(type) {
	case *ast.Text:
		return escapeTelegramText(string(node.Segment.Value(source)))
	case *ast.Emphasis:
		return getEmphasis(node) + handleNode(node, source) + getEmphasis(node)
	case *ast.Link:
		return "[" + handleNode(node, source) + "](" + escapeTelegramText(string(node.Destination)) + ")"
	case *ast.Paragraph:
		return handleNode(node, source) + "\n\n"
	case *ast.CodeSpan:
		return "`" + handleNode(node, source) + "`"
	case *ast.Blockquote:
		return ">" + handleNode(node, source) + "\n"
	case *ast.FencedCodeBlock:
		return "```" + string(node.Language(source)) + "\n" + escapeTelegramText(string(node.Lines().Value(source))) +
			"\n```\n"
	default:
		return handleNode(node, source)
	}
}

func escapeTelegramText(input string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, ch := range specialChars {
		input = strings.ReplaceAll(input, ch, "\\"+ch)
	}

	return input
}

func getEmphasis(node *ast.Emphasis) string {
	if node.Level == 1 {
		return "_"
	}

	return "*"
}

func handleNode(node ast.Node, source []byte) string {
	res := ""

	for i, child := 1, node.FirstChild(); child != nil; child, i = child.NextSibling(), i+1 {
		if list, ok := node.(*ast.List); ok {
			prefix := "\\- "

			if list.IsOrdered() {
				prefix = strconv.FormatInt(int64(i), 10) + "\\. "
			}

			res += prefix + nodeToTelegram(child, source) + "\n"
		} else {
			res += nodeToTelegram(child, source)
		}
	}

	return res
}
