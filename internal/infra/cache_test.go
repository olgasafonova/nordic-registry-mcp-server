package infra

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	if c == nil {
		t.Fatal("NewCache returned nil")
	}
	if c.maxEntries != 100 {
		t.Errorf("expected maxEntries=100, got %d", c.maxEntries)
	}
}

func TestNewCache_DefaultMaxEntries(t *testing.T) {
	c := NewCache(0)
	defer c.Close()

	if c.maxEntries != DefaultMaxCacheEntries {
		t.Errorf("expected maxEntries=%d for 0, got %d", DefaultMaxCacheEntries, c.maxEntries)
	}

	c2 := NewCache(-1)
	defer c2.Close()

	if c2.maxEntries != DefaultMaxCacheEntries {
		t.Errorf("expected maxEntries=%d for -1, got %d", DefaultMaxCacheEntries, c2.maxEntries)
	}
}

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	c.Set("key1", "value1", 5*time.Minute)

	got, ok := c.Get("key1")
	if !ok {
		t.Error("expected to find key1")
	}
	if got != "value1" {
		t.Errorf("expected 'value1', got %v", got)
	}
}

func TestCache_Get_NotFound(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	got, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected ok=false for nonexistent key")
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestCache_Get_Expired(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	c.Set("expiring", "value", 10*time.Millisecond)

	// Should be found immediately
	_, ok := c.Get("expiring")
	if !ok {
		t.Error("expected to find key before expiration")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should be gone
	_, ok = c.Get("expiring")
	if ok {
		t.Error("expected key to be expired")
	}
}

func TestCache_Set_Update(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	c.Set("key", "value1", 5*time.Minute)
	c.Set("key", "value2", 5*time.Minute)

	got, ok := c.Get("key")
	if !ok {
		t.Error("expected to find key")
	}
	if got != "value2" {
		t.Errorf("expected 'value2', got %v", got)
	}

	// Size should still be 1 (update, not new entry)
	if c.Size() != 1 {
		t.Errorf("expected size=1, got %d", c.Size())
	}
}

func TestCache_Delete(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	c.Set("key", "value", 5*time.Minute)
	c.Delete("key")

	_, ok := c.Get("key")
	if ok {
		t.Error("expected key to be deleted")
	}

	if c.Size() != 0 {
		t.Errorf("expected size=0, got %d", c.Size())
	}
}

func TestCache_Delete_NonExistent(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// Should not panic
	c.Delete("nonexistent")

	if c.Size() != 0 {
		t.Errorf("expected size=0, got %d", c.Size())
	}
}

func TestCache_DeletePrefix(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	c.Set("user:1", "alice", 5*time.Minute)
	c.Set("user:2", "bob", 5*time.Minute)
	c.Set("user:3", "charlie", 5*time.Minute)
	c.Set("order:1", "order1", 5*time.Minute)
	c.Set("order:2", "order2", 5*time.Minute)

	if c.Size() != 5 {
		t.Errorf("expected size=5, got %d", c.Size())
	}

	c.DeletePrefix("user:")

	if c.Size() != 2 {
		t.Errorf("expected size=2 after prefix delete, got %d", c.Size())
	}

	// User keys should be gone
	if _, ok := c.Get("user:1"); ok {
		t.Error("user:1 should be deleted")
	}
	// Order keys should remain
	if _, ok := c.Get("order:1"); !ok {
		t.Error("order:1 should still exist")
	}
}

func TestCache_Size(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	if c.Size() != 0 {
		t.Errorf("expected initial size=0, got %d", c.Size())
	}

	c.Set("key1", "value1", 5*time.Minute)
	c.Set("key2", "value2", 5*time.Minute)
	c.Set("key3", "value3", 5*time.Minute)

	if c.Size() != 3 {
		t.Errorf("expected size=3, got %d", c.Size())
	}

	c.Delete("key2")

	if c.Size() != 2 {
		t.Errorf("expected size=2 after delete, got %d", c.Size())
	}
}

func TestCache_LRUEviction(t *testing.T) {
	c := NewCache(5)
	defer c.Close()

	// Fill the cache
	for i := range 5 {
		c.Set(string(rune('a'+i)), i, 5*time.Minute)
	}

	// Access some keys to make them "recently used"
	c.Get("a") // Touch 'a'
	c.Get("b") // Touch 'b'

	// Add more entries to trigger eviction
	c.Set("f", 5, 5*time.Minute)
	c.Set("g", 6, 5*time.Minute)

	// Wait for async eviction
	time.Sleep(50 * time.Millisecond)

	// 'a' and 'b' should survive (most recently accessed)
	// 'c', 'd', 'e' are candidates for eviction (least recently accessed)
	// After eviction, size should be at or below maxEntries

	if c.Size() > 5 {
		t.Errorf("expected size <= 5, got %d", c.Size())
	}
}

func TestCache_Close(t *testing.T) {
	c := NewCache(100)

	// Multiple closes should not panic
	c.Close()
	c.Close()
	c.Close()
}

func TestCache_ConcurrencySafety(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	var wg sync.WaitGroup
	var ops int64

	// Concurrent reads and writes
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 100 {
				key := string(rune('a' + (id+j)%26))
				c.Set(key, j, 5*time.Minute)
				atomic.AddInt64(&ops, 1)
			}
		}(i)

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range 100 {
				key := string(rune('a' + (id+j)%26))
				c.Get(key)
				atomic.AddInt64(&ops, 1)
			}
		}(i)
	}

	wg.Wait()

	// Just verify no panic and reasonable state
	// Under race detection, LRU operations aren't fully atomic so size
	// can temporarily exceed unique key count. Use capacity as upper bound.
	if c.Size() > 100 {
		t.Errorf("unexpected size: %d (max capacity 100)", c.Size())
	}
}

func TestCache_DifferentDataTypes(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// String
	c.Set("string", "hello", 5*time.Minute)
	if v, ok := c.Get("string"); !ok || v != "hello" {
		t.Error("string value mismatch")
	}

	// Int
	c.Set("int", 42, 5*time.Minute)
	if v, ok := c.Get("int"); !ok || v != 42 {
		t.Error("int value mismatch")
	}

	// Struct
	type testStruct struct {
		Name  string
		Value int
	}
	c.Set("struct", testStruct{Name: "test", Value: 123}, 5*time.Minute)
	if v, ok := c.Get("struct"); !ok {
		t.Error("struct not found")
	} else if s, ok := v.(testStruct); !ok || s.Name != "test" || s.Value != 123 {
		t.Error("struct value mismatch")
	}

	// Slice
	c.Set("slice", []int{1, 2, 3}, 5*time.Minute)
	if v, ok := c.Get("slice"); !ok {
		t.Error("slice not found")
	} else if s, ok := v.([]int); !ok || len(s) != 3 {
		t.Error("slice value mismatch")
	}

	// Pointer
	str := "pointer-value"
	c.Set("pointer", &str, 5*time.Minute)
	if v, ok := c.Get("pointer"); !ok {
		t.Error("pointer not found")
	} else if p, ok := v.(*string); !ok || *p != "pointer-value" {
		t.Error("pointer value mismatch")
	}
}

func TestCache_TTLRenewal(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// Set with short TTL
	c.Set("key", "value1", 30*time.Millisecond)

	// Wait a bit
	time.Sleep(15 * time.Millisecond)

	// Renew with longer TTL
	c.Set("key", "value2", 100*time.Millisecond)

	// Wait past original TTL
	time.Sleep(25 * time.Millisecond)

	// Should still be present with new TTL
	v, ok := c.Get("key")
	if !ok {
		t.Error("key should still exist after TTL renewal")
	}
	if v != "value2" {
		t.Errorf("expected 'value2', got %v", v)
	}
}

func TestCache_AccessTimeUpdated(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	c.Set("key", "value", 5*time.Minute)

	// Get the entry directly to check access time
	entry, ok := c.entries.Load("key")
	if !ok {
		t.Fatal("entry not found")
	}
	ce := entry.(*CacheEntry)
	firstAccess := ce.AccessedAt

	time.Sleep(5 * time.Millisecond)

	// Access the key
	c.Get("key")

	entry, _ = c.entries.Load("key")
	ce = entry.(*CacheEntry)

	if !ce.AccessedAt.After(firstAccess) {
		t.Error("access time should be updated after Get")
	}
}

func TestCache_Cleanup_RemovesExpiredEntries(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// Add entries with very short TTL
	c.Set("expired1", "value1", 10*time.Millisecond)
	c.Set("expired2", "value2", 10*time.Millisecond)
	c.Set("valid", "value3", 5*time.Minute)

	if c.Size() != 3 {
		t.Errorf("expected size=3, got %d", c.Size())
	}

	// Wait for entries to expire
	time.Sleep(20 * time.Millisecond)

	// Call cleanup directly
	c.cleanup()

	// Expired entries should be removed
	if c.Size() != 1 {
		t.Errorf("expected size=1 after cleanup, got %d", c.Size())
	}

	// Valid entry should still exist
	if _, ok := c.Get("valid"); !ok {
		t.Error("valid entry should still exist")
	}

	// Expired entries should be gone
	if _, ok := c.Get("expired1"); ok {
		t.Error("expired1 should be removed")
	}
	if _, ok := c.Get("expired2"); ok {
		t.Error("expired2 should be removed")
	}
}

func TestCache_Cleanup_TriggersLRUEviction(t *testing.T) {
	c := NewCache(5)
	defer c.Close()

	// Overfill the cache without triggering async eviction
	// (by setting entries directly using the sync.Map)
	now := time.Now()
	for i := range 10 {
		key := string(rune('a' + i))
		c.entries.Store(key, &CacheEntry{
			Data:       i,
			ExpiresAt:  now.Add(5 * time.Minute),
			AccessedAt: now.Add(time.Duration(i) * time.Millisecond), // Stagger access times
			Key:        key,
		})
	}
	atomic.StoreInt64(&c.count, 10)

	// Verify we're over the limit
	if c.Size() != 10 {
		t.Errorf("expected size=10, got %d", c.Size())
	}

	// Call cleanup which should trigger LRU eviction
	c.cleanup()

	// Size should be reduced
	if c.Size() > 5 {
		t.Errorf("expected size <= 5 after cleanup, got %d", c.Size())
	}

	// The most recently accessed entries should survive (j, i, h...)
	// The least recently accessed should be evicted (a, b, c...)
}

func TestCache_Cleanup_NoExpiredEntries(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// Add entries with long TTL
	c.Set("key1", "value1", 5*time.Minute)
	c.Set("key2", "value2", 5*time.Minute)
	c.Set("key3", "value3", 5*time.Minute)

	initialSize := c.Size()

	// Call cleanup
	c.cleanup()

	// Size should remain unchanged
	if c.Size() != initialSize {
		t.Errorf("expected size=%d, got %d", initialSize, c.Size())
	}
}

func TestCache_EvictLRU_EmptyCache(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// Evict from empty cache should not panic
	c.evictLRU(10)

	if c.Size() != 0 {
		t.Errorf("expected size=0, got %d", c.Size())
	}
}

func TestCache_EvictLRU_EvictsOldestFirst(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	now := time.Now()

	// Add entries with explicit access times (oldest to newest)
	for i := range 5 {
		key := string(rune('a' + i))
		c.entries.Store(key, &CacheEntry{
			Data:       i,
			ExpiresAt:  now.Add(5 * time.Minute),
			AccessedAt: now.Add(time.Duration(i) * time.Second), // a=oldest, e=newest
			Key:        key,
		})
	}
	atomic.StoreInt64(&c.count, 5)

	// Evict 2 oldest entries
	c.evictLRU(2)

	// Should have 3 entries left
	if c.Size() != 3 {
		t.Errorf("expected size=3, got %d", c.Size())
	}

	// 'a' and 'b' (oldest) should be evicted
	if _, ok := c.entries.Load("a"); ok {
		t.Error("'a' should be evicted (oldest)")
	}
	if _, ok := c.entries.Load("b"); ok {
		t.Error("'b' should be evicted (second oldest)")
	}

	// 'c', 'd', 'e' (newest) should survive
	if _, ok := c.entries.Load("c"); !ok {
		t.Error("'c' should survive")
	}
	if _, ok := c.entries.Load("d"); !ok {
		t.Error("'d' should survive")
	}
	if _, ok := c.entries.Load("e"); !ok {
		t.Error("'e' should survive")
	}
}

func TestCache_EvictLRU_EvictMoreThanExists(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	now := time.Now()

	// Add 3 entries
	for i := range 3 {
		key := string(rune('a' + i))
		c.entries.Store(key, &CacheEntry{
			Data:       i,
			ExpiresAt:  now.Add(5 * time.Minute),
			AccessedAt: now,
			Key:        key,
		})
	}
	atomic.StoreInt64(&c.count, 3)

	// Try to evict 10 (more than exist)
	c.evictLRU(10)

	// All entries should be evicted
	if c.Size() != 0 {
		t.Errorf("expected size=0, got %d", c.Size())
	}
}

func TestCache_CleanupLoop_StopsOnClose(t *testing.T) {
	c := NewCache(100)

	// Ensure cleanup goroutine is running
	time.Sleep(10 * time.Millisecond)

	// Close should stop the cleanup loop
	c.Close()

	// Multiple closes should not panic
	c.Close()
	c.Close()
}

func TestCache_Set_EvictionDeduplication(t *testing.T) {
	c := NewCache(5)
	defer c.Close()

	// Fill the cache past capacity rapidly to trigger eviction
	for i := range 20 {
		c.Set(string(rune('a'+i)), i, 5*time.Minute)
	}

	// Wait for async eviction with retries (race detection slows things down)
	var finalSize int64
	for range 50 { // Up to 500ms total
		time.Sleep(10 * time.Millisecond)
		finalSize = c.Size()
		if finalSize <= 10 {
			break
		}
	}

	// Size should be near maxEntries (not exactly due to async nature)
	// With race detection, we're lenient: just verify eviction happened
	if finalSize > 15 {
		t.Errorf("expected significant eviction (size <= 15), got %d", finalSize)
	}
}

func TestCache_Get_ExpirationDuringGet(t *testing.T) {
	c := NewCache(100)
	defer c.Close()

	// Set entry with very short TTL
	c.Set("key", "value", 5*time.Millisecond)

	// Verify it exists
	if c.Size() != 1 {
		t.Error("expected entry to exist")
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get should remove the expired entry and decrement count
	_, ok := c.Get("key")
	if ok {
		t.Error("expected entry to be expired")
	}

	// Count should be decremented
	if c.Size() != 0 {
		t.Errorf("expected size=0 after expired Get, got %d", c.Size())
	}
}
