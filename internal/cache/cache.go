package cache

import (
	"sync"
	"time"
)

type cacheItem[T any] struct {
	Value     T
	ExpiresAt time.Time
}

type Expiring[K comparable, V any] struct {
	items map[K]cacheItem[V]
	mu    sync.RWMutex
	ttl   time.Duration
}

func New[K comparable, V any](ttl time.Duration) *Expiring[K, V] {
	cache := &Expiring[K, V]{
		items: make(map[K]cacheItem[V]),
		ttl:   ttl,
	}

	go cache.startCleanupRoutine()
	return cache
}

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

func (c *Expiring[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem[V]{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Expiring[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

func (c *Expiring[K, V]) Cleanup() {
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
		c.Cleanup()
	}
}
