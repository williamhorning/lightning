package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/williamhorning/lightning/internal/bridge"
	"github.com/williamhorning/lightning/pkg/lightning"
	_ "github.com/williamhorning/lightning/pkg/platforms/discord"
	_ "github.com/williamhorning/lightning/pkg/platforms/guilded"
	_ "github.com/williamhorning/lightning/pkg/platforms/revolt"
	_ "github.com/williamhorning/lightning/pkg/platforms/telegram"
)

type config struct {
	CommandPrefix  string                `toml:"prefix,omitempty"`
	DatabaseConfig bridge.DatabaseConfig `toml:"database"`
	ErrorURL       string                `toml:"error_url"`
	LogLevel       int                   `toml:"log_level"`
	Plugins        map[string]any        `toml:"plugins"`
}

func run(cmd *cobra.Command, args []string) {
	var config config

	if len(args) != 1 {
		args = []string{"lightning.toml"}
	}

	if _, err := toml.DecodeFile(args[0], &config); err != nil {
		lightning.LogError(err, "something went wrong with loading the config", nil, nil)
		os.Exit(1)
	}

	lightning.Log.SetLevel(log.Level(config.LogLevel))

	if err := os.Setenv("LIGHTNING_ERROR_WEBHOOK", config.ErrorURL); err != nil {
		lightning.LogError(err, "something went wrong with setting the webhook url", nil, nil)
		os.Exit(1)
	}

	for plugin, cfg := range config.Plugins {
		if err := lightning.Plugins.RegisterPlugin(plugin, cfg); err != nil {
			lightning.LogError(err, "something went wrong setting up a plugin", nil, nil)
			os.Exit(1)
		}
	}

	db, err := config.DatabaseConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "something went wrong with setting up the database", nil, nil)
		os.Exit(1)
	}

	bridge.Setup(db)

	lightning.SetupCommands(config.CommandPrefix)

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	lightning.LogError(errors.New("lightning instance stopped"), "lightning instance stopped", nil, nil)
}
