package lightning

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
)

type Config struct {
	BridgeDelay    *int64         `toml:"bridge_delay,omitempty"`
	CommandPrefix  string         `toml:"prefix,omitempty"`
	DatabaseConfig DatabaseConfig `toml:"database"`
	ErrorURL       string         `toml:"error_url"`
	LogLevel       *int8          `toml:"log_level"`
	Plugins        map[string]any `toml:"plugins"`
}

func LoadConfig(path string) (Config, error) {
	var config Config

	if _, err := toml.DecodeFile(path, &config); err != nil {
		return Config{}, err
	}

	if config.LogLevel == nil {
		defaultLogLevel := int8(1)
		config.LogLevel = &defaultLogLevel
	}

	Log = Log.Level(zerolog.Level(*config.LogLevel))

	Log.WithLevel(zerolog.Level(*config.LogLevel)).Msg("Set log level!")

	err := os.Setenv("LIGHTNING_ERROR_WEBHOOK", config.ErrorURL)

	if err != nil {
		return Config{}, err
	}

	if config.BridgeDelay != nil {
		Plugins.eventDelay = time.Duration(*config.BridgeDelay) * time.Millisecond
	}

	SetupCommands(config.CommandPrefix)

	for plugin, cfg := range config.Plugins {
		Plugins.registerPlugin(plugin, cfg)
	}

	return config, nil
}
