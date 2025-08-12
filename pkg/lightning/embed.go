package lightning

import "time"

// ToMarkdown converts a lightning Embed to a Markdown string
// and it handles every field except for the color, which can't
// be represented in Markdown.
func (embed *Embed) ToMarkdown() string {
	if embed == nil {
		return ""
	}

	content := "***embed***"
	content += formatTitle(embed)
	content += formatTimestamp(embed)
	content += formatAuthor(embed)
	content += formatDescription(embed)
	content += formatImage(embed)
	content += formatThumbnail(embed)
	content += formatVideo(embed)
	content += formatFields(embed)
	content += formatFooter(embed)

	return content
}

func formatTitle(embed *Embed) string {
	if embed.Title == nil {
		return ""
	}

	if embed.URL != nil {
		return ": [" + *embed.Title + "](" + *embed.URL + ")"
	}

	return ": " + *embed.Title + ""
}

func formatTimestamp(embed *Embed) string {
	if embed.Timestamp == nil {
		return "\n\n"
	}

	return " (" + embed.Timestamp.Format(time.RFC3339) + ")\n\n"
}

func formatAuthor(embed *Embed) string {
	if embed.Author == nil {
		return ""
	}

	if embed.Author.URL != nil {
		return "[" + embed.Author.Name + "](" + *embed.Author.URL + ")\n\n"
	}

	return embed.Author.Name + "\n\n"
}

func formatDescription(embed *Embed) string {
	if embed.Description == nil {
		return ""
	}

	return *embed.Description + "\n\n"
}

func formatImage(embed *Embed) string {
	if embed.Image == nil {
		return ""
	}

	return "![image](" + embed.Image.URL + ")\n\n"
}

func formatThumbnail(embed *Embed) string {
	if embed.Thumbnail == nil {
		return ""
	}

	return "![thumbnail](" + embed.Thumbnail.URL + ")\n\n"
}

func formatVideo(embed *Embed) string {
	if embed.Video == nil {
		return ""
	}

	return "[video](" + embed.Video.URL + ")\n\n"
}

func formatFields(embed *Embed) string {
	content := ""

	for _, field := range embed.Fields {
		content += "**" + field.Name + "**\n" + field.Value + "\n\n"
	}

	return content
}

func formatFooter(embed *Embed) string {
	if embed.Footer == nil {
		return ""
	}

	if embed.Footer.IconURL != nil {
		return "[" + embed.Footer.Text + "](" + *embed.Footer.IconURL + ")\n"
	}

	return embed.Footer.Text + "\n"
}
