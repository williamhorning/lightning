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
	ttl   time.Duration
}

// New makes an expiring cache with the given TTL.
func New[K comparable, V any](ttl time.Duration) *Expiring[K, V] {
	cache := &Expiring[K, V]{
		items: make(map[K]cacheItem[V]),
		ttl:   ttl,
	}

	go cache.startCleanupRoutine()

	return cache
}

// Get a key from the cache, returning its value and a whether it exists.
//
//nolint:ireturn
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

	c.items[key] = cacheItem[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete a key from the cache.
func (c *Expiring[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *Expiring[K, V]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

func (c *Expiring[K, V]) startCleanupRoutine() {
	ticker := time.NewTicker(max(c.ttl/2, time.Second))
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}
