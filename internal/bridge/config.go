package bridge

import (
	"log"

	"github.com/BurntSushi/toml"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// Config is the configuration for the bridge bot.
type Config struct {
	Author         *lightning.MessageAuthor     `toml:"author,omitempty"`
	DatabaseConfig data.DatabaseConfig          `toml:"database"`
	Plugins        map[string]map[string]string `toml:"plugins"`
	CommandPrefix  string                       `toml:"prefix,omitempty"`
	ErrorURL       string                       `toml:"error_url"`
	LogLevel       int                          `toml:"log_level"`
}

// GetConfig loads the configuration from the given file.
func GetConfig(file string) (Config, bool) {
	var config Config

	if _, err := toml.DecodeFile(file, &config); err != nil {
		log.Printf("bridge: error loading config: %v\n", err)

		return config, false
	}

	if config.Author == nil {
		config.Author = &lightning.MessageAuthor{
			ID:             "lightning",
			Nickname:       "Lightning",
			Username:       "lightning",
			ProfilePicture: "https://williamhorning.dev/assets/lightning.png",
			Color:          "#487C7E",
		}
	}

	return config, true
}
