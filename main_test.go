package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/norway"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)
	defer rl.Close()

	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	if rl.rate != 10 {
		t.Errorf("rate = %d, want 10", rl.rate)
	}
	if rl.interval != time.Minute {
		t.Errorf("interval = %v, want %v", rl.interval, time.Minute)
	}
	if rl.stopCh == nil {
		t.Error("stopCh should be initialized")
	}
}

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)
	defer rl.Close()

	ip := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.Allow(ip) {
		t.Error("4th request should be denied")
	}
}

func TestRateLimiterMultipleIPs(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)
	defer rl.Close()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Each IP should have its own bucket
	for i := 0; i < 2; i++ {
		if !rl.Allow(ip1) {
			t.Errorf("Request %d for ip1 should be allowed", i+1)
		}
		if !rl.Allow(ip2) {
			t.Errorf("Request %d for ip2 should be allowed", i+1)
		}
	}

	// Both should now be rate limited
	if rl.Allow(ip1) {
		t.Error("ip1 should be rate limited")
	}
	if rl.Allow(ip2) {
		t.Error("ip2 should be rate limited")
	}
}

func TestRateLimiterClose(t *testing.T) {
	rl := NewRateLimiter(10, time.Minute)

	// Close should not panic
	rl.Close()

	// Multiple closes should be safe
	rl.Close()
	rl.Close()
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewRateLimiter(1, 10*time.Millisecond)
	defer rl.Close()

	ip := "192.168.1.1"

	// First request allowed
	if !rl.Allow(ip) {
		t.Error("First request should be allowed")
	}

	// Immediate second should be denied
	if rl.Allow(ip) {
		t.Error("Immediate second request should be denied")
	}

	// Wait for refill
	time.Sleep(15 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow(ip) {
		t.Error("Request after refill should be allowed")
	}
}

func TestRecoverPanic(t *testing.T) {
	// This test verifies recoverPanic properly catches panics
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Simulate panic recovery
	func() {
		defer recoverPanic(logger, "test operation")
		panic("test panic")
	}()

	// If we get here, the panic was recovered
}

// Mock handler for testing
type mockHandler struct {
	called bool
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.called = true
	w.WriteHeader(http.StatusOK)
}

func TestSecurityMiddlewareBasic(t *testing.T) {
	// Test basic middleware functionality
	handler := &mockHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{
		MaxBodySize: 1000,
	}

	sm := NewSecurityMiddleware(handler, logger, config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	sm.ServeHTTP(w, req)

	if !handler.called {
		t.Error("Handler should have been called")
	}
}

func TestSecurityMiddlewareWithRateLimit(t *testing.T) {
	handler := &mockHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{
		RateLimit:   2, // 2 requests per minute
		MaxBodySize: 1000,
	}

	sm := NewSecurityMiddleware(handler, logger, config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		handler.called = false
		w := httptest.NewRecorder()
		sm.ServeHTTP(w, req)
		if !handler.called {
			t.Errorf("Request %d should have been allowed", i+1)
		}
	}

	// Third request should be rate limited
	handler.called = false
	w := httptest.NewRecorder()
	sm.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}
}

func TestSecurityMiddlewareBearerToken(t *testing.T) {
	handler := &mockHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{
		BearerToken: "secret-token",
		MaxBodySize: 1000,
	}

	sm := NewSecurityMiddleware(handler, logger, config)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		handlerCalled  bool
	}{
		{"no auth header", "", http.StatusUnauthorized, false},
		{"wrong prefix", "Basic secret-token", http.StatusUnauthorized, false},
		{"wrong token", "Bearer wrong-token", http.StatusUnauthorized, false},
		{"correct token", "Bearer secret-token", http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.called = false
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			sm.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			if handler.called != tt.handlerCalled {
				t.Errorf("Handler called = %v, want %v", handler.called, tt.handlerCalled)
			}
		})
	}
}

func TestSecurityMiddlewareCORS(t *testing.T) {
	handler := &mockHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{
		AllowedOrigins: []string{"https://allowed.com"},
		MaxBodySize:    1000,
	}

	sm := NewSecurityMiddleware(handler, logger, config)

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
	}{
		{"allowed origin GET", "https://allowed.com", "GET", http.StatusOK},
		{"disallowed origin GET", "https://evil.com", "GET", http.StatusForbidden},
		{"allowed origin OPTIONS", "https://allowed.com", "OPTIONS", http.StatusNoContent},
		{"no origin", "", "GET", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler.called = false
			req := httptest.NewRequest(tt.method, "/", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			sm.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestSecurityMiddlewareTrustedProxies(t *testing.T) {
	handler := &mockHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{
		TrustedProxies: []string{"10.0.0.0/8"},
		RateLimit:      1,
		MaxBodySize:    1000,
	}

	sm := NewSecurityMiddleware(handler, logger, config)

	// Request from trusted proxy with X-Forwarded-For
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:12345" // Trusted proxy
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	w := httptest.NewRecorder()

	sm.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSecurityMiddlewareRequestBodyLimit(t *testing.T) {
	handler := &mockHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{
		MaxBodySize: 100,
	}

	sm := NewSecurityMiddleware(handler, logger, config)

	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.ContentLength = 1000 // Exceeds limit
	w := httptest.NewRecorder()

	sm.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413, got %d", w.Code)
	}
}

func TestGetClientIPWithoutProxy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{MaxBodySize: 1000}
	sm := NewSecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), logger, config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	ip := sm.getClientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("Expected IP 192.168.1.1, got %s", ip)
	}
}

func TestGetClientIPWithIPv6(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	config := SecurityConfig{MaxBodySize: 1000}
	sm := NewSecurityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), logger, config)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:12345"

	ip := sm.getClientIP(req)
	if ip != "::1" {
		t.Errorf("Expected IP ::1, got %s", ip)
	}
}

func TestIsLoopbackAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"localhost:8080", true},
		{"[::1]:8080", true},
		{"127.0.0.1", true},
		{"localhost", true},
		{":8080", false},        // empty host binds every interface
		{"0.0.0.0:8080", false}, // wildcard bind
		{"192.168.1.5:8080", false},
		{"[2001:db8::1]:8080", false},
		{"example.com:8080", false}, // non-loopback hostname
	}
	for _, tt := range tests {
		if got := isLoopbackAddr(tt.addr); got != tt.want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", tt.addr, got, tt.want)
		}
	}
}

// TestRegisterHTTPRoutesSecuresDiagnostics verifies that the diagnostics
// endpoints (and any non-probe path) fall through to the secured handler,
// while the liveness probe stays public.
func TestRegisterHTTPRoutesSecuresDiagnostics(t *testing.T) {
	const sentinel = "SECURED"
	securedHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(sentinel))
	})
	mux := registerHTTPRoutes(httpServerConfig{}, securedHandler)

	for _, path := range []string{"/metrics", "/status", "/tools", "/"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if got := w.Body.String(); got != sentinel {
			t.Errorf("path %q: expected secured handler (%q), got %q", path, sentinel, got)
		}
	}

	// The health probe must remain public (must not reach the secured handler).
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if got := w.Body.String(); got == sentinel {
		t.Error("/health routed through secured handler; expected public health handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("/health status = %d, want %d", w.Code, http.StatusOK)
	}
}

// TestNewMCPHandler_ServesProtocol20260728 pins the reason newMCPHandler passes
// Stateless: true. Without it the transport rejects every >= 2026-07-28 request
// with HTTP 400 ("only supported on stateless HTTP servers"), which no build or
// lint pass can detect. Verified by ablation: both subtests fail when the option
// is removed.
//
// tools/list, not server/discover: a stateful handler exempts server/discover so
// clients can still read supported versions off DiscoverResult, so probing
// discover would pass either way and pin nothing.
func TestNewMCPHandler_ServesProtocol20260728(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	clients := &countryClients{norway: norway.NewClient(norway.WithLogger(logger))}
	defer clients.norway.Close()

	server, _ := buildServer(logger, clients)
	handler := newMCPHandler(httpServerConfig{server: server, logger: logger})

	t.Run("tools/list is answered at 2026-07-28", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{` +
			`"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
			`"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},` +
			`"io.modelcontextprotocol/clientCapabilities":{}}}}`

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		// SEP-2243: a request carrying io.modelcontextprotocol/protocolVersion in
		// _meta MUST also send the matching header, or the transport rejects it
		// with -32020 HeaderMismatch.
		req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
		req.Header.Set("Mcp-Method", "tools/list")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
		}
		if got := w.Body.String(); !strings.Contains(got, `"resultType":"complete"`) {
			t.Errorf("response does not carry resultType complete: %s", got)
		}
	})

	t.Run("DELETE is rejected because there are no sessions to terminate", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/", nil))

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", w.Code)
		}
	})
}

// TestToolsListAdvertisesCacheTTL pins that buildServer attaches the ttlMs
// stamping middleware. Without it the SDK leaves ttlMs at 0, which the spec
// reads as "immediately stale", so every client re-fetches the tool list on
// every turn. Verified by ablation: remove the AddReceivingMiddleware call in
// buildServer and this fails with "ttlMs = 0, want 3600000".
//
// Driven through newMCPHandler rather than an in-memory stdio session on
// purpose. This server serves both transports from one *mcp.Server, and a
// stdio-only assertion would pass while the HTTP path went unstamped.
func TestToolsListAdvertisesCacheTTL(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	clients := &countryClients{norway: norway.NewClient(norway.WithLogger(logger))}
	defer clients.norway.Close()

	server, _ := buildServer(logger, clients)
	handler := newMCPHandler(httpServerConfig{server: server, logger: logger})

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{` +
		`"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
		`"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},` +
		`"io.modelcontextprotocol/clientCapabilities":{}}}}`

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	req.Header.Set("Mcp-Method", "tools/list")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	const wantTTL = 3600000 // one hour, matching buildServer
	got := extractTTLMs(t, w.Body.String())
	if got != wantTTL {
		t.Errorf("ttlMs = %d, want %d", got, wantTTL)
	}
}

// extractTTLMs pulls ttlMs off a tools/list response. The handler may answer as
// plain JSON or as an SSE frame depending on the Accept header, so scan for the
// data payload rather than assuming one shape.
func extractTTLMs(t *testing.T, raw string) int {
	t.Helper()

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimPrefix(strings.TrimSpace(line), "data: ")
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var envelope struct {
			Result struct {
				TTLMs *int `json:"ttlMs"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}
		if envelope.Result.TTLMs != nil {
			return *envelope.Result.TTLMs
		}
	}

	t.Fatalf("no ttlMs field found in response: %s", raw)
	return 0
}

// TestNewMCPHandler_ParamHeaderPassthrough proves the x-mcp-header annotation
// on norway_get_company is live on the HTTP transport (SEP-2243), not merely
// present in a schema: a tools/call whose org_number agrees with its
// Mcp-Param-Org-Number header reaches the handler, and any disagreement —
// wrong value or missing header — is rejected with -32020 HeaderMismatch
// before the handler runs. The mismatch half is the proof: a malformed
// annotation is silently ignored by the SDK, so only a rejection demonstrates
// the binding exists. The deliberately invalid org number keeps the agree
// case off the network: argument validation fails fast inside the handler,
// which is already past the header check.
func TestNewMCPHandler_ParamHeaderPassthrough(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	clients := &countryClients{norway: norway.NewClient(norway.WithLogger(logger))}
	defer clients.norway.Close()

	server, _ := buildServer(logger, clients)
	handler := newMCPHandler(httpServerConfig{server: server, logger: logger})

	callCompany := func(t *testing.T, paramHeader string) *httptest.ResponseRecorder {
		t.Helper()
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"_meta":{` +
			`"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
			`"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},` +
			`"io.modelcontextprotocol/clientCapabilities":{}},` +
			`"name":"norway_get_company","arguments":{"org_number":"123"}}}`

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
		req.Header.Set("Mcp-Method", "tools/call")
		req.Header.Set("Mcp-Name", "norway_get_company")
		if paramHeader != "" {
			req.Header.Set("Mcp-Param-Org-Number", paramHeader)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}

	t.Run("annotation is visible in tools/list", func(t *testing.T) {
		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{` +
			`"io.modelcontextprotocol/protocolVersion":"2026-07-28",` +
			`"io.modelcontextprotocol/clientInfo":{"name":"test","version":"1"},` +
			`"io.modelcontextprotocol/clientCapabilities":{}}}}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
		req.Header.Set("Mcp-Method", "tools/list")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
		}
		got := w.Body.String()
		if !strings.Contains(got, `"x-mcp-header":"Org-Number"`) {
			t.Error("tools/list does not expose the x-mcp-header annotation")
		}
		// The SDK drops tools with malformed annotations from tools/list; the
		// annotated tool must still be there.
		if !strings.Contains(got, `"name":"norway_get_company"`) {
			t.Error("norway_get_company missing from tools/list — annotation rejected by the SDK?")
		}
	})

	t.Run("agreeing header reaches the handler", func(t *testing.T) {
		w := callCompany(t, "123")
		if got := w.Body.String(); strings.Contains(got, "-32020") {
			t.Errorf("agreeing body+header was rejected with HeaderMismatch: %s", got)
		}
	})

	t.Run("disagreeing header is rejected with -32020", func(t *testing.T) {
		w := callCompany(t, "999")
		if got := w.Body.String(); !strings.Contains(got, "-32020") {
			t.Errorf("disagreeing header was not rejected with HeaderMismatch: status=%d body=%s", w.Code, got)
		}
	})

	t.Run("missing header for a present annotated param is rejected with -32020", func(t *testing.T) {
		w := callCompany(t, "")
		if got := w.Body.String(); !strings.Contains(got, "-32020") {
			t.Errorf("missing Mcp-Param-Org-Number was not rejected with HeaderMismatch: status=%d body=%s", w.Code, got)
		}
	})
}
