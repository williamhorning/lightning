// Package main is the entrypoint for Lightning, the bridge bot thing.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"codeberg.org/jersey/lightning/pkg/platforms/discord"
	"codeberg.org/jersey/lightning/pkg/platforms/matrix"
	"codeberg.org/jersey/lightning/pkg/platforms/stoat"
	"codeberg.org/jersey/lightning/pkg/platforms/telegram"
)

func main() {
	cfgPath := flag.String("config", "lightning.toml", "path to the configuration file")

	flag.Parse()

	config, err := getConfig(*cfgPath)
	if err != nil {
		log.Fatalf("bridge: %v\n", err)
	}

	setupLogging(config.ErrorURL)

	log.Println("bridge: bot started")

	bot := lightning.NewBot(config.Prefix)

	database, err := newDatabase(config.Database)
	if err != nil {
		log.Fatalf("bridge: %v\n", err)
	}

	bot.AddHandler(bridgeCreate(database))
	bot.AddHandler(bridgeEdit(database))
	bot.AddHandler(bridgeDelete(database))

	bot.AddPluginType("discord", discord.New)
	bot.AddPluginType("stoat", stoat.New)
	bot.AddPluginType("telegram", telegram.New)
	bot.AddPluginType("matrix", matrix.New)

	for _, plugin := range config.Plugins {
		go func() {
			err = setupPlugin(bot, &plugin)

			for err != nil {
				log.Println(err.Error())

				time.Sleep(time.Second)

				err = setupPlugin(bot, &plugin)
			}
		}()
	}

	registerCommands(bot, database, config.Username)

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, os.Interrupt, syscall.SIGTERM)
	<-quitChannel

	log.Println("bridge: bot stopped")
}

func setupPlugin(bot *lightning.Bot, plugin *pluginConfig) error {
	if err := bot.UsePluginType(plugin.Type, plugin.Name, plugin.Config); err != nil {
		return fmt.Errorf("bridge: failed to setup plugin for %s instance %q: %w", plugin.Type, plugin.Name, err)
	}

	return nil
}
