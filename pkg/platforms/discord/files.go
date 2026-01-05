package discord

import (
	"context"
	"net/http"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
)

func getMaxFileSize(session *discordgo.Session, channel string) int64 {
	maxFileSize := int64(10485760)

	if ch, err := session.State.Channel(channel); err == nil && ch.GuildID != "" {
		if guild, err := session.State.Guild(ch.GuildID); err == nil {
			switch guild.PremiumTier { //nolint:exhaustive
			case discordgo.PremiumTier2:
				maxFileSize = 52428800
			case discordgo.PremiumTier3:
				maxFileSize = 104857600
			default:
			}
		}
	}

	return maxFileSize
}

func lightningToDiscordFiles(session *discordgo.Session, msg *lightning.Message) ([]*discordgo.File, []func()) {
	files := make([]*discordgo.File, 0, len(msg.Attachments))

	functions := make([]func(), 0, len(msg.Attachments))

	maxSize := getMaxFileSize(session, msg.ChannelID)

	for _, file := range msg.Attachments {
		if file.Size > maxSize {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
		if err != nil {
			cancel()

			continue
		}

		resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
		if err != nil {
			cancel()

			continue
		}

		files = append(files, &discordgo.File{
			Name: file.Name, ContentType: resp.Header.Get("Content-Type"),
			Reader: resp.Body,
		})

		functions = append(functions, cancel, func() { _ = resp.Body.Close() })
	}

	return files, functions
}
