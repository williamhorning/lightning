package guilded

import (
	"sync"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	assetCacheTTL   = 24 * time.Hour
	defaultCacheTTL = 30 * time.Second
)

type cacheItem[T any] struct {
	Value     T
	ExpiresAt time.Time
}

type expiringCache[K comparable, V any] struct {
	items map[K]cacheItem[V]
	mu    sync.RWMutex
	ttl   time.Duration
}

func newExpiringCache[K comparable, V any](ttl time.Duration) *expiringCache[K, V] {
	cache := &expiringCache[K, V]{
		items: make(map[K]cacheItem[V]),
		ttl:   ttl,
	}

	go cache.startCleanupRoutine()
	return cache
}

func (c *expiringCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		var zero V
		return zero, false
	}

	if time.Now().After(item.ExpiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		var zero V
		return zero, false
	}

	return item.Value, true
}

func (c *expiringCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *expiringCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *expiringCache[K, V]) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

func (c *expiringCache[K, V]) startCleanupRoutine() {
	cleanupInterval := c.ttl / 2
	if cleanupInterval < time.Second {
		cleanupInterval = time.Second
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.Cleanup()
	}
}

type guildedCache struct {
	Assets   *expiringCache[string, lightning.Attachment]
	Members  *expiringCache[string, guildedServerMember]
	Webhooks *expiringCache[string, guildedWebhook]
}

func newGuildedCache() *guildedCache {
	return &guildedCache{
		Assets:   newExpiringCache[string, lightning.Attachment](assetCacheTTL),
		Members:  newExpiringCache[string, guildedServerMember](defaultCacheTTL),
		Webhooks: newExpiringCache[string, guildedWebhook](defaultCacheTTL),
	}
}

var cache = newGuildedCache()
