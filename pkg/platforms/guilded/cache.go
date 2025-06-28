package guilded

import (
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	assetCacheTTL   = 24 * time.Hour
	defaultCacheTTL = 30 * time.Second
)

type guildedCache struct {
	Assets     *lightning.ExpiringCache[string, lightning.Attachment]
	Members    *lightning.ExpiringCache[string, guildedServerMember]
	Webhooks   *lightning.ExpiringCache[string, guildedWebhook]
	WebhookIDs *lightning.ExpiringCache[string, bool]
}

func newGuildedCache() *guildedCache {
	return &guildedCache{
		Assets:     lightning.NewExpiringCache[string, lightning.Attachment](assetCacheTTL),
		Members:    lightning.NewExpiringCache[string, guildedServerMember](defaultCacheTTL),
		Webhooks:   lightning.NewExpiringCache[string, guildedWebhook](defaultCacheTTL),
		WebhookIDs: lightning.NewExpiringCache[string, bool](defaultCacheTTL),
	}
}

var cache = newGuildedCache()
