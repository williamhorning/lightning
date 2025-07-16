package telegram

import (
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func getMarkdownV2(input string) string {
	md := goldmark.New()
	reader := text.NewReader([]byte(input))
	result := nodeToTelegram(md.Parser().Parse(reader), reader.Source())

	return strings.TrimSpace(result)
}

func escapeTelegramText(input string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, ch := range specialChars {
		input = strings.ReplaceAll(input, ch, "\\"+ch)
	}

	return input
}

//nolint:cyclop // how on earth would i simplify this more
func nodeToTelegram(astNode ast.Node, source []byte) string {
	switch node := astNode.(type) {
	case *ast.Text:
		return handleTextNode(node, source)
	case *ast.Emphasis:
		return handleEmphasisNode(node, source)
	case *ast.Link:
		return handleLinkNode(node, source)
	case *ast.Paragraph:
		return handleParagraphNode(node, source)
	case *ast.CodeSpan:
		return handleCodeSpanNode(node, source)
	case *ast.Blockquote:
		return handleBlockquoteNode(node, source)
	case *ast.FencedCodeBlock:
		return handleFencedCodeBlockNode(node, source)
	case *ast.List:
		return handleListNode(node, source)
	case *ast.ListItem:
		return handleListItemNode(node, source)
	default:
		return handleOtherNode(node, source)
	}
}

func handleTextNode(node *ast.Text, source []byte) string {
	return escapeTelegramText(string(node.Segment.Value(source)))
}

func handleEmphasisNode(node *ast.Emphasis, source []byte) string {
	res := ""

	var emphasisChar string

	if node.Level == 1 {
		emphasisChar = "_"
	} else {
		emphasisChar = "*"
	}

	res += emphasisChar

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		res += nodeToTelegram(child, source)
	}

	return res + emphasisChar
}

func handleLinkNode(node *ast.Link, source []byte) string {
	var textContent string

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		textContent += nodeToTelegram(child, source)
	}

	return "[" + textContent + "](" + escapeTelegramText(string(node.Destination)) + ")"
}

func handleParagraphNode(node *ast.Paragraph, source []byte) string {
	res := ""

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		res += nodeToTelegram(child, source)
	}

	return res + "\n\n"
}

func handleCodeSpanNode(node *ast.CodeSpan, source []byte) string {
	var content []byte

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*ast.Text); ok {
			segment := textNode.Segment
			content = append(content, segment.Value(source)...)
		}
	}

	return "`" + escapeTelegramText(string(content)) + "`"
}

func handleBlockquoteNode(node *ast.Blockquote, source []byte) string {
	res := ">"

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		res += nodeToTelegram(child, source)
	}

	return res + "\n"
}

func handleFencedCodeBlockNode(node *ast.FencedCodeBlock, source []byte) string {
	res := "```"

	if len(node.Language(source)) > 0 {
		res += string(node.Language(source))
	}

	res += "\n"
	res += escapeTelegramText(string(node.Lines().Value(source)))

	return res + "\n```\n"
}

func handleListNode(node *ast.List, source []byte) string {
	res := ""

	for index, child := 0, node.FirstChild(); child != nil; child = child.NextSibling() {
		index++

		prefix := "\\- "

		if node.IsOrdered() {
			prefix = strconv.Itoa(index) + "\\. "
		}

		res += prefix
		res += nodeToTelegram(child, source)
		res += "\n"
	}

	return res
}

func handleListItemNode(node *ast.ListItem, source []byte) string {
	res := ""

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		res += nodeToTelegram(child, source)
	}

	return res
}

func handleOtherNode(node ast.Node, source []byte) string {
	res := ""

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		res += nodeToTelegram(child, source)
	}

	return res
}
