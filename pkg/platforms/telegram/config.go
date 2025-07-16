package telegram

import "github.com/williamhorning/lightning/pkg/lightning"

type telegramConfig struct {
	token     string
	proxyURL  string
	proxyPort int64
}

func getTelegramConfig(config any) (telegramConfig, error) {
	cfg, ok := config.(map[string]any)
	if !ok {
		return telegramConfig{}, lightning.LogError(
			lightning.PluginConfigError{},
			"Invalid config for Telegram plugin",
			nil,
			nil,
		)
	}

	token, ok := cfg["token"].(string)
	if !ok || token == "" {
		return telegramConfig{}, lightning.LogError(
			lightning.PluginConfigError{},
			"Missing or invalid token in Telegram plugin config",
			nil,
			nil,
		)
	}

	proxyPort, ok := cfg["proxy_port"].(int64)
	if !ok || proxyPort < 0 {
		return telegramConfig{}, lightning.LogError(
			lightning.PluginConfigError{},
			"Missing or invalid proxy port in Telegram plugin config",
			nil,
			nil,
		)
	}

	proxyURL, ok := cfg["proxy_url"].(string)
	if !ok || proxyURL == "" {
		return telegramConfig{}, lightning.LogError(
			lightning.PluginConfigError{},
			"Missing or invalid proxy URL in Telegram plugin config",
			nil,
			nil,
		)
	}

	return telegramConfig{token, proxyURL, proxyPort}, nil
}
