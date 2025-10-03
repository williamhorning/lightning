// Package main is the entrypoint for Lightning, the bridge bot thing.
package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/williamhorning/lightning/internal/bridge"
	"github.com/williamhorning/lightning/pkg/lightning"
	"github.com/williamhorning/lightning/pkg/platforms/discord"
	"github.com/williamhorning/lightning/pkg/platforms/guilded"
	"github.com/williamhorning/lightning/pkg/platforms/matrix"
	"github.com/williamhorning/lightning/pkg/platforms/stoat"
	"github.com/williamhorning/lightning/pkg/platforms/telegram"
)

func main() {
	config := flag.String("config", "lightning.toml", "path to the configuration file")
	flag.Parse()

	handler := bridge.SetupLogging()

	cfg, ok := bridge.GetConfig(*config)
	if !ok {
		os.Exit(1)
	}

	handler.URL = cfg.ErrorURL

	bot := lightning.NewBot(lightning.BotOptions{
		Prefix: cfg.CommandPrefix,
	})

	if err := errors.Join(
		bot.AddPluginType("discord", discord.New),
		bot.AddPluginType("guilded", guilded.New),
		bot.AddPluginType("revolt", stoat.New),
		bot.AddPluginType("telegram", telegram.New),
		bot.AddPluginType("matrix", matrix.New),
	); err != nil {
		log.Fatalf("failed to setup platform plugins: %v\n", err)
	}

	database, err := cfg.DatabaseConfig.GetDatabase()
	if err != nil {
		log.Fatalf("failed to setup database: %v\n", err)
	}

	bridge.Setup(bot, cfg.Author, database)

	for plugin, cfg := range cfg.Plugins {
		if err := bot.UsePluginType(plugin, "", cfg); err != nil {
			log.Fatalf("failed to setup plugin for %s: %v\n", plugin, err)
		}
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, os.Interrupt, syscall.SIGTERM)
	<-quitChannel

	log.Println("bot stopped")
}
