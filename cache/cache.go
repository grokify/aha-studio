// Package cache provides caching for AQL query results.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/grokify/aha-studio/result"
)

// Cache provides in-memory caching for query results.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*entry
	ttl     time.Duration
	maxSize int
}

type entry struct {
	result    *result.Result
	expiresAt time.Time
}

// Options configures the cache.
type Options struct {
	// TTL is the time-to-live for cache entries. Default: 5 minutes.
	TTL time.Duration

	// MaxSize is the maximum number of entries. Default: 100.
	MaxSize int
}

// DefaultOptions returns default cache options.
func DefaultOptions() Options {
	return Options{
		TTL:     5 * time.Minute,
		MaxSize: 100,
	}
}

// New creates a new cache with the given options.
func New(opts Options) *Cache {
	if opts.TTL == 0 {
		opts.TTL = 5 * time.Minute
	}
	if opts.MaxSize == 0 {
		opts.MaxSize = 100
	}
	return &Cache{
		entries: make(map[string]*entry),
		ttl:     opts.TTL,
		maxSize: opts.MaxSize,
	}
}

// Get retrieves a cached result by key.
// Returns nil if the entry doesn't exist or has expired.
func (c *Cache) Get(key string) *result.Result {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.entries[key]
	if !ok {
		return nil
	}

	if time.Now().After(e.expiresAt) {
		// Entry expired, will be cleaned up later
		return nil
	}

	return e.result
}

// Set stores a result in the cache.
func (c *Cache) Set(key string, res *result.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict expired entries if we're at capacity
	if len(c.entries) >= c.maxSize {
		c.evictExpired()
	}

	// If still at capacity, evict oldest entry
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &entry{
		result:    res,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*entry)
}

// Size returns the number of entries in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache statistics.
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	expired := 0
	for _, e := range c.entries {
		if now.After(e.expiresAt) {
			expired++
		}
	}

	return Stats{
		Size:    len(c.entries),
		Expired: expired,
		MaxSize: c.maxSize,
		TTL:     c.ttl,
	}
}

// Stats contains cache statistics.
type Stats struct {
	Size    int
	Expired int
	MaxSize int
	TTL     time.Duration
}

// evictExpired removes all expired entries. Must be called with lock held.
func (c *Cache) evictExpired() {
	now := time.Now()
	for key, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// evictOldest removes the oldest entry. Must be called with lock held.
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, e := range c.entries {
		if oldestKey == "" || e.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = e.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// KeyFromQuery generates a cache key from a query string and options.
func KeyFromQuery(query string, productID string) string {
	h := sha256.New()
	h.Write([]byte(query))
	h.Write([]byte(productID))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
