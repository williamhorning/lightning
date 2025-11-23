// Package main is the entrypoint for Lightning, the bridge bot thing.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/williamhorning/lightning/internal/app"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
	"github.com/williamhorning/lightning/pkg/platforms/discord"
	"github.com/williamhorning/lightning/pkg/platforms/guilded"
	"github.com/williamhorning/lightning/pkg/platforms/matrix"
	"github.com/williamhorning/lightning/pkg/platforms/stoat"
	"github.com/williamhorning/lightning/pkg/platforms/telegram"
)

func main() {
	cfgPath := flag.String("config", "lightning.toml", "path to the configuration file")
	flag.Parse()

	config, err := app.GetConfig(*cfgPath)
	if err != nil {
		log.Fatalf("failed to get config: %v\n", err)
	}

	app.SetupLogging(config.ErrorURL)

	bot := lightning.NewBot(lightning.BotOptions{Prefix: config.CommandPrefix})

	database, err := data.GetDatabase(config.DatabaseConfig)
	if err != nil {
		log.Fatalf("failed to setup database: %v\n", err)
	}

	app.RegisterCommands(bot, database, config.Username)

	bot.AddPluginType("discord", discord.New)
	bot.AddPluginType("guilded", guilded.New)
	bot.AddPluginType("revolt", stoat.New)
	bot.AddPluginType("telegram", telegram.New)
	bot.AddPluginType("matrix", matrix.New)

	bot.AddHandler(app.Create(database))
	bot.AddHandler(app.Edit(database))
	bot.AddHandler(app.Delete(database))

	for plugin, cfg := range config.Plugins {
		if err := bot.UsePluginType(plugin, "", cfg); err != nil {
			log.Fatalf("failed to setup plugin for %s: %v\n", plugin, err)
		}
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, os.Interrupt, syscall.SIGTERM)
	<-quitChannel

	log.Println("bot stopped")
}
