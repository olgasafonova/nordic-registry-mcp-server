// Package infra provides shared infrastructure components for the Nordic Registry MCP server.
// It includes resilience patterns (circuit breaker, request deduplication) and HTTP utilities.
package infra

import (
	"context"
	"sync"
	"time"
)

// RequestDeduplicator coalesces identical in-flight requests to reduce API load.
// When multiple goroutines request the same data simultaneously, only one request
// is made and all waiters receive the same result.
type RequestDeduplicator struct {
	mu       sync.Mutex
	inflight map[string]*inflightRequest
}

// inflightRequest tracks a request in progress with waiters
type inflightRequest struct {
	done   chan struct{}
	result interface{}
	err    error
	count  int // Number of waiters for metrics
}

// NewRequestDeduplicator creates a new request deduplicator
func NewRequestDeduplicator() *RequestDeduplicator {
	return &RequestDeduplicator{
		inflight: make(map[string]*inflightRequest),
	}
}

// Do executes fn only if no identical request (by key) is in flight.
// If a request with the same key is already running, waits for its result.
// Returns the result, whether it was shared from another request, and any error.
func (d *RequestDeduplicator) Do(ctx context.Context, key string, fn func() (interface{}, error)) (interface{}, bool, error) {
	d.mu.Lock()

	// Check if request is already in flight
	if req, ok := d.inflight[key]; ok {
		req.count++
		d.mu.Unlock()

		// Wait for the in-flight request to complete
		select {
		case <-req.done:
			return req.result, true, req.err
		case <-ctx.Done():
			return nil, false, ctx.Err()
		}
	}

	// Create new in-flight request
	req := &inflightRequest{
		done:  make(chan struct{}),
		count: 1,
	}
	d.inflight[key] = req
	d.mu.Unlock()

	// Execute the actual request
	req.result, req.err = fn()

	// Signal completion and cleanup
	close(req.done)

	d.mu.Lock()
	delete(d.inflight, key)
	d.mu.Unlock()

	return req.result, false, req.err
}

// Stats returns the current number of in-flight requests
func (d *RequestDeduplicator) Stats() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.inflight)
}

// CircuitBreaker prevents cascading failures by failing fast when an API is unresponsive.
// It tracks consecutive failures and opens the circuit after a threshold is reached.
type CircuitBreaker struct {
	mu sync.RWMutex

	// Configuration
	failureThreshold int           // Consecutive failures before opening
	resetTimeout     time.Duration // Time to wait before attempting recovery
	halfOpenMax      int           // Max requests allowed in half-open state

	// State
	state            CircuitState
	consecutiveFails int
	lastFailure      time.Time
	halfOpenCount    int
}

// CircuitState represents the current state of the circuit breaker
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Failing fast, rejecting requests
	CircuitHalfOpen                     // Testing if service recovered
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// NewCircuitBreaker creates a new circuit breaker with sensible defaults
func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: 5,                // Open after 5 consecutive failures
		resetTimeout:     30 * time.Second, // Try recovery after 30 seconds
		halfOpenMax:      2,                // Allow 2 test requests in half-open
		state:            CircuitClosed,
	}
}

// NewCircuitBreakerWithConfig creates a circuit breaker with custom configuration
func NewCircuitBreakerWithConfig(failureThreshold int, resetTimeout time.Duration, halfOpenMax int) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenMax:      halfOpenMax,
		state:            CircuitClosed,
	}
}

// Allow checks if a request should be allowed through the circuit breaker.
// Returns true if the request can proceed, false if circuit is open.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCount = 0
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow limited requests to test recovery
		if cb.halfOpenCount < cb.halfOpenMax {
			cb.halfOpenCount++
			return true
		}
		return false

	default:
		return false
	}
}

// RecordSuccess records a successful request, potentially closing the circuit
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFails = 0

	if cb.state == CircuitHalfOpen {
		// Successful request in half-open state closes the circuit
		cb.state = CircuitClosed
		cb.halfOpenCount = 0
	}
}

// RecordFailure records a failed request, potentially opening the circuit
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFails++
	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.consecutiveFails >= cb.failureThreshold {
			cb.state = CircuitOpen
		}

	case CircuitHalfOpen:
		// Any failure in half-open goes back to open
		cb.state = CircuitOpen
		cb.halfOpenCount = 0
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return CircuitBreakerStats{
		State:            cb.state.String(),
		ConsecutiveFails: cb.consecutiveFails,
		LastFailure:      cb.lastFailure,
	}
}

// CircuitBreakerStats contains circuit breaker statistics
type CircuitBreakerStats struct {
	State            string    `json:"state"`
	ConsecutiveFails int       `json:"consecutive_failures"`
	LastFailure      time.Time `json:"last_failure,omitempty"`
}

// ErrCircuitOpen is returned when the circuit breaker is open
type ErrCircuitOpen struct {
	State    string
	RetryAt  time.Time
	Failures int
}

func (e ErrCircuitOpen) Error() string {
	return "circuit breaker is open: API is experiencing issues, retry after " + e.RetryAt.Format(time.RFC3339)
}
