package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
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
	LogLevel       *int8                 `toml:"log_level"`
	Plugins        map[string]any        `toml:"plugins"`
}

func run(ctx context.Context, c *cli.Command) error {
	var config config

	if _, err := toml.DecodeFile(c.StringArg("config"), &config); err != nil {
		lightning.LogError(err, "something went wrong with loading the config", nil, lightning.ChannelDisabled{})
		os.Exit(1)
	}

	if config.LogLevel == nil {
		defaultLogLevel := int8(1)
		config.LogLevel = &defaultLogLevel
	}

	lightning.SetupLogs(zerolog.Level(*config.LogLevel))

	if err := os.Setenv("LIGHTNING_ERROR_WEBHOOK", config.ErrorURL); err != nil {
		lightning.LogError(err, "something went wrong with setting the webhook url", nil, lightning.ChannelDisabled{})
		os.Exit(1)
	}

	lightning.SetupCommands(config.CommandPrefix)

	for plugin, cfg := range config.Plugins {
		if err := lightning.Plugins.RegisterPlugin(plugin, cfg); err != nil {
			lightning.LogError(err, "something went wrong setting up a plugin", nil, lightning.ChannelDisabled{})
			os.Exit(1)
		}
	}

	db, err := config.DatabaseConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "something went wrong with setting up the database", nil, lightning.ChannelDisabled{})
		os.Exit(1)
	}

	bridge.Setup(db)

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	lightning.LogError(errors.New("lightning instance stopped"), "lightning instance stopped", nil, lightning.ChannelDisabled{})
	return nil
}
