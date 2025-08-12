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
		return telegramConfig{}, lightning.PluginConfigError{Plugin: "telegram", Message: "invalid config"}
	}

	token, ok := cfg["token"].(string)
	if !ok || token == "" {
		return telegramConfig{}, lightning.PluginConfigError{Plugin: "telegram", Message: "invalid token"}
	}

	proxyPort, ok := cfg["proxy_port"].(int64)
	if !ok || proxyPort < 0 {
		return telegramConfig{}, lightning.PluginConfigError{Plugin: "telegram", Message: "invalid proxy port"}
	}

	proxyURL, ok := cfg["proxy_url"].(string)
	if !ok || proxyURL == "" {
		return telegramConfig{}, lightning.PluginConfigError{Plugin: "telegram", Message: "invalid proxy URL"}
	}

	return telegramConfig{token, proxyURL, proxyPort}, nil
}
