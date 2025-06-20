package lightning

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/rs/zerolog"
)

type Config struct {
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

	SetupCommands(config.CommandPrefix)

	for plugin, cfg := range config.Plugins {
		registerPlugin(plugin, cfg)
	}

	return config, nil
}
