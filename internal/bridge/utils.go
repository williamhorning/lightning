package bridge

import "strings"

func parseChannelID(channelID string) (string, string) {
	plugin, chID, found := strings.Cut(channelID, "::")

	if !found {
		return "", channelID
	}

	return plugin, chID
}

func normalizeChannelID(channel bridgeChannel) string {
	plugin := channel.DeprecatedPlugin
	if plugin != "" {
		plugin = strings.Replace(plugin, "bolt-", "", 1)
	}

	if strings.Contains(channel.ID, "::") {
		return channel.ID
	}

	if plugin != "" {
		return plugin + "::" + channel.ID
	}

	return channel.ID
}

func compareChannelIDs(channel bridgeChannel, targetID string) bool {
	normalizedID := normalizeChannelID(channel)
	plugin1, id1 := parseChannelID(normalizedID)
	plugin2, id2 := parseChannelID(targetID)

	if id1 != id2 {
		return false
	}

	return plugin1 == plugin2 || plugin1 == "" || plugin2 == ""
}
