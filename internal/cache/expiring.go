// Package cache provides a simple expiring cache
package cache

import (
	"sync"
	"time"
)

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
func (c *Expiring[K, V]) Get(key K) (V, bool) { //nolint:nolintlint,ireturn
	c.mu.RLock()

	if c.items == nil {
		c.mu.RUnlock()

		var zero V

		return zero, false
	}

	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		var zero V

		return zero, false
	}

	if time.Now().After(item.ExpiresAt) {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()

		var zero V

		return zero, false
	}

	return item.Value, true
}

// Set a key in the cache, replacing any existing value.
func (c *Expiring[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.TTL == 0 {
		c.TTL = 30 * time.Second
	}

	if c.items == nil {
		c.items = make(map[K]cacheItem[V])
	}

	c.items[key] = cacheItem[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.TTL),
	}
}
