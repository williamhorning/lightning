package telegram

import (
	"strconv"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func telegramToLightningMessage(bot *gotgbot.Bot, ctx *ext.Context, proxyPath string) lightning.Message {
	msg := lightning.Message{
		Author: &lightning.MessageAuthor{
			ID:             strconv.FormatInt(ctx.EffectiveSender.Id(), 10),
			Username:       ctx.EffectiveSender.Name(),
			ProfilePicture: telegramToLightningProfilePicture(bot, ctx, proxyPath),
			Color:          "#24A1DE",
		},
		BaseMessage: lightning.BaseMessage{
			EventID:   strconv.FormatInt(ctx.EffectiveMessage.GetMessageId(), 10),
			ChannelID: strconv.FormatInt(ctx.EffectiveChat.Id, 10),
			Time:      time.UnixMilli(ctx.EffectiveMessage.Date * 1000),
		},
	}

	if ctx.EffectiveMessage.ReplyToMessage != nil {
		msg.RepliedTo = append(msg.RepliedTo, strconv.FormatInt(ctx.EffectiveMessage.ReplyToMessage.GetMessageId(), 10))
	}

	switch {
	case ctx.EffectiveMessage.Text != "":
		msg.Content = ctx.EffectiveMessage.Text
	case ctx.EffectiveMessage.Dice != nil:
		msg.Content = ctx.EffectiveMessage.Dice.Emoji + " " + strconv.FormatInt(ctx.EffectiveMessage.Dice.Value, 10)
	case ctx.EffectiveMessage.Location != nil:
		msg.Content = "https://www.openstreetmap.org/#map=18/" +
			strconv.FormatFloat(ctx.EffectiveMessage.Location.Latitude, 'f', 6, 64) + "/" +
			strconv.FormatFloat(ctx.EffectiveMessage.Location.Longitude, 'f', 6, 64)
	case ctx.EffectiveMessage.Caption != "" || len(ctx.EffectiveMessage.NewChatPhoto) != 0:
		msg.Content = ctx.EffectiveMessage.Caption

		fileID, fileName := getFileDetails(ctx)

		if f, err := bot.GetFile(fileID, nil); err == nil {
			msg.Attachments = append(msg.Attachments, lightning.Attachment{
				URL:  proxyPath + f.FilePath,
				Name: fileName,
				Size: f.FileSize,
			})
		}
	default:
	}

	return msg
}

func telegramToLightningProfilePicture(bot *gotgbot.Bot, ctx *ext.Context, proxyPath string) string {
	var fileID string

	switch {
	case ctx.EffectiveChat != nil:
		chat, err := ctx.EffectiveChat.Get(bot, nil)
		if err != nil || chat.Photo == nil {
			return ""
		}

		fileID = chat.Photo.BigFileId
	case ctx.EffectiveUser != nil:
		pics, err := ctx.EffectiveUser.GetProfilePhotos(bot, nil)
		if err != nil || len(pics.Photos) == 0 {
			return ""
		}

		fileID = getBestPhoto(pics.Photos[0])
	default:
		return ""
	}

	if f, err := bot.GetFile(fileID, nil); err == nil {
		return proxyPath + f.FilePath
	}

	return ""
}

func getFileDetails(ctx *ext.Context) (string, string) { //nolint:revive,cyclop
	switch {
	case len(ctx.EffectiveMessage.NewChatPhoto) != 0:
		return getBestPhoto(ctx.EffectiveMessage.NewChatPhoto), "photo.jpg"
	case len(ctx.EffectiveMessage.Photo) != 0:
		return getBestPhoto(ctx.EffectiveMessage.Photo), "photo.jpg"
	case ctx.EffectiveMessage.Document != nil:
		return ctx.EffectiveMessage.Document.FileId, ctx.EffectiveMessage.Document.FileName
	case ctx.EffectiveMessage.Animation != nil:
		return ctx.EffectiveMessage.Animation.FileId, ctx.EffectiveMessage.Animation.FileName
	case ctx.EffectiveMessage.Audio != nil:
		return ctx.EffectiveMessage.Audio.FileId, ctx.EffectiveMessage.Audio.FileName
	case ctx.EffectiveMessage.Sticker != nil:
		return ctx.EffectiveMessage.Sticker.FileId, ctx.ChannelPost.Sticker.SetName +
			getStickerExtension(ctx.ChannelPost.Sticker)
	case ctx.EffectiveMessage.Video != nil:
		return ctx.EffectiveMessage.Video.FileId, ctx.EffectiveMessage.Video.FileName
	case ctx.EffectiveMessage.VideoNote != nil:
		return ctx.EffectiveMessage.VideoNote.FileId, ctx.EffectiveMessage.VideoNote.FileId + ".mp4"
	case ctx.EffectiveMessage.Voice != nil:
		return ctx.EffectiveMessage.Voice.FileId, ctx.EffectiveMessage.Voice.FileId + ".ogg"
	default:
		return "", ""
	}
}

func getBestPhoto(size []gotgbot.PhotoSize) string {
	bestPhoto := &size[0]

	for _, photo := range size {
		if bestPhoto == nil || photo.Width > bestPhoto.Width {
			bestPhoto = &photo
		}
	}

	return bestPhoto.FileId
}

func getStickerExtension(sticker *gotgbot.Sticker) string {
	if sticker.IsAnimated {
		return ".tgs"
	} else if sticker.IsVideo {
		return ".webm"
	}

	return ".webp"
}
