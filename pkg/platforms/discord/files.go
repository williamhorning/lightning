package discord

import (
	"context"
	"net/http"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func getMaxFileSize(session *bot.Client, channel string) int64 {
	maxFileSize := int64(10485760)

	if id, err := snowflake.Parse(channel); err == nil {
		if ch, ok := session.Caches.Channel(id); ok {
			if guild, ok := session.Caches.Guild(ch.GuildID()); ok {
				switch guild.PremiumTier {
				case discord.PremiumTier2:
					maxFileSize = 52428800
				case discord.PremiumTier3:
					maxFileSize = 104857600
				case discord.PremiumTierNone, discord.PremiumTier1:
				default:
				}
			}
		}
	}

	return maxFileSize
}

func lightningToDiscordFiles(session *bot.Client, msg *lightning.Message) ([]*discord.File, []func()) {
	files := make([]*discord.File, 0, len(msg.Attachments))

	functions := make([]func(), 0, len(msg.Attachments))

	maxSize := getMaxFileSize(session, msg.ChannelID)

	for _, file := range msg.Attachments {
		if file.Size > maxSize {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, http.NoBody)
		if err != nil {
			cancel()

			continue
		}

		resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
		if err != nil {
			cancel()

			continue
		}

		files = append(files, &discord.File{Name: file.Name, Reader: resp.Body})

		functions = append(functions, cancel, func() { _ = resp.Body.Close() })
	}

	return files, functions
}
