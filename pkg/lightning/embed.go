package lightning

// ToMarkdown converts a lightning Embed to a Markdown string
// and it handles every field except for the color, which can't
// be represented in Markdown.
func (embed *Embed) ToMarkdown() string {
	if embed == nil {
		return ""
	}

	str := ""

	if embed.Title != nil && embed.URL != nil {
		str += "[" + *embed.Title + "](" + *embed.URL + ")"
	} else if embed.Title != nil {
		str += *embed.Title
	}

	if embed.Timestamp != nil {
		str += " (" + *embed.Timestamp + ")"
	}

	str += "\n\n"

	if embed.Author != nil && embed.Author.URL != nil {
		str += "[" + embed.Author.Name + "](" + *embed.Author.URL + ")\n\n"
	} else if embed.Author != nil {
		str += embed.Author.Name + "\n\n"
	}

	if embed.Description != nil {
		str += *embed.Description + "\n\n"
	}

	str += formatMedia(embed.Image) + formatMedia(embed.Thumbnail) + formatMedia(embed.Video) + formatFooter(embed)

	return str
}

func formatFooter(embed *Embed) string {
	str := ""

	for _, field := range embed.Fields {
		str += "**" + field.Name + "**\n" + field.Value + "\n\n"
	}

	if embed.Footer != nil && embed.Footer.IconURL != nil {
		str += "[" + embed.Footer.Text + "](" + *embed.Footer.IconURL + ")\n"
	} else if embed.Footer != nil {
		str += embed.Footer.Text + "\n"
	}

	return str
}

func formatMedia(media *Media) string {
	if media == nil {
		return ""
	}

	return "![](" + media.URL + ")\n\n"
}
