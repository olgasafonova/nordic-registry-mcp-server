package infra

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Cache size limits to prevent unbounded memory growth
const (
	DefaultMaxCacheEntries = 1000            // Maximum number of cache entries
	DefaultCacheCleanup    = 5 * time.Minute // How often to run cache cleanup
)

// CacheEntry holds cached data with expiration and LRU tracking
type CacheEntry struct {
	Data       interface{}
	ExpiresAt  time.Time
	AccessedAt time.Time // For LRU eviction
	Key        string    // Store key for eviction
	mu         sync.Mutex
}

// Cache provides an LRU cache with TTL support
type Cache struct {
	entries    sync.Map // key (string) -> *CacheEntry
	count      int64    // Atomic counter for cache size
	maxEntries int64
	mu         sync.Mutex // Protects eviction operations

	// Graceful shutdown
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewCache creates a new LRU cache with the specified max entries
func NewCache(maxEntries int) *Cache {
	if maxEntries <= 0 {
		maxEntries = DefaultMaxCacheEntries
	}
	c := &Cache{
		maxEntries: int64(maxEntries),
		stopCh:     make(chan struct{}),
	}
	go c.cleanupLoop()
	return c
}

// Get retrieves a cached value if it exists and hasn't expired
func (c *Cache) Get(key string) (interface{}, bool) {
	if entry, ok := c.entries.Load(key); ok {
		ce := entry.(*CacheEntry)
		now := time.Now()
		if now.Before(ce.ExpiresAt) {
			// Update access time for LRU tracking
			ce.mu.Lock()
			ce.AccessedAt = now
			ce.mu.Unlock()
			return ce.Data, true
		}
		// Expired, delete it
		c.entries.Delete(key)
		atomic.AddInt64(&c.count, -1)
	}
	return nil, false
}

// Set stores a value in the cache with the specified TTL
func (c *Cache) Set(key string, data interface{}, ttl time.Duration) {
	now := time.Now()

	// Check if this is a new entry or update
	_, existed := c.entries.Load(key)

	c.entries.Store(key, &CacheEntry{
		Data:       data,
		ExpiresAt:  now.Add(ttl),
		AccessedAt: now,
		Key:        key,
	})

	// Only increment count for new entries
	if !existed {
		newCount := atomic.AddInt64(&c.count, 1)

		// Trigger eviction if over limit (async to not block caller)
		if newCount > c.maxEntries {
			go c.evictLRU(int(newCount - c.maxEntries + c.maxEntries/10))
		}
	}
}

// Delete removes a key from the cache
func (c *Cache) Delete(key string) {
	if _, existed := c.entries.LoadAndDelete(key); existed {
		atomic.AddInt64(&c.count, -1)
	}
}

// DeletePrefix removes all cache entries with keys starting with prefix
func (c *Cache) DeletePrefix(prefix string) {
	var deletedCount int64
	c.entries.Range(func(key, value interface{}) bool {
		if k := key.(string); len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			c.entries.Delete(key)
			deletedCount++
		}
		return true
	})
	if deletedCount > 0 {
		atomic.AddInt64(&c.count, -deletedCount)
	}
}

// Size returns the current number of entries in the cache
func (c *Cache) Size() int64 {
	return atomic.LoadInt64(&c.count)
}

// Close stops the background cleanup goroutine
func (c *Cache) Close() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

// cleanupLoop periodically cleans up expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(DefaultCacheCleanup)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes expired entries and evicts LRU entries if over limit
func (c *Cache) cleanup() {
	now := time.Now()
	var expiredCount int64

	// First pass: remove expired entries
	c.entries.Range(func(key, value interface{}) bool {
		ce := value.(*CacheEntry)
		if now.After(ce.ExpiresAt) {
			c.entries.Delete(key)
			expiredCount++
		}
		return true
	})

	// Update counter for expired entries
	if expiredCount > 0 {
		atomic.AddInt64(&c.count, -expiredCount)
	}

	// Check if we need to evict for size limit
	currentCount := atomic.LoadInt64(&c.count)
	if currentCount > c.maxEntries {
		c.evictLRU(int(currentCount - c.maxEntries + c.maxEntries/10)) // Evict 10% extra
	}
}

// evictLRU removes the least recently used entries
func (c *Cache) evictLRU(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Collect all entries with their access times
	type entryInfo struct {
		key        string
		accessedAt time.Time
	}
	var entries []entryInfo

	c.entries.Range(func(key, value interface{}) bool {
		ce := value.(*CacheEntry)
		ce.mu.Lock()
		accessedAt := ce.AccessedAt
		ce.mu.Unlock()
		entries = append(entries, entryInfo{
			key:        key.(string),
			accessedAt: accessedAt,
		})
		return true
	})

	// Sort by access time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].accessedAt.Before(entries[j].accessedAt)
	})

	// Evict the oldest entries
	evicted := 0
	for _, entry := range entries {
		if evicted >= count {
			break
		}
		c.entries.Delete(entry.key)
		evicted++
	}

	if evicted > 0 {
		atomic.AddInt64(&c.count, -int64(evicted))
	}
}
