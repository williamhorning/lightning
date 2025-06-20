package revolt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sentinelb51/revoltgo"
	"github.com/williamhorning/lightning"
)

func toEdit(message revoltgo.MessageSend) revoltgo.MessageEditData {
	return revoltgo.MessageEditData{
		Content: message.Content,
		Embeds:  message.Embeds,
	}
}

func getOutgoingMessage(s *revoltgo.Session, message lightning.Message, skipfiles bool, skipmasq bool) revoltgo.MessageSend {
	content := message.Content

	if content == "" && len(message.Embeds) == 0 && len(message.Attachments) == 0 {
		content = "*empty message*"
	} else if len([]rune(content)) > 2000 {
		content = string([]rune(content)[:1997]) + "..."
	}

	return revoltgo.MessageSend{
		Attachments:  getOutgoingAttachments(s, message.Attachments, skipfiles),
		Content:      content,
		Embeds:       getOutgoingEmbeds(message.Embeds),
		Replies:      getOutgoingReplies(message.RepliedTo),
		Masquerade:   getOutgoingMasquerade(message.Author, skipmasq),
		Interactions: nil,
	}
}

func getOutgoingAttachments(s *revoltgo.Session, attachments []lightning.Attachment, skip bool) []string {
	if skip || len(attachments) == 0 {
		return nil
	}

	attachmentIDs := make([]string, 0, len(attachments))

	for _, attachment := range attachments {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		req, err := http.NewRequestWithContext(ctx, "GET", attachment.URL, nil)
		if err != nil {
			cancel()
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			cancel()
			continue
		}

		file, err := s.AttachmentUpload(&revoltgo.File{
			Name:   attachment.Name,
			Reader: &cancelableReadCloser{resp.Body, cancel},
		})

		if err != nil {
			cancel()
			continue
		}

		attachmentIDs = append(attachmentIDs, file.ID)
	}

	return attachmentIDs
}

type cancelableReadCloser struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelableReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

func getOutgoingEmbeds(embeds []lightning.Embed) []*revoltgo.MessageEmbed {
	if len(embeds) == 0 {
		return nil
	}

	result := make([]*revoltgo.MessageEmbed, 0, len(embeds))

	for _, embed := range embeds {
		revoltEmbed := &revoltgo.MessageEmbed{}

		if embed.Title != nil {
			revoltEmbed.Title = *embed.Title
		}

		description := ""
		if embed.Description != nil {
			description = *embed.Description
		}

		if len(embed.Fields) > 0 {
			for _, field := range embed.Fields {
				if description != "" {
					description += "\n\n"
				}
				description += fmt.Sprintf("**%s**\n%s", field.Name, field.Value)
			}
		}

		if description != "" {
			revoltEmbed.Description = description
		}

		if embed.URL != nil {
			revoltEmbed.URL = *embed.URL
		}

		if embed.Color != nil {
			revoltEmbed.Colour = fmt.Sprintf("#%06x", *embed.Color)
		}

		if embed.Image != nil {
			revoltEmbed.Image = &revoltgo.MessageEmbedImage{
				URL:    embed.Image.URL,
				Width:  embed.Image.Width,
				Height: embed.Image.Height,
			}
		}

		if embed.Video != nil {
			revoltEmbed.Video = &revoltgo.MessageEmbedVideo{
				URL:    embed.Video.URL,
				Width:  embed.Video.Width,
				Height: embed.Video.Height,
			}
		}

		if embed.Thumbnail != nil {
			revoltEmbed.IconURL = embed.Thumbnail.URL
		}

		result = append(result, revoltEmbed)
	}

	return result
}

func getOutgoingReplies(replyIDs []string) []*revoltgo.MessageReplies {
	if len(replyIDs) == 0 {
		return nil
	}

	replies := make([]*revoltgo.MessageReplies, len(replyIDs))
	for i, id := range replyIDs {
		replies[i] = &revoltgo.MessageReplies{
			ID:      id,
			Mention: false,
		}
	}

	return replies
}

func getOutgoingMasquerade(author lightning.MessageAuthor, skipmasq bool) *revoltgo.MessageMasquerade {
	if skipmasq {
		return nil
	}

	avatar := ""
	if author.ProfilePicture != nil {
		avatar = *author.ProfilePicture
	}

	return &revoltgo.MessageMasquerade{
		Colour: author.Color,
		Name:   author.Nickname,
		Avatar: avatar,
	}
}
