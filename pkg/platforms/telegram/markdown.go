package telegram

import (
	"strings"
	"unicode/utf16"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type textBuilder struct {
	out      strings.Builder
	offset   int64
	entities []gotgbot.MessageEntity
	src      []byte
}

func (tb *textBuilder) write(s string) {
	for _, r := range s {
		tb.out.WriteRune(r)
		tb.offset += int64(utf16.RuneLen(r))
	}
}

func (tb *textBuilder) walk(node ast.Node) {
	start := tb.offset

	typ, url, lang := tb.handleNode(node)

	if _, ok := node.(*ast.Text); ok {
		return
	}

	if _, isText := node.(*ast.Text); !isText {
		if _, isFenced := node.(*ast.FencedCodeBlock); !isFenced {
			for c := node.FirstChild(); c != nil; c = c.NextSibling() {
				tb.walk(c)
			}
		}
	}

	if typ != "" && tb.offset > start {
		tb.entities = append(tb.entities, gotgbot.MessageEntity{
			Type:     typ,
			Offset:   start,
			Length:   tb.offset - start,
			Url:      url,
			Language: lang,
		})
	}

	if _, ok := node.(*ast.Paragraph); ok {
		tb.write("\n")
	}
}

func (tb *textBuilder) handleNode(n ast.Node) (string, string, string) { //nolint:cyclop,revive
	switch nod := n.(type) {
	case *east.Strikethrough:
		return "strikethrough", "", ""
	case *ast.Text:
		tb.write(string(nod.Segment.Value(tb.src)))

		if nod.SoftLineBreak() || nod.HardLineBreak() {
			tb.write("\n")
		}

		return "", "", ""
	case *ast.Emphasis:
		if nod.Level == 2 {
			return "bold", "", ""
		}

		return "italic", "", ""
	case *ast.CodeSpan:
		return "code", "", ""
	case *ast.FencedCodeBlock:
		tb.write(string(nod.Lines().Value(tb.src)))

		return "pre", "", string(nod.Language(tb.src))
	case *ast.Link:
		return "text_link", string(nod.Destination), ""
	default:
		return "", "", ""
	}
}

func markdownToTelegram(md string) (string, []gotgbot.MessageEntity) {
	parser := goldmark.New(goldmark.WithExtensions(extension.Strikethrough)).Parser()
	src := []byte(md)
	root := parser.Parse(text.NewReader(src))

	tb := &textBuilder{src: src}
	tb.walk(root)

	return strings.TrimSuffix(tb.out.String(), "\n"), tb.entities
}
