package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type config struct {
	Database string         `toml:"database"`
	ErrorURL string         `toml:"error_url"`
	Prefix   string         `toml:"prefix"`
	Username string         `toml:"username"`
	Plugins  []pluginConfig `toml:"plugins"`
}

type pluginConfig struct {
	Name   string            `toml:"name,omitempty"`
	Type   string            `toml:"type"`
	Config map[string]string `toml:"config"`
}

func getConfig(file string) (config, error) {
	var config config

	if _, err := toml.DecodeFile(file, &config); err != nil {
		return config, fmt.Errorf("failed decoding config: %w", err)
	}

	if config.Prefix == "" {
		config.Prefix = "!"
	}

	if config.Username == "" {
		config.Username = "lightning"
	}

	return config, nil
}
