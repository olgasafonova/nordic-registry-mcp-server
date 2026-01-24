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
	for i := 0; i < 5; i++ {
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
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := string(rune('a' + (id+j)%26))
				c.Set(key, j, 5*time.Minute)
				atomic.AddInt64(&ops, 1)
			}
		}(i)

		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				key := string(rune('a' + (id+j)%26))
				c.Get(key)
				atomic.AddInt64(&ops, 1)
			}
		}(i)
	}

	wg.Wait()

	// Just verify no panic and reasonable state
	if c.Size() > 26 {
		t.Errorf("unexpected size: %d (max 26 unique keys)", c.Size())
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
