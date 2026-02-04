package live

import (
	"sync"
	"time"
)

// LiveContext holds team and player IDs derived from live series.
type LiveContext struct {
	TeamIDs   []int
	PlayerIDs []int
}

// Cache is a TTL cache for LiveContext.
type Cache struct {
	mu       sync.RWMutex
	entry    *cacheEntry
	ttl      time.Duration
	loadFunc func() (LiveContext, error)
}

type cacheEntry struct {
	ctx   LiveContext
	until time.Time
}

// NewCache creates a cache with the given TTL.
func NewCache(ttl time.Duration, load func() (LiveContext, error)) *Cache {
	return &Cache{ttl: ttl, loadFunc: load}
}

// Get returns the cached LiveContext if valid, otherwise loads and caches.
func (c *Cache) Get() (LiveContext, error) {
	c.mu.RLock()
	if c.entry != nil && time.Now().Before(c.entry.until) {
		ctx := c.entry.ctx
		c.mu.RUnlock()
		return ctx, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entry != nil && time.Now().Before(c.entry.until) {
		return c.entry.ctx, nil
	}
	ctx, err := c.loadFunc()
	if err != nil {
		return LiveContext{}, err
	}
	c.entry = &cacheEntry{ctx: ctx, until: time.Now().Add(c.ttl)}
	return ctx, nil
}
