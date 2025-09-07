package bridge

import (
	"fmt"
	"log/slog"

	"github.com/BurntSushi/toml"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// Config is the configuration for the bridge bot.
type Config struct {
	Author         *lightning.MessageAuthor `toml:"author,omitempty"`
	DatabaseConfig data.DatabaseConfig      `toml:"database"`
	Plugins        map[string]any           `toml:"plugins"`
	CommandPrefix  string                   `toml:"prefix,omitempty"`
	ErrorURL       string                   `toml:"error_url"`
	LogLevel       int                      `toml:"log_level"`
}

// GetConfig loads the configuration from the given file.
func GetConfig(file string) (Config, bool) {
	var config Config

	if _, err := toml.DecodeFile(file, &config); err != nil {
		slog.Error(fmt.Errorf("error loading config: %w", err).Error())

		return config, false
	}

	if config.Author == nil {
		picture := "https://williamhorning.eu.org/assets/clouds.jpg"

		config.Author = &lightning.MessageAuthor{
			ID:             "lightning",
			Nickname:       "Lightning",
			Username:       "lightning",
			ProfilePicture: &picture,
			Color:          "#487C7E",
		}
	}

	return config, true
}
