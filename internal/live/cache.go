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

// LiveSnapshot holds the full cached responses for live series, players, and teams.
// Caching this reduces Atlas API calls: all three endpoints serve from cache when valid.
type LiveSnapshot struct {
	Context LiveContext
	Series  []byte
	Players []byte
	Teams   []byte
}

// Cache is a TTL cache for LiveSnapshot.
type Cache struct {
	mu       sync.RWMutex
	entry    *cacheEntry
	ttl      time.Duration
	loadFunc func() (LiveSnapshot, error)
}

type cacheEntry struct {
	snapshot LiveSnapshot
	until    time.Time
}

// NewCache creates a cache with the given TTL.
func NewCache(ttl time.Duration, load func() (LiveSnapshot, error)) *Cache {
	return &Cache{ttl: ttl, loadFunc: load}
}

// Get returns the cached LiveSnapshot if valid, otherwise loads and caches.
func (c *Cache) Get() (LiveSnapshot, error) {
	c.mu.RLock()
	if c.entry != nil && time.Now().Before(c.entry.until) {
		snap := c.entry.snapshot
		c.mu.RUnlock()
		return snap, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entry != nil && time.Now().Before(c.entry.until) {
		return c.entry.snapshot, nil
	}
	snap, err := c.loadFunc()
	if err != nil {
		return LiveSnapshot{}, err
	}
	c.entry = &cacheEntry{snapshot: snap, until: time.Now().Add(c.ttl)}
	return snap, nil
}
