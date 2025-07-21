package bridge

import "strings"

func parseChannelID(channelID string) (string, string) {
	plugin, id, _ := strings.Cut(channelID, "::")

	return plugin, id
}

func normalizeChannelID(channel bridgeChannel) string {
	if strings.Contains(channel.ID, "::") {
		return channel.ID
	}

	if channel.DeprecatedPlugin != "" {
		plugin := strings.Replace(channel.DeprecatedPlugin, "bolt-", "", 1)

		return plugin + "::" + channel.ID
	}

	return channel.ID
}

func compareChannelIDs(channel bridgeChannel, targetID string) bool {
	normalizedID := normalizeChannelID(channel)
	plugin1, id1, _ := strings.Cut(normalizedID, "::")
	plugin2, id2, _ := strings.Cut(targetID, "::")

	return id1 == id2 && (plugin1 == plugin2 || plugin1 == "" || plugin2 == "")
}
