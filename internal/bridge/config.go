package bridge

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is the configuration for the bridge bot.
type Config struct {
	DatabaseConfig DatabaseConfig `toml:"database"`
	Plugins        map[string]any `toml:"plugins"`
	CommandPrefix  string         `toml:"prefix,omitempty"`
	ErrorURL       string         `toml:"error_url"`
	LogLevel       int            `toml:"log_level"`
}

// GetConfig loads the configuration from the given file.
func GetConfig(file string) (Config, bool) {
	var config Config

	if _, err := toml.DecodeFile(file, &config); err != nil {
		slog.Error("error loading config", "err", err)

		return config, false
	}

	if err := os.Setenv("LIGHTNING_ERROR_WEBHOOK", config.ErrorURL); err != nil {
		slog.Error("error setting webhook url", "err", err)

		return config, false
	}

	return config, true
}
