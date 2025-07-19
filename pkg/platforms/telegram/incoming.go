package telegram

import (
	"fmt"
	"mime"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func getBase(ctx *ext.Context) lightning.BaseMessage {
	return lightning.BaseMessage{
		EventID:   strconv.FormatInt(ctx.EffectiveMessage.GetMessageId(), 10),
		ChannelID: strconv.FormatInt(ctx.EffectiveChat.Id, 10),
		Plugin:    "telegram",
		Time:      time.UnixMilli(ctx.EffectiveMessage.GetDate() * 1000),
	}
}

func getCommand(cmdName string, bot *gotgbot.Bot, ctx *ext.Context) lightning.CommandEvent {
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
			BaseMessage: getBase(ctx),
			Prefix:      "/",
		},
		Command: cmdName,
		Options: args,
		Reply: func(message string) error {
			_, err := ctx.EffectiveMessage.Reply(bot, getMarkdownV2(message), &gotgbot.SendMessageOpts{
				ParseMode: gotgbot.ParseModeMarkdownV2,
			})

			return lightning.LogError(err, "Failed to reply to Telegram command", nil, nil)
		},
	}
}

func getMessage(bot *gotgbot.Bot, ctx *ext.Context, proxyPath string) lightning.Message {
	msg := lightning.Message{
		BaseMessage: getBase(ctx),
		Attachments: []lightning.Attachment{},
		Author:      getLightningAuthor(bot, ctx, proxyPath),
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

	addAttachment(bot, ctx, &msg, proxyPath)

	return msg
}

func addAttachment(bot *gotgbot.Bot, ctx *ext.Context, msg *lightning.Message, proxyPath string) {
	if doc := ctx.EffectiveMessage.Document; doc != nil {
		handleAttachment(bot, doc.FileId, doc.FileName, doc.FileSize, doc.MimeType, msg, proxyPath)
	}

	if anim := ctx.EffectiveMessage.Animation; anim != nil {
		handleAttachment(bot, anim.FileId, anim.FileName, anim.FileSize, anim.MimeType, msg, proxyPath)
	}

	if audio := ctx.EffectiveMessage.Audio; audio != nil {
		handleAttachment(bot, audio.FileId, audio.FileName, audio.FileSize, audio.MimeType, msg, proxyPath)
	}

	if photos := ctx.EffectiveMessage.Photo; len(photos) > 0 {
		handleAttachment(bot, photos[0].FileId, photos[0].FileId+".jpg",
			photos[0].FileSize, "image/jpeg", msg, proxyPath)
	}

	if sticker := ctx.EffectiveMessage.Sticker; sticker != nil {
		extension := getStickerExtension(sticker)
		handleAttachment(bot, sticker.FileId, sticker.SetName+extension, sticker.FileSize, "", msg, proxyPath)
	}

	if video := ctx.EffectiveMessage.Video; video != nil {
		handleAttachment(bot, video.FileId, video.FileName, video.FileSize, video.MimeType, msg, proxyPath)
	}

	if vnote := ctx.EffectiveMessage.VideoNote; vnote != nil {
		handleAttachment(bot, vnote.FileId, "video_note.mp4", vnote.FileSize, "video/mp4", msg, proxyPath)
	}

	if voice := ctx.EffectiveMessage.Voice; voice != nil {
		handleAttachment(bot, voice.FileId, "voice.ogg", voice.FileSize, "audio/ogg", msg, proxyPath)
	}
}

func getStickerExtension(sticker *gotgbot.Sticker) string {
	if sticker.IsAnimated {
		return ".tgs"
	} else if sticker.IsVideo {
		return ".webm"
	}

	return ".webp"
}

func handleAttachment(
	bot *gotgbot.Bot,
	fileID, name string,
	size int64,
	mimeType string,
	msg *lightning.Message,
	proxyPath string,
) {
	extension := ""

	if mimeType != "" {
		if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
			extension = exts[0]
		}
	}

	if f, err := bot.GetFile(fileID, nil); err == nil {
		msg.Attachments = append(msg.Attachments, lightning.Attachment{
			URL:  proxyPath + f.FilePath + extension,
			Name: name,
			Size: size,
		})
	}
}

func getLightningAuthor(bot *gotgbot.Bot, ctx *ext.Context, proxyPath string) lightning.MessageAuthor {
	author := lightning.MessageAuthor{
		ID:             strconv.FormatInt(ctx.EffectiveSender.Id(), 10),
		Nickname:       ctx.EffectiveSender.Name(),
		Username:       ctx.EffectiveSender.Username(),
		ProfilePicture: nil,
		Color:          "#24A1DE",
	}

	if ctx.EffectiveUser == nil {
		return author
	}

	pics, err := ctx.EffectiveUser.GetProfilePhotos(bot, nil)
	if err == nil && pics.TotalCount > 0 {
		var bestPhoto *gotgbot.PhotoSize

		for i := range pics.Photos[0] {
			photo := &pics.Photos[0][i]
			if bestPhoto == nil || photo.Width > bestPhoto.Width {
				bestPhoto = photo
			}
		}

		if bestPhoto != nil {
			if f, err := bot.GetFile(bestPhoto.FileId, nil); err == nil {
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
