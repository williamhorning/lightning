// Package main is the entrypoint for Lightning, the bridge bot thing.
package main

import (
	"cmp"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/williamhorning/lightning/internal/bridge"
	"github.com/williamhorning/lightning/pkg/lightning"
	"github.com/williamhorning/lightning/pkg/platforms/discord"
	"github.com/williamhorning/lightning/pkg/platforms/guilded"
	"github.com/williamhorning/lightning/pkg/platforms/revolt"
	"github.com/williamhorning/lightning/pkg/platforms/telegram"
)

func main() {
	config := flag.String("config", "lightning.toml", "path to the configuration file")
	flag.Parse()

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))

	cfg, ok := bridge.GetConfig(*config)
	if !ok {
		os.Exit(1)
	}

	profileURL := "https://williamhorning.eu.org/assets/lightning/logo_color.svg"

	bot := lightning.NewBot(lightning.BotOptions{
		Author: lightning.MessageAuthor{
			Username:       "lightning",
			Nickname:       "lightning",
			ID:             "lightning",
			ProfilePicture: &profileURL,
			Color:          "#487C7E",
		},
		Prefix: cfg.CommandPrefix,
	})

	if err := cmp.Or(
		bot.AddPluginType("discord", discord.New),
		bot.AddPluginType("guilded", guilded.New),
		bot.AddPluginType("revolt", revolt.New),
		bot.AddPluginType("telegram", telegram.New),
	); err != nil {
		slog.Error("failed to setup platform plugins", "err", err)
		os.Exit(1)
	}

	database, err := cfg.DatabaseConfig.GetDatabase()
	if err != nil {
		slog.Error("failed to setup database", "err", err)
		os.Exit(1)
	}

	bridge.Setup(bot, database)

	for plugin, cfg := range cfg.Plugins {
		if err := bot.UsePluginType(plugin, "", cfg); err != nil {
			slog.Error("failed to setup a plugin", "err", err)
			os.Exit(1)
		}
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, os.Interrupt, syscall.SIGTERM)
	<-quitChannel

	slog.Error("bot stopped", "err", bridge.LogError(nil, "bot stopped", nil, nil))
}
