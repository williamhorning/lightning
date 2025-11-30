package app

import (
	"fmt"

	"codeberg.org/jersey/lightning/internal/data"
	"github.com/BurntSushi/toml"
)

// Config is the configuration for the bridge bot.
type Config struct {
	DatabaseConfig data.DatabaseConfig          `toml:"database"`
	Plugins        map[string]map[string]string `toml:"plugins"`
	CommandPrefix  string                       `toml:"prefix"`
	ErrorURL       string                       `toml:"error_url"`
	Username       string                       `toml:"username"`
}

// GetConfig loads the configuration from the given file.
func GetConfig(file string) (Config, error) {
	var config Config

	if _, err := toml.DecodeFile(file, &config); err != nil {
		return config, fmt.Errorf("failed loading config: %w", err)
	}

	if config.Username == "" {
		config.Username = "lightning"
	}

	return config, nil
}
