package wiki

import (
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

// createTestClient creates a client for testing with minimal config
func createTestClient(t *testing.T) *Client {
	t.Helper()
	config := &Config{
		BaseURL:    "https://test.wiki.com/api.php",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		UserAgent:  "TestClient/1.0",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewClient(config, logger)
}

func TestSetAuditLogger(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	// Initially should be nil
	if client.auditLogger != nil {
		t.Error("Expected auditLogger to be nil initially")
	}

	// Set a NullAuditLogger
	nullLogger := NullAuditLogger{}
	client.SetAuditLogger(nullLogger)

	if client.auditLogger == nil {
		t.Error("Expected auditLogger to be set")
	}

	// Set to nil
	client.SetAuditLogger(nil)

	if client.auditLogger != nil {
		t.Error("Expected auditLogger to be nil after setting nil")
	}
}

func TestNewClient(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.stopCh == nil {
		t.Error("stopCh should be initialized")
	}
	if client.semaphore == nil {
		t.Error("semaphore should be initialized")
	}
	if cap(client.semaphore) != MaxConcurrentRequests {
		t.Errorf("semaphore capacity = %d, want %d", cap(client.semaphore), MaxConcurrentRequests)
	}
}

func TestClientClose(t *testing.T) {
	client := createTestClient(t)

	// Close should not panic
	client.Close()

	// Multiple closes should be safe (stopOnce)
	client.Close()
	client.Close()
}

func TestClientCache(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	// Test setCache and getCached
	key := "test:key1"
	data := map[string]string{"foo": "bar"}

	client.setCache(key, data, "page_content")

	// Retrieve cached data
	cached, ok := client.getCached(key)
	if !ok {
		t.Fatal("Expected cached data to exist")
	}

	cachedMap, ok := cached.(map[string]string)
	if !ok {
		t.Fatalf("Expected map[string]string, got %T", cached)
	}
	if cachedMap["foo"] != "bar" {
		t.Errorf("Expected foo=bar, got foo=%s", cachedMap["foo"])
	}
}

func TestClientCacheExpiration(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	// Set a cache entry with very short TTL by manipulating the entry directly
	key := "test:expiring"
	now := time.Now()

	client.cache.Store(key, &CacheEntry{
		Data:       "test data",
		ExpiresAt:  now.Add(-1 * time.Second), // Already expired
		AccessedAt: now,
		Key:        key,
	})

	// getCached should return false for expired entries
	_, ok := client.getCached(key)
	if ok {
		t.Error("Expected expired entry to return false")
	}
}

func TestClientCacheLRUEviction(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	// Add entries with different access times
	now := time.Now()
	for i := 0; i < 10; i++ {
		key := "lru:test:" + string(rune('a'+i))
		client.cache.Store(key, &CacheEntry{
			Data:       i,
			ExpiresAt:  now.Add(1 * time.Hour),
			AccessedAt: now.Add(time.Duration(i) * time.Second), // Staggered access times
			Key:        key,
		})
		client.cacheCount++
	}

	// Evict 5 entries
	client.evictLRU(5)

	// Count remaining entries
	remaining := 0
	client.cache.Range(func(key, value interface{}) bool {
		remaining++
		return true
	})

	if remaining != 5 {
		t.Errorf("Expected 5 remaining entries, got %d", remaining)
	}
}

func TestClientCleanupCache(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	now := time.Now()

	// Add mix of expired and valid entries
	client.cache.Store("valid:1", &CacheEntry{
		Data:       "valid",
		ExpiresAt:  now.Add(1 * time.Hour),
		AccessedAt: now,
		Key:        "valid:1",
	})
	client.cache.Store("expired:1", &CacheEntry{
		Data:       "expired",
		ExpiresAt:  now.Add(-1 * time.Hour),
		AccessedAt: now,
		Key:        "expired:1",
	})
	client.cacheCount = 2

	// Run cleanup
	client.cleanupCache()

	// Check valid entry still exists
	if _, ok := client.cache.Load("valid:1"); !ok {
		t.Error("Valid entry should still exist")
	}

	// Check expired entry was removed
	if _, ok := client.cache.Load("expired:1"); ok {
		t.Error("Expired entry should have been removed")
	}
}

func TestInvalidateCachePrefix(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	now := time.Now()

	// Add entries with different prefixes
	client.cache.Store("page:one", &CacheEntry{Data: "1", ExpiresAt: now.Add(time.Hour), AccessedAt: now})
	client.cache.Store("page:two", &CacheEntry{Data: "2", ExpiresAt: now.Add(time.Hour), AccessedAt: now})
	client.cache.Store("search:one", &CacheEntry{Data: "3", ExpiresAt: now.Add(time.Hour), AccessedAt: now})
	client.cacheCount = 3

	// Invalidate page prefix
	client.InvalidateCachePrefix("page:")

	// Check page entries removed
	if _, ok := client.cache.Load("page:one"); ok {
		t.Error("page:one should have been invalidated")
	}
	if _, ok := client.cache.Load("page:two"); ok {
		t.Error("page:two should have been invalidated")
	}

	// Check search entry still exists
	if _, ok := client.cache.Load("search:one"); !ok {
		t.Error("search:one should still exist")
	}
}

// Test type assertion helpers

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"valid string", "hello", "hello"},
		{"empty string", "", ""},
		{"nil", nil, ""},
		{"int", 42, ""},
		{"float", 3.14, ""},
		{"bool", true, ""},
		{"map", map[string]string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.input)
			if result != tt.expected {
				t.Errorf("getString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{"valid float64", float64(42), 42},
		{"zero", float64(0), 0},
		{"negative", float64(-10), -10},
		{"decimal truncation", float64(3.9), 3},
		{"nil", nil, 0},
		{"string", "42", 0},
		{"int (wrong type)", 42, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInt(tt.input)
			if result != tt.expected {
				t.Errorf("getInt(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"valid float64", float64(3.14), 3.14},
		{"zero", float64(0), 0},
		{"negative", float64(-2.5), -2.5},
		{"nil", nil, 0},
		{"string", "3.14", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloat64(tt.input)
			if result != tt.expected {
				t.Errorf("getFloat64(%v) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"true", true, true},
		{"false", false, false},
		{"nil", nil, false},
		{"string true", "true", false},
		{"int 1", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBool(tt.input)
			if result != tt.expected {
				t.Errorf("getBool(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetMap(t *testing.T) {
	validMap := map[string]interface{}{"key": "value"}

	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{"valid map", validMap, validMap},
		{"nil", nil, nil},
		{"string", "not a map", nil},
		{"slice", []string{"a"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMap(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("getMap(%v) = %v, want nil", tt.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("getMap(%v) = nil, want %v", tt.input, tt.expected)
				}
			}
		})
	}
}

func TestGetSlice(t *testing.T) {
	validSlice := []interface{}{"a", "b"}

	tests := []struct {
		name     string
		input    interface{}
		expected []interface{}
	}{
		{"valid slice", validSlice, validSlice},
		{"nil", nil, nil},
		{"string", "not a slice", nil},
		{"map", map[string]string{}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSlice(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("getSlice(%v) = %v, want nil", tt.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("getSlice(%v) = nil, want non-nil", tt.input)
				}
			}
		})
	}
}

func TestGetNestedMap(t *testing.T) {
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"value": "found",
			},
		},
	}

	tests := []struct {
		name     string
		keys     []string
		expected bool // whether result should be non-nil
	}{
		{"empty keys", []string{}, true},
		{"one level", []string{"level1"}, true},
		{"two levels", []string{"level1", "level2"}, true},
		{"missing key", []string{"missing"}, false},
		{"missing nested", []string{"level1", "missing"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNestedMap(data, tt.keys...)
			if tt.expected && result == nil {
				t.Errorf("getNestedMap(..., %v) = nil, want non-nil", tt.keys)
			}
			if !tt.expected && result != nil {
				t.Errorf("getNestedMap(..., %v) = %v, want nil", tt.keys, result)
			}
		})
	}
}

func TestGetNestedString(t *testing.T) {
	data := map[string]interface{}{
		"simple": "value1",
		"nested": map[string]interface{}{
			"deep": "value2",
		},
	}

	tests := []struct {
		name     string
		keys     []string
		expected string
	}{
		{"simple key", []string{"simple"}, "value1"},
		{"nested key", []string{"nested", "deep"}, "value2"},
		{"missing key", []string{"missing"}, ""},
		{"empty keys", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNestedString(data, tt.keys...)
			if result != tt.expected {
				t.Errorf("getNestedString(..., %v) = %q, want %q", tt.keys, result, tt.expected)
			}
		})
	}
}

// Concurrency tests

func TestCacheConcurrentAccess(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "concurrent:" + string(rune('a'+n%26))
			client.setCache(key, n, "page_content")
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "concurrent:" + string(rune('a'+n%26))
			client.getCached(key)
		}(i)
	}

	wg.Wait()
}

func TestCacheCleanupLoopStops(t *testing.T) {
	client := createTestClient(t)

	// Verify the goroutine responds to close
	done := make(chan struct{})
	go func() {
		client.Close()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Close did not complete in time")
	}
}
