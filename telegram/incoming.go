package telegram

import (
	"fmt"
	"mime"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/williamhorning/lightning"
)

func getCommand(cmdName string, b *gotgbot.Bot, ctx *ext.Context) lightning.CommandEvent {
	if cmdName == "start" {
		cmdName = "help"
	}

	fullText := ctx.EffectiveMessage.Text
	args := []string{}

	if spaceIndex := strings.Index(fullText, " "); spaceIndex != -1 {
		args = strings.Fields(fullText[spaceIndex+1:])
	}

	return lightning.CommandEvent{
		CommandOptions: lightning.CommandOptions{
			Channel: strconv.FormatInt(ctx.EffectiveChat.Id, 10),
			Plugin:  "bolt-telegram",
			Prefix:  "/",
			Time:    time.UnixMilli(ctx.EffectiveMessage.GetDate()),
		},
		Command: cmdName,
		Options: &args,
		EventID: strconv.FormatInt(ctx.EffectiveMessage.GetMessageId(), 10),
		Reply: func(message string) error {
			_, err := ctx.EffectiveMessage.Reply(b, telegramifyMarkdown(message), &gotgbot.SendMessageOpts{
				ParseMode: gotgbot.ParseModeMarkdownV2,
			})
			return err
		},
	}
}

func getMessage(b *gotgbot.Bot, ctx *ext.Context, proxyPath string) lightning.Message {
	msg := lightning.Message{
		BaseMessage: lightning.BaseMessage{
			EventID:   strconv.FormatInt(ctx.EffectiveMessage.GetMessageId(), 10),
			ChannelID: strconv.FormatInt(ctx.EffectiveChat.Id, 10),
			Plugin:    "bolt-telegram",
			Time:      time.UnixMilli(ctx.EffectiveMessage.GetDate() * 1000),
		},
		Attachments: []lightning.Attachment{},
		Author:      getLightningAuthor(b, ctx, proxyPath),
		Embeds:      []lightning.Embed{},
		RepliedTo:   getLightningReply(ctx),
	}

	if text := ctx.EffectiveMessage.Text; text != "" {
		msg.Content = text
		return msg
	}

	if dice := ctx.EffectiveMessage.Dice; dice != nil {
		msg.Content = dice.Emoji + " " + strconv.FormatInt(dice.Value, 10)
		return msg
	}

	if location := ctx.EffectiveMessage.Location; location != nil {
		msg.Content = fmt.Sprintf("https://www.openstreetmap.org/#map=18/%f/%f", location.Latitude, location.Longitude)
		return msg
	}

	msg.Content = ctx.EffectiveMessage.Caption

	addAttachment(b, ctx, &msg, proxyPath)

	return msg
}

func addAttachment(b *gotgbot.Bot, ctx *ext.Context, msg *lightning.Message, proxyPath string) {
	m := ctx.EffectiveMessage

	if doc := m.Document; doc != nil {
		handleAttachment(b, doc.FileId, doc.FileName, doc.FileSize, doc.MimeType, msg, proxyPath)
	} else if anim := m.Animation; anim != nil {
		handleAttachment(b, anim.FileId, anim.FileName, anim.FileSize, anim.MimeType, msg, proxyPath)
	} else if audio := m.Audio; audio != nil {
		handleAttachment(b, audio.FileId, audio.FileName, audio.FileSize, audio.MimeType, msg, proxyPath)
	} else if photos := m.Photo; len(photos) > 0 {
		handleAttachment(b, photos[0].FileId, photos[0].FileId+".jpg", photos[0].FileSize, "image/jpeg", msg, proxyPath)
	} else if sticker := m.Sticker; sticker != nil {
		ext := ".webp"
		if sticker.IsAnimated {
			ext = ".tgs"
		} else if sticker.IsVideo {
			ext = ".webm"
		}
		handleAttachment(b, sticker.FileId, sticker.SetName+ext, sticker.FileSize, "", msg, proxyPath)
	} else if video := m.Video; video != nil {
		handleAttachment(b, video.FileId, video.FileName, video.FileSize, video.MimeType, msg, proxyPath)
	} else if vnote := m.VideoNote; vnote != nil {
		handleAttachment(b, vnote.FileId, "video_note.mp4", vnote.FileSize, "video/mp4", msg, proxyPath)
	} else if voice := m.Voice; voice != nil {
		handleAttachment(b, voice.FileId, "voice.ogg", voice.FileSize, "audio/ogg", msg, proxyPath)
	}
}

func handleAttachment(b *gotgbot.Bot, fileID, name string, size int64, mimeType string, msg *lightning.Message, proxyPath string) {
	ext := ""
	if mimeType != "" {
		if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}

	if f, err := b.GetFile(fileID, nil); err == nil {
		msg.Attachments = append(msg.Attachments, lightning.Attachment{
			URL:  proxyPath + f.FilePath + ext,
			Name: name,
			Size: float64(size) / 1048576,
		})
	}
}

func getLightningAuthor(b *gotgbot.Bot, ctx *ext.Context, proxyPath string) lightning.MessageAuthor {
	author := lightning.MessageAuthor{
		ID:             strconv.FormatInt(ctx.EffectiveSender.Id(), 10),
		Nickname:       ctx.EffectiveSender.Name(),
		Username:       ctx.EffectiveSender.Username(),
		ProfilePicture: nil,
		Color:          "#24A1DE",
	}

	if pics, err := ctx.EffectiveUser.GetProfilePhotos(b, nil); err == nil && pics.TotalCount > 0 {
		var bestPhoto *gotgbot.PhotoSize

		for i := range pics.Photos[0] {
			photo := &pics.Photos[0][i]
			if bestPhoto == nil || photo.Width > bestPhoto.Width {
				bestPhoto = photo
			}
		}

		if bestPhoto != nil {
			if f, err := b.GetFile(bestPhoto.FileId, nil); err == nil {
				url := proxyPath + f.FilePath
				author.ProfilePicture = &url
			}
		}
	}

	return author
}

func getLightningReply(ctx *ext.Context) []string {
	if ctx.EffectiveMessage.ReplyToMessage == nil {
		return nil
	}
	return []string{strconv.FormatInt(ctx.EffectiveMessage.ReplyToMessage.GetMessageId(), 10)}
}
