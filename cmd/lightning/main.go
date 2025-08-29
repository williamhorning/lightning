// Package main is the entrypoint for Lightning, the bridge bot thing.
package main

import (
	"cmp"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	handler := bridge.NewLogHandler("", slog.LevelDebug)

	slog.SetDefault(slog.New(handler))

	cfg, ok := bridge.GetConfig(*config)
	if !ok {
		os.Exit(1)
	}

	handler.URL = cfg.ErrorURL
	handler.Level = slog.Level(cfg.LogLevel)

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
		slog.Error(fmt.Errorf("failed to setup platform plugins: %w", err).Error())
		os.Exit(1)
	}

	database, err := cfg.DatabaseConfig.GetDatabase()
	if err != nil {
		slog.Error(fmt.Errorf("failed to setup database: %w", err).Error())
		os.Exit(1)
	}

	bridge.Setup(bot, database)

	for plugin, cfg := range cfg.Plugins {
		if err := bot.UsePluginType(plugin, "", cfg); err != nil {
			slog.Error(fmt.Errorf("failed to setup plugin for %s: %w", plugin, err).Error())
			os.Exit(1)
		}
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, os.Interrupt, syscall.SIGTERM)
	<-quitChannel

	slog.Error("bot stopped")
}
