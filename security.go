// Nordic Registry MCP Server - Security middleware for HTTP transport
// Provides rate limiting, CORS, bearer token auth, and trusted-proxy handling.
package main

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// recoverPanic wraps a function with panic recovery and returns an error instead of crashing
func recoverPanic(logger *slog.Logger, operation string) {
	if r := recover(); r != nil {
		logger.Error("Panic recovered",
			"operation", operation,
			"panic", r,
			"stack", string(debug.Stack()))
	}
}

// =============================================================================
// Security Middleware for HTTP Transport
// =============================================================================

// RateLimiter implements a simple token bucket rate limiter per IP
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	rate     int
	interval time.Duration
	cleanup  time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
}

type clientLimiter struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a rate limiter with specified rate per interval
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     rate,
		interval: interval,
		cleanup:  interval * 10,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Close gracefully shuts down the rate limiter cleanup loop
func (rl *RateLimiter) Close() {
	rl.stopOnce.Do(func() {
		close(rl.stopCh)
	})
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, client := range rl.clients {
				if now.Sub(client.lastCheck) > rl.cleanup {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &clientLimiter{
			tokens:    rl.rate - 1,
			lastCheck: now,
		}
		return true
	}

	elapsed := now.Sub(client.lastCheck)
	refill := int(elapsed/rl.interval) * rl.rate
	client.tokens = min(client.tokens+refill, rl.rate)
	client.lastCheck = now

	if client.tokens > 0 {
		client.tokens--
		return true
	}
	return false
}

// SecurityMiddleware wraps an HTTP handler with security checks
type SecurityMiddleware struct {
	handler        http.Handler
	logger         *slog.Logger
	bearerToken    string
	allowedOrigins map[string]bool
	rateLimiter    *RateLimiter
	maxBodySize    int64
	trustedProxies []*net.IPNet
}

// SecurityConfig holds configuration for the security middleware
type SecurityConfig struct {
	BearerToken    string // #nosec G117 -- config struct field, not serialized to external output
	AllowedOrigins []string
	RateLimit      int
	MaxBodySize    int64
	TrustedProxies []string
}

const (
	DefaultMaxBodySize = 2 * 1024 * 1024
	MaxAllowedBodySize = 10 * 1024 * 1024
)

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(handler http.Handler, logger *slog.Logger, config SecurityConfig) *SecurityMiddleware {
	origins := make(map[string]bool)
	for _, o := range config.AllowedOrigins {
		origins[o] = true
	}

	var rl *RateLimiter
	if config.RateLimit > 0 {
		rl = NewRateLimiter(config.RateLimit, time.Minute)
	}

	maxBody := config.MaxBodySize
	if maxBody <= 0 {
		maxBody = DefaultMaxBodySize
	} else if maxBody > MaxAllowedBodySize {
		maxBody = MaxAllowedBodySize
	}

	var trustedProxies []*net.IPNet
	for _, cidr := range config.TrustedProxies {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Warn("Invalid trusted proxy CIDR, skipping", "cidr", cidr, "error", err)
			continue
		}
		trustedProxies = append(trustedProxies, ipNet)
	}

	return &SecurityMiddleware{
		handler:        handler,
		logger:         logger,
		bearerToken:    config.BearerToken,
		allowedOrigins: origins,
		rateLimiter:    rl,
		maxBodySize:    maxBody,
		trustedProxies: trustedProxies,
	}
}

func (s *SecurityMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP := s.getClientIP(r)

	if r.Body != nil && r.ContentLength > s.maxBodySize {
		s.logger.Warn("Request body too large", "client_ip", clientIP, "content_length", r.ContentLength)
		http.Error(w, fmt.Sprintf("Request body too large (max %d bytes)", s.maxBodySize), http.StatusRequestEntityTooLarge)
		return
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, s.maxBodySize)
	}

	if s.rateLimiter != nil && !s.rateLimiter.Allow(clientIP) {
		s.logger.Warn("Rate limit exceeded", "client_ip", clientIP, "path", r.URL.Path)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	origin := r.Header.Get("Origin")
	if origin != "" && len(s.allowedOrigins) > 0 {
		if !s.allowedOrigins[origin] && !s.allowedOrigins["*"] {
			s.logger.Warn("Origin not allowed", "origin", origin, "client_ip", clientIP)
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}
	}

	if s.bearerToken != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			s.logger.Warn("Missing Bearer token", "client_ip", clientIP, "path", r.URL.Path)
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		// Length check uses constant-time comparison to prevent timing attacks on token length
		if len(token) != len(s.bearerToken) || subtle.ConstantTimeCompare([]byte(token), []byte(s.bearerToken)) != 1 {
			s.logger.Warn("Invalid Bearer token", "client_ip", clientIP, "path", r.URL.Path)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")

	if r.Method == http.MethodOptions {
		setCORSHeaders(w, r, s.allowedOrigins)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	setCORSHeaders(w, r, s.allowedOrigins)

	s.logger.Info("HTTP request", "method", r.Method, "path", r.URL.Path, "client_ip", clientIP, "origin", origin)
	s.handler.ServeHTTP(w, r)
}

func setCORSHeaders(w http.ResponseWriter, r *http.Request, allowedOrigins map[string]bool) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}

	if len(allowedOrigins) > 0 {
		if allowedOrigins["*"] {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

func (s *SecurityMiddleware) getClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remoteIP); err == nil {
		remoteIP = host
	}

	if len(s.trustedProxies) == 0 {
		return remoteIP
	}

	if !s.isTrustedProxy(remoteIP) {
		return remoteIP
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		for i := len(ips) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(ips[i])
			if ip == "" {
				continue
			}
			if !s.isTrustedProxy(ip) {
				return ip
			}
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		xri = strings.TrimSpace(xri)
		if xri != "" && !s.isTrustedProxy(xri) {
			return xri
		}
	}

	return remoteIP
}

func (s *SecurityMiddleware) isTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range s.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
