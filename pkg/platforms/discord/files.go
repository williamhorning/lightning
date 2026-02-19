package discord

import (
	"context"
	"net/http"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

func getMaxFileSize(client *client, channel string) int64 {
	maxFileSize := int64(10485760)

	if ch, ok := client.getChannel(channel); ok {
		if guild, ok := client.getGuild(ch.GuildID); ok {
			switch guild.PremiumTier {
			case premium2:
				maxFileSize = 52428800
			case premium3:
				maxFileSize = 104857600
			case premium1, premiumNone:
			default:
			}
		}
	}

	return maxFileSize
}

func lightningToDiscordFiles(client *client, msg *lightning.Message) []file {
	files := make([]file, 0, len(msg.Attachments))
	maxSize := getMaxFileSize(client, msg.ChannelID)

	for _, original := range msg.Attachments {
		if original.Size > maxSize {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, original.URL, http.NoBody)
		if err != nil {
			cancel()

			continue
		}

		resp, err := http.DefaultClient.Do(req) //nolint:bodyclose,gosec
		if err != nil {
			cancel()

			continue
		}

		files = append(files, file{Name: original.Name, Reader: resp.Body, Cancel: func() {
			_ = resp.Body.Close()

			cancel()
		}})
	}

	return files
}
