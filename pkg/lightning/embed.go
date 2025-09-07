package lightning

// ToMarkdown converts a lightning Embed to a Markdown string
// and it handles every field except for the color, which can't
// be represented in Markdown.
func (embed *Embed) ToMarkdown() string {
	if embed == nil {
		return ""
	}

	return formatTitle(embed) + formatTimestamp(embed) + formatAuthor(embed) + formatDescription(embed) +
		formatImage(embed) + formatThumbnail(embed) + formatVideo(embed) + formatFields(embed) + formatFooter(embed)
}

func formatTitle(embed *Embed) string {
	if embed.Title == nil {
		return ""
	}

	if embed.URL != nil {
		return "[" + *embed.Title + "](" + *embed.URL + ")"
	}

	return *embed.Title
}

func formatTimestamp(embed *Embed) string {
	if embed.Timestamp == nil {
		return "\n\n"
	}

	return " (" + *embed.Timestamp + ")\n\n"
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

	return "![](" + embed.Image.URL + ")\n\n"
}

func formatThumbnail(embed *Embed) string {
	if embed.Thumbnail == nil {
		return ""
	}

	return "![](" + embed.Thumbnail.URL + ")\n\n"
}

func formatVideo(embed *Embed) string {
	if embed.Video == nil {
		return ""
	}

	return embed.Video.URL + "\n\n"
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
