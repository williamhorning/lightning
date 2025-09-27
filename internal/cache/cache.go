// Package cache provides a simple expiring cache
package cache

import (
	"sync"
	"time"
)

// DefaultTTL is the default time-to-live for cache items.
const DefaultTTL = time.Second * 30

type cacheItem[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// An Expiring cache that automatically removes items, when given a TTL.
type Expiring[K comparable, V any] struct {
	items map[K]cacheItem[V]
	mu    sync.RWMutex
	TTL   time.Duration
}

// Get a key from the cache, returning its value and whether it exists.
func (c *Expiring[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		var zero V
		return zero, false
	}

	if time.Now().After(item.ExpiresAt) {
		c.mu.Lock()
		defer c.mu.Unlock()

		item, exists := c.items[key]
		if !exists || time.Now().After(item.ExpiresAt) {
			delete(c.items, key)
			var zero V
			return zero, false
		}
	}

	return item.Value, true
}

// Set a key in the cache, replacing any existing value.
func (c *Expiring[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.TTL == 0 {
		c.TTL = DefaultTTL
	}

	if c.items == nil {
		c.items = make(map[K]cacheItem[V])
	}

	c.items[key] = cacheItem[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.TTL),
	}
}

// Delete a key from the cache.
func (c *Expiring[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}
