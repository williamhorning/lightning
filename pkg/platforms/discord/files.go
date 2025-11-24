package discord

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/internal/workaround"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func getMaxFileSize(session *discordgo.Session, channel string) int64 {
	maxFileSize := int64(10485760)

	if ch, err := session.State.Channel(channel); err == nil && ch.GuildID != "" {
		if guild, err := session.State.Guild(ch.GuildID); err == nil {
			switch guild.PremiumTier {
			case discordgo.PremiumTier2:
				maxFileSize = 52428800
			case discordgo.PremiumTier3:
				maxFileSize = 104857600
			case discordgo.PremiumTier1, discordgo.PremiumTierNone:
			default:
			}
		}
	}

	return maxFileSize
}

type cancelableReadCloser struct {
	io.ReadCloser

	cancel context.CancelFunc
}

func (c *cancelableReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()

	if err != nil {
		return fmt.Errorf("discord: failed closing cancelable read closer: %w", err)
	}

	return nil
}

func lightningToDiscordFiles(session *discordgo.Session, msg *lightning.Message) []*discordgo.File {
	files := make([]*discordgo.File, 0, len(msg.Attachments))

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

		resp, err := workaround.Do(req) //nolint:bodyclose // see cancelableReadCloser
		if err != nil {
			cancel()

			continue
		}

		files = append(files, &discordgo.File{
			Name: file.Name, ContentType: resp.Header.Get("Content-Type"),
			Reader: cancelableReadCloser{resp.Body, cancel},
		})
	}

	return files
}
