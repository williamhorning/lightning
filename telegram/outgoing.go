package telegram

import (
	"regexp"
	"slices"
	"strings"

	"github.com/williamhorning/lightning"
)

var telegramSpecialCharacters = []string{
	"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!",
}

var headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

func parseContent(message lightning.Message, opts *lightning.BridgeMessageOptions) string {
	bridged := opts != nil

	content := ""

	if bridged {
		content += escapeMarkdownV2(message.Author.Nickname) + " » "
	}

	content += message.Content

	if len(message.Content) == 0 {
		content += "_no content_"
	}

	if len(message.Embeds) > 0 {
		content += "\n_this message has embeds_"
	}

	return telegramifyMarkdown(content)
}

func telegramifyMarkdown(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if headingMatches := headingPattern.FindStringSubmatch(trimmed); len(headingMatches) == 3 {
			headingText := headingMatches[2]
			lines[i] = "*" + escapeMarkdownV2(headingText) + "*"
		} else {
			lines[i] = processInlineMarkdown(line)
		}
	}

	return strings.Join(lines, "\n")
}

func escapeMarkdownV2(text string) string {
	for _, char := range telegramSpecialCharacters {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func processInlineMarkdown(input string) string {
	output := strings.Builder{}
	chars := []rune(input)

	i := 0
	for i < len(chars) {
		switch {
		case checkPrefix(chars, i, "**") && findClosing(chars, i+2, "**") != -1:
			closing := findClosing(chars, i+2, "**")
			innerText := string(chars[i+2 : closing])
			output.WriteString("*")
			output.WriteString(escapeMarkdownV2(innerText))
			output.WriteString("*")
			i = closing + 2

		case checkPrefix(chars, i, "*") && findClosing(chars, i+1, "*") != -1:
			closing := findClosing(chars, i+1, "*")
			innerText := string(chars[i+1 : closing])
			output.WriteString("_")
			output.WriteString(escapeMarkdownV2(innerText))
			output.WriteString("_")
			i = closing + 1

		case checkPrefix(chars, i, "_") && findClosing(chars, i+1, "_") != -1:
			closing := findClosing(chars, i+1, "_")
			innerText := string(chars[i+1 : closing])
			output.WriteString("_")
			output.WriteString(escapeMarkdownV2(innerText))
			output.WriteString("_")
			i = closing + 1

		case checkPrefix(chars, i, "~~") && findClosing(chars, i+2, "~~") != -1:
			closing := findClosing(chars, i+2, "~~")
			innerText := string(chars[i+2 : closing])
			output.WriteString("~")
			output.WriteString(escapeMarkdownV2(innerText))
			output.WriteString("~")
			i = closing + 2

		case checkPrefix(chars, i, "```") && findClosing(chars, i+3, "```") != -1:
			closing := findClosing(chars, i+3, "```")
			innerText := string(chars[i+3 : closing])
			output.WriteString("```")
			output.WriteString(innerText)
			output.WriteString("```")
			i = closing + 3

		case checkPrefix(chars, i, "`") && findClosing(chars, i+1, "`") != -1:
			closing := findClosing(chars, i+1, "`")
			innerText := string(chars[i+1 : closing])
			output.WriteString("`")
			output.WriteString(innerText)
			output.WriteString("`")
			i = closing + 1

		case checkPrefix(chars, i, "[") && findClosingLink(chars, i) != -1:
			closingBracket := findClosing(chars, i+1, "]")
			openingParen := closingBracket + 1
			closingParen := findClosing(chars, openingParen+1, ")")

			if closingBracket != -1 && openingParen < len(chars) && string(chars[openingParen]) == "(" && closingParen != -1 {
				linkText := string(chars[i+1 : closingBracket])
				linkURL := string(chars[openingParen+1 : closingParen])

				output.WriteString("[")
				output.WriteString(escapeMarkdownV2(linkText))
				output.WriteString("](")
				output.WriteString(escapeMarkdownV2(linkURL))
				output.WriteString(")")

				i = closingParen + 1
			} else {
				output.WriteString("\\[")
				i++
			}

		default:
			if i < len(chars) {
				needsEscape := slices.Contains(telegramSpecialCharacters, string(chars[i]))

				if needsEscape {
					output.WriteString("\\")
				}
				output.WriteRune(chars[i])
				i++
			}
		}
	}

	return output.String()
}

func checkPrefix(chars []rune, pos int, prefix string) bool {
	if pos+len(prefix) > len(chars) {
		return false
	}

	for i, r := range []rune(prefix) {
		if chars[pos+i] != r {
			return false
		}
	}
	return true
}

func findClosing(chars []rune, start int, delimiter string) int {
	delim := []rune(delimiter)
	i := start

	for i < len(chars) {
		if i > 0 && chars[i-1] == '\\' {
			i++
			continue
		}

		if i+len(delimiter) <= len(chars) {
			match := true
			for j, r := range delim {
				if chars[i+j] != r {
					match = false
					break
				}
			}
			if match {
				return i
			}
		}
		i++
	}
	return -1
}

func findClosingLink(chars []rune, start int) int {
	if start >= len(chars) || chars[start] != '[' {
		return -1
	}

	closingBracket := findClosing(chars, start+1, "]")
	if closingBracket == -1 || closingBracket+1 >= len(chars) {
		return -1
	}

	if chars[closingBracket+1] != '(' {
		return -1
	}

	closingParen := findClosing(chars, closingBracket+2, ")")
	if closingParen == -1 {
		return -1
	}

	return closingParen
}
