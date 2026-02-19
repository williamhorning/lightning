package discord

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ratelimiter struct {
	mu      sync.Mutex
	global  atomic.Pointer[time.Time]
	buckets map[string]*bucket
}

func (r *ratelimiter) getBucket(key string) *bucket {
	r.mu.Lock()
	defer r.mu.Unlock()

	if b, ok := r.buckets[key]; ok {
		return b
	}

	bkt := &bucket{remaining: 1, ratelimiter: r}

	r.buckets[key] = bkt

	return bkt
}

func (r *ratelimiter) waitUntil(key string) *bucket {
	parts := strings.Split(key, "/")

	bkt := r.getBucket(strings.Join(parts[:min(len(parts), 2)], ""))

	bkt.mu.Lock()

	if bkt.remaining < 1 && bkt.reset.After(time.Now()) {
		time.Sleep(time.Until(bkt.reset))
	} else if r.global.Load().After(time.Now()) {
		time.Sleep(time.Until(*r.global.Load()))
	}

	bkt.remaining--

	return bkt
}

type bucket struct {
	mu          sync.Mutex
	remaining   int64
	reset       time.Time
	ratelimiter *ratelimiter
}

func (b *bucket) release(headers http.Header) {
	defer b.mu.Unlock()

	remaining := headers.Get("X-RateLimit-Remaining") //nolint:canonicalheader
	reset := headers.Get("X-RateLimit-Reset")         //nolint:canonicalheader
	after := headers.Get("X-RateLimit-Reset-After")   //nolint:canonicalheader
	global := headers.Get("X-RateLimit-Global")       //nolint:canonicalheader

	if after != "" {
		resetAt := getTime(time.Now(), after)

		if global != "" {
			b.ratelimiter.global.Swap(&resetAt)
		} else {
			b.reset = resetAt
		}
	} else if reset != "" {
		httpTime, err := http.ParseTime(headers.Get("Date"))
		if err != nil {
			httpTime = time.Now().Add(150 * time.Microsecond)
		}

		b.reset = getTime(httpTime, reset)
	}

	if remaining != "" {
		b.remaining, _ = strconv.ParseInt(remaining, 10, 64)
	}
}

func getTime(current time.Time, resetAfter string) time.Time {
	parsedAfter, err := strconv.ParseFloat(resetAfter, 64)
	if err != nil {
		parsedAfter = 10
	}

	resetAt := current.Add(time.Duration(parsedAfter * float64(time.Second)))

	return resetAt
}
