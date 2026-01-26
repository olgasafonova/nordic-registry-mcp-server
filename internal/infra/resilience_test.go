package infra

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// RequestDeduplicator Tests
// =============================================================================

func TestNewRequestDeduplicator(t *testing.T) {
	d := NewRequestDeduplicator()
	if d == nil {
		t.Fatal("NewRequestDeduplicator returned nil")
	}
	if d.inflight == nil {
		t.Error("inflight map is nil")
	}
}

func TestRequestDeduplicator_Do_SingleRequest(t *testing.T) {
	d := NewRequestDeduplicator()

	called := 0
	result, shared, err := d.Do(context.Background(), "key1", func() (interface{}, error) {
		called++
		return "value1", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if shared {
		t.Error("expected shared=false for single request")
	}
	if result != "value1" {
		t.Errorf("expected result='value1', got %v", result)
	}
	if called != 1 {
		t.Errorf("expected function to be called once, got %d", called)
	}
}

func TestRequestDeduplicator_Do_ConcurrentRequests(t *testing.T) {
	d := NewRequestDeduplicator()

	var callCount int32
	var wg sync.WaitGroup

	// Start 10 concurrent requests with the same key
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, _, err := d.Do(context.Background(), "shared-key", func() (interface{}, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(50 * time.Millisecond) // Simulate slow operation
				return "shared-value", nil
			})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != "shared-value" {
				t.Errorf("expected 'shared-value', got %v", result)
			}
		}()
	}

	wg.Wait()

	// Function should only be called once due to deduplication
	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected function to be called once, got %d", callCount)
	}
}

func TestRequestDeduplicator_Do_DifferentKeys(t *testing.T) {
	d := NewRequestDeduplicator()

	var callCount int32
	var wg sync.WaitGroup

	// Start requests with different keys
	for i := range 5 {
		wg.Add(1)
		key := "key-" + string(rune('a'+i))
		go func(k string) {
			defer wg.Done()
			_, _, err := d.Do(context.Background(), k, func() (interface{}, error) {
				atomic.AddInt32(&callCount, 1)
				time.Sleep(20 * time.Millisecond)
				return k, nil
			})
			if err != nil {
				t.Errorf("unexpected error for key %s: %v", k, err)
			}
		}(key)
	}

	wg.Wait()

	// Each key should trigger its own call
	if atomic.LoadInt32(&callCount) != 5 {
		t.Errorf("expected 5 calls for different keys, got %d", callCount)
	}
}

func TestRequestDeduplicator_Do_ErrorPropagation(t *testing.T) {
	d := NewRequestDeduplicator()

	expectedErr := errors.New("test error")
	result, _, err := d.Do(context.Background(), "error-key", func() (interface{}, error) {
		return nil, expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestRequestDeduplicator_Do_ContextCancellation(t *testing.T) {
	d := NewRequestDeduplicator()

	// Start a slow request
	go func() {
		_, _, _ = d.Do(context.Background(), "slow-key", func() (interface{}, error) {
			time.Sleep(500 * time.Millisecond)
			return "slow-value", nil
		})
	}()

	// Give the first request time to start
	time.Sleep(20 * time.Millisecond)

	// Start a second request with a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := d.Do(ctx, "slow-key", func() (interface{}, error) {
		return "should-not-call", nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestRequestDeduplicator_Stats(t *testing.T) {
	d := NewRequestDeduplicator()

	// Initially no in-flight requests
	if d.Stats() != 0 {
		t.Errorf("expected 0 in-flight, got %d", d.Stats())
	}

	// Start a slow request
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(started)
		_, _, _ = d.Do(context.Background(), "slow-key", func() (interface{}, error) {
			<-done
			return "value", nil
		})
	}()

	<-started
	time.Sleep(10 * time.Millisecond) // Let the request register

	if d.Stats() != 1 {
		t.Errorf("expected 1 in-flight, got %d", d.Stats())
	}

	close(done)
	time.Sleep(10 * time.Millisecond) // Let the request complete

	if d.Stats() != 0 {
		t.Errorf("expected 0 in-flight after completion, got %d", d.Stats())
	}
}

// =============================================================================
// CircuitBreaker Tests
// =============================================================================

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()
	if cb == nil {
		t.Fatal("NewCircuitBreaker returned nil")
	}
	if cb.failureThreshold != 5 {
		t.Errorf("expected failureThreshold=5, got %d", cb.failureThreshold)
	}
	if cb.resetTimeout != 30*time.Second {
		t.Errorf("expected resetTimeout=30s, got %v", cb.resetTimeout)
	}
	if cb.halfOpenMax != 2 {
		t.Errorf("expected halfOpenMax=2, got %d", cb.halfOpenMax)
	}
	if cb.state != CircuitClosed {
		t.Errorf("expected state=Closed, got %v", cb.state)
	}
}

func TestNewCircuitBreakerWithConfig(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(3, 10*time.Second, 1)

	if cb.failureThreshold != 3 {
		t.Errorf("expected failureThreshold=3, got %d", cb.failureThreshold)
	}
	if cb.resetTimeout != 10*time.Second {
		t.Errorf("expected resetTimeout=10s, got %v", cb.resetTimeout)
	}
	if cb.halfOpenMax != 1 {
		t.Errorf("expected halfOpenMax=1, got %d", cb.halfOpenMax)
	}
}

func TestCircuitBreaker_Allow_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker()

	// Closed circuit should allow all requests
	for range 100 {
		if !cb.Allow() {
			t.Error("closed circuit should allow requests")
		}
	}
}

func TestCircuitBreaker_TransitionToOpen(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(3, 1*time.Second, 1)

	// Record failures to open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Error("circuit should still be closed after 2 failures")
	}

	cb.RecordFailure() // 3rd failure should open circuit
	if cb.State() != CircuitOpen {
		t.Errorf("circuit should be open after 3 failures, got %v", cb.State())
	}

	// Open circuit should reject requests
	if cb.Allow() {
		t.Error("open circuit should reject requests")
	}
}

func TestCircuitBreaker_TransitionToHalfOpen(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(2, 50*time.Millisecond, 1)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatal("circuit should be open")
	}

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Next Allow() should transition to half-open
	if !cb.Allow() {
		t.Error("circuit should allow request after reset timeout")
	}
	if cb.State() != CircuitHalfOpen {
		t.Errorf("circuit should be half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClose(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(2, 10*time.Millisecond, 1)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Transition to half-open
	cb.Allow()
	if cb.State() != CircuitHalfOpen {
		t.Fatal("circuit should be half-open")
	}

	// Success in half-open should close circuit
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Errorf("circuit should be closed after success in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(2, 10*time.Millisecond, 1)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Transition to half-open
	cb.Allow()
	if cb.State() != CircuitHalfOpen {
		t.Fatal("circuit should be half-open")
	}

	// Failure in half-open should re-open circuit
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Errorf("circuit should be open after failure in half-open, got %v", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenMaxRequests(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(2, 10*time.Millisecond, 2)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// First request transitions Open -> HalfOpen (doesn't count against max)
	if !cb.Allow() {
		t.Error("first request (transition) should be allowed")
	}
	// Next two requests count against halfOpenMax=2
	if !cb.Allow() {
		t.Error("second request should be allowed")
	}
	if !cb.Allow() {
		t.Error("third request should be allowed (halfOpenMax=2)")
	}
	// Fourth should be rejected
	if cb.Allow() {
		t.Error("fourth request should be rejected (max=2 reached)")
	}
}

func TestCircuitBreaker_RecordSuccessResetsFails(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(5, 1*time.Second, 1)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	// Success should reset consecutive fails
	cb.RecordSuccess()

	// Need 5 more failures to open circuit
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Error("circuit should still be closed after 4 failures post-success")
	}

	cb.RecordFailure() // 5th failure
	if cb.State() != CircuitOpen {
		t.Error("circuit should be open after 5 failures")
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	cb := NewCircuitBreaker()

	stats := cb.Stats()
	if stats.State != "closed" {
		t.Errorf("expected state='closed', got %q", stats.State)
	}
	if stats.ConsecutiveFails != 0 {
		t.Errorf("expected 0 consecutive fails, got %d", stats.ConsecutiveFails)
	}

	cb.RecordFailure()
	cb.RecordFailure()

	stats = cb.Stats()
	if stats.ConsecutiveFails != 2 {
		t.Errorf("expected 2 consecutive fails, got %d", stats.ConsecutiveFails)
	}
	if stats.LastFailure.IsZero() {
		t.Error("LastFailure should be set")
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}

func TestErrCircuitOpen_Error(t *testing.T) {
	err := ErrCircuitOpen{
		State:    "open",
		RetryAt:  time.Now().Add(30 * time.Second),
		Failures: 5,
	}

	msg := err.Error()
	if msg == "" {
		t.Error("error message should not be empty")
	}
	if !contains(msg, "circuit breaker is open") {
		t.Errorf("error message should contain 'circuit breaker is open', got %q", msg)
	}
}

// =============================================================================
// Concurrency Safety Tests
// =============================================================================

func TestCircuitBreaker_ConcurrencySafety(t *testing.T) {
	cb := NewCircuitBreakerWithConfig(10, 100*time.Millisecond, 5)

	var wg sync.WaitGroup

	// Hammer the circuit breaker from multiple goroutines
	for range 100 {
		wg.Add(3)

		go func() {
			defer wg.Done()
			cb.Allow()
		}()

		go func() {
			defer wg.Done()
			cb.RecordSuccess()
		}()

		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
	}

	wg.Wait()

	// Just verify it didn't panic and state is valid
	state := cb.State()
	if state != CircuitClosed && state != CircuitOpen && state != CircuitHalfOpen {
		t.Errorf("unexpected state: %v", state)
	}
}

func TestRequestDeduplicator_ConcurrencySafety(t *testing.T) {
	d := NewRequestDeduplicator()

	var wg sync.WaitGroup

	// Many concurrent requests with various keys
	for i := range 50 {
		wg.Add(1)
		key := "key-" + string(rune('a'+i%10))
		go func(k string) {
			defer wg.Done()
			_, _, _ = d.Do(context.Background(), k, func() (interface{}, error) {
				time.Sleep(10 * time.Millisecond)
				return k, nil
			})
		}(key)
	}

	wg.Wait()

	// Verify all requests completed
	if d.Stats() != 0 {
		t.Errorf("expected 0 in-flight after all complete, got %d", d.Stats())
	}
}

func TestCircuitBreaker_Allow_UnknownState(t *testing.T) {
	cb := NewCircuitBreaker()

	// Directly set an invalid state to test the default case
	cb.mu.Lock()
	cb.state = CircuitState(99) // Invalid state
	cb.mu.Unlock()

	// Should return false for unknown state
	if cb.Allow() {
		t.Error("unknown state should return false")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
