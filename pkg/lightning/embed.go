package lightning

import "strings"

// ToMarkdown transforms a Discord-style embed to markdown.
func (embed *Embed) ToMarkdown() string {
	if embed == nil {
		return ""
	}

	var str strings.Builder

	if embed.Title != "" && embed.URL != "" {
		str.WriteString("[")
		str.WriteString(embed.Title)
		str.WriteString("](")
		str.WriteString(embed.URL)
		str.WriteString(")")
	} else if embed.Title != "" {
		str.WriteString(embed.Title)
	}

	if embed.Timestamp != "" {
		str.WriteString(" (")
		str.WriteString(embed.Timestamp)
		str.WriteString(")")
	}

	str.WriteString("\n\n")

	if embed.Author != nil && embed.Author.URL != "" {
		str.WriteString("[")
		str.WriteString(embed.Author.Name)
		str.WriteString("](")
		str.WriteString(embed.Author.URL)
		str.WriteString(")\n\n")
	} else if embed.Author != nil {
		str.WriteString(embed.Author.Name)
		str.WriteString("\n\n")
	}

	if embed.Description != "" {
		str.WriteString(embed.Description)
		str.WriteString("\n\n")
	}

	str.WriteString(formatMedia(embed.Image))
	str.WriteString(formatMedia(embed.Thumbnail))
	str.WriteString(formatMedia(embed.Video))
	str.WriteString(formatFooter(embed))

	return str.String()
}

func formatFooter(embed *Embed) string {
	var str strings.Builder

	for _, field := range embed.Fields {
		str.WriteString("**" + field.Name + "**\n" + field.Value + "\n\n")
	}

	if embed.Footer != nil && embed.Footer.IconURL != "" {
		str.WriteString("[" + embed.Footer.Text + "](" + embed.Footer.IconURL + ")\n")
	} else if embed.Footer != nil {
		str.WriteString(embed.Footer.Text + "\n")
	}

	return str.String()
}

func formatMedia(media *Media) string {
	if media == nil {
		return ""
	}

	return "![](" + media.URL + ")\n\n"
}
