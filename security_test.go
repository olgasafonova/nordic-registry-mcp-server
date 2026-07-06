package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// RateLimiter Tests
// =============================================================================

func TestRateLimiterRefillCap(t *testing.T) {
	rl := NewRateLimiter(2, 10*time.Millisecond)
	defer rl.Close()

	ip := "192.168.1.1"

	// Drain the bucket
	if !rl.Allow(ip) {
		t.Error("First request should be allowed")
	}
	if !rl.Allow(ip) {
		t.Error("Second request should be allowed")
	}
	if rl.Allow(ip) {
		t.Error("Third request should be denied")
	}

	// Wait several intervals; refill must cap at the configured rate
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 2; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d after refill should be allowed", i+1)
		}
	}
	if rl.Allow(ip) {
		t.Error("Tokens must be capped at rate; extra request should be denied")
	}
}

func TestRateLimiterPerIPIsolationAfterExhaustion(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Close()

	// Exhaust one IP
	if !rl.Allow("10.0.0.1") {
		t.Error("First request for 10.0.0.1 should be allowed")
	}
	if rl.Allow("10.0.0.1") {
		t.Error("Second request for 10.0.0.1 should be denied")
	}

	// A different IP must be unaffected
	if !rl.Allow("10.0.0.2") {
		t.Error("Exhausting 10.0.0.1 must not affect 10.0.0.2")
	}
}

// =============================================================================
// setCORSHeaders Tests
// =============================================================================

func TestSetCORSHeaders(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins map[string]bool
		wantACAO       string
		wantVary       string
	}{
		{
			name:           "no origin header sets nothing",
			origin:         "",
			allowedOrigins: map[string]bool{"https://allowed.com": true},
			wantACAO:       "",
			wantVary:       "",
		},
		{
			name:           "wildcard allowlist",
			origin:         "https://anyone.com",
			allowedOrigins: map[string]bool{"*": true},
			wantACAO:       "*",
			wantVary:       "",
		},
		{
			name:           "exact allowed origin echoes origin with Vary",
			origin:         "https://allowed.com",
			allowedOrigins: map[string]bool{"https://allowed.com": true},
			wantACAO:       "https://allowed.com",
			wantVary:       "Origin",
		},
		{
			name:           "disallowed origin gets no allow-origin header",
			origin:         "https://evil.com",
			allowedOrigins: map[string]bool{"https://allowed.com": true},
			wantACAO:       "",
			wantVary:       "",
		},
		{
			name:           "empty allowlist defaults to wildcard",
			origin:         "https://anyone.com",
			allowedOrigins: map[string]bool{},
			wantACAO:       "*",
			wantVary:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			setCORSHeaders(w, req, tt.allowedOrigins)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantACAO {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantACAO)
			}
			if got := w.Header().Get("Vary"); got != tt.wantVary {
				t.Errorf("Vary = %q, want %q", got, tt.wantVary)
			}

			// Common CORS headers are set whenever an Origin is present
			if tt.origin != "" {
				if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
					t.Error("Access-Control-Allow-Methods should be set")
				}
				if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
					t.Error("Access-Control-Allow-Headers should be set")
				}
				if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
					t.Errorf("Access-Control-Max-Age = %q, want %q", got, "86400")
				}
			} else if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
				t.Error("No CORS headers should be set without an Origin header")
			}
		})
	}
}

// =============================================================================
// isTrustedProxy Tests
// =============================================================================

func TestIsTrustedProxy(t *testing.T) {
	sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{
		TrustedProxies: []string{"10.0.0.0/8", "::1"},
	})

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"inside IPv4 CIDR", "10.1.2.3", true},
		{"outside IPv4 CIDR", "192.168.1.1", false},
		{"IPv6 loopback via bare address", "::1", true},
		{"other IPv6 address", "2001:db8::1", false},
		{"invalid IP string", "not-an-ip", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sm.isTrustedProxy(tt.ip); got != tt.want {
				t.Errorf("isTrustedProxy(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsTrustedProxyNoProxiesConfigured(t *testing.T) {
	sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{})

	if sm.isTrustedProxy("10.0.0.1") {
		t.Error("No configured proxies means nothing is trusted")
	}
}

// =============================================================================
// getClientIP Tests
// =============================================================================

func TestGetClientIPForwardedFor(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name:       "rightmost untrusted XFF entry wins",
			remoteAddr: "10.0.0.1:12345",
			xff:        "198.51.100.7, 203.0.113.50, 10.0.0.2",
			want:       "203.0.113.50",
		},
		{
			name:       "all-trusted XFF falls back to remote address",
			remoteAddr: "10.0.0.1:12345",
			xff:        "10.0.0.3, 10.0.0.2",
			want:       "10.0.0.1",
		},
		{
			name:       "X-Real-IP fallback when XFF absent",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "untrusted remote ignores forwarding headers",
			remoteAddr: "192.168.1.1:12345",
			xff:        "203.0.113.50",
			xRealIP:    "203.0.113.9",
			want:       "192.168.1.1",
		},
		{
			name:       "empty XFF entries skipped",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.50, , ",
			want:       "203.0.113.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{
				TrustedProxies: []string{"10.0.0.0/8"},
			})

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			if got := sm.getClientIP(req); got != tt.want {
				t.Errorf("getClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetClientIPMalformedRemoteAddr(t *testing.T) {
	sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1" // no port

	if got := sm.getClientIP(req); got != "192.168.1.1" {
		t.Errorf("getClientIP() = %q, want %q", got, "192.168.1.1")
	}
}

// =============================================================================
// NewSecurityMiddleware Config Normalization Tests
// =============================================================================

func TestNewSecurityMiddlewareBodySizeDefaults(t *testing.T) {
	tests := []struct {
		name        string
		maxBodySize int64
		want        int64
	}{
		{"zero uses default", 0, DefaultMaxBodySize},
		{"negative uses default", -1, DefaultMaxBodySize},
		{"custom value kept", 5000, 5000},
		{"oversized capped at maximum", MaxAllowedBodySize + 1, MaxAllowedBodySize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{MaxBodySize: tt.maxBodySize})
			if sm.maxBodySize != tt.want {
				t.Errorf("maxBodySize = %d, want %d", sm.maxBodySize, tt.want)
			}
		})
	}
}

func TestNewSecurityMiddlewareTrustedProxyParsing(t *testing.T) {
	tests := []struct {
		name    string
		proxies []string
		want    int
	}{
		{"bare IPv4 becomes /32", []string{"10.0.0.1"}, 1},
		{"bare IPv6 becomes /128", []string{"::1"}, 1},
		{"explicit CIDR kept", []string{"10.0.0.0/8"}, 1},
		{"invalid CIDR skipped", []string{"not-a-cidr"}, 0},
		{"empty entries skipped", []string{"", "  "}, 0},
		{"mixed valid and invalid", []string{"10.0.0.0/8", "garbage", "192.168.1.1"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{TrustedProxies: tt.proxies})
			if len(sm.trustedProxies) != tt.want {
				t.Errorf("trustedProxies count = %d, want %d", len(sm.trustedProxies), tt.want)
			}
		})
	}
}

// =============================================================================
// ServeHTTP Behavior Tests
// =============================================================================

func TestSecurityMiddlewareSecurityHeaders(t *testing.T) {
	sm := NewSecurityMiddleware(&mockHandler{}, testLogger(), SecurityConfig{})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	sm.ServeHTTP(w, req)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Cache-Control":          "no-store",
	}
	for header, want := range headers {
		if got := w.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestSecurityMiddlewareOptionsPreflight(t *testing.T) {
	handler := &mockHandler{}
	sm := NewSecurityMiddleware(handler, testLogger(), SecurityConfig{
		AllowedOrigins: []string{"https://allowed.com"},
	})

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Origin", "https://allowed.com")
	w := httptest.NewRecorder()

	sm.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if handler.called {
		t.Error("Preflight must not reach the wrapped handler")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://allowed.com")
	}
}

func TestSecurityMiddlewareWildcardOrigin(t *testing.T) {
	handler := &mockHandler{}
	sm := NewSecurityMiddleware(handler, testLogger(), SecurityConfig{
		AllowedOrigins: []string{"*"},
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("Origin", "https://anyone.example")
	w := httptest.NewRecorder()

	sm.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}
	if !handler.called {
		t.Error("Wildcard allowlist should let any origin through")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestSecurityMiddlewarePerIPRateLimitIsolation(t *testing.T) {
	handler := &mockHandler{}
	sm := NewSecurityMiddleware(handler, testLogger(), SecurityConfig{RateLimit: 1})

	// Exhaust the first IP
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	sm.ServeHTTP(httptest.NewRecorder(), req1)

	w := httptest.NewRecorder()
	sm.ServeHTTP(w, req1)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Second request from same IP: status = %d, want 429", w.Code)
	}

	// A different IP must still get through
	handler.called = false
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.2:12345"
	w2 := httptest.NewRecorder()
	sm.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Request from second IP: status = %d, want 200", w2.Code)
	}
	if !handler.called {
		t.Error("Second IP should not be affected by first IP's rate limit")
	}
}
