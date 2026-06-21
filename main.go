// Nordic Registry MCP Server - A Model Context Protocol server for Nordic business registries
// Provides tools for searching and retrieving company information from Nordic countries
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/denmark"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/finland"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/norway"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/sweden"
	"github.com/olgasafonova/nordic-registry-mcp-server/tools"
	"github.com/olgasafonova/nordic-registry-mcp-server/tracing"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	ServerName    = "nordic-registry-mcp-server"
	ServerVersion = "1.0.0"
)

// cliFlags holds the parsed command-line flags.
type cliFlags struct {
	httpAddr       string
	bearerToken    string
	allowedOrigins string
	rateLimit      int
	trustedProxies string
}

// countryClients groups the per-country registry clients.
type countryClients struct {
	norway  *norway.Client
	denmark *denmark.Client
	finland *finland.Client
	sweden  *sweden.Client
}

// httpServerConfig groups everything runHTTPServer needs to stand up the
// HTTP transport, replacing an 11-argument signature.
type httpServerConfig struct {
	server    *mcp.Server
	logger    *slog.Logger
	flags     cliFlags
	authToken string
	clients   *countryClients
	registry  *tools.HandlerRegistry
}

func parseFlags() cliFlags {
	httpAddr := flag.String("http", "", "HTTP address to listen on (e.g., :8080). If empty, uses stdio transport.")
	bearerToken := flag.String("token", "", "Bearer token for HTTP authentication. Can also use MCP_AUTH_TOKEN env var.")
	allowedOrigins := flag.String("origins", "", "Comma-separated allowed origins for CORS.")
	rateLimit := flag.Int("rate-limit", 60, "Maximum requests per minute per IP (0 = unlimited)")
	trustedProxies := flag.String("trusted-proxies", "", "Comma-separated trusted proxy IPs/CIDRs.")
	flag.Parse()

	return cliFlags{
		httpAddr:       *httpAddr,
		bearerToken:    *bearerToken,
		allowedOrigins: *allowedOrigins,
		rateLimit:      *rateLimit,
		trustedProxies: *trustedProxies,
	}
}

// setupTracing initializes OpenTelemetry tracing and returns a shutdown
// function (nil when tracing is disabled or failed to initialize).
func setupTracing(logger *slog.Logger) func() {
	tracingConfig := tracing.DefaultConfig()
	tracingConfig.ServiceVersion = ServerVersion
	shutdownTracing, err := tracing.Setup(context.Background(), tracingConfig)
	if err != nil {
		logger.Warn("Failed to initialize tracing", "error", err)
		return nil
	}
	if !tracingConfig.Enabled {
		return nil
	}
	logger.Info("OpenTelemetry tracing enabled",
		"endpoint", tracingConfig.OTLPEndpoint,
		"service", tracingConfig.ServiceName)
	return func() { _ = shutdownTracing(context.Background()) }
}

// buildClients creates the per-country registry clients. The Sweden client
// is only created when OAuth2 credentials are configured.
func buildClients(logger *slog.Logger) *countryClients {
	clients := &countryClients{
		norway:  norway.NewClient(norway.WithLogger(logger)),
		denmark: denmark.NewClient(denmark.WithLogger(logger)),
		finland: finland.NewClient(finland.WithLogger(logger)),
	}

	if !sweden.IsConfigured() {
		logger.Info("Sweden client not configured (set BOLAGSVERKET_CLIENT_ID and BOLAGSVERKET_CLIENT_SECRET)")
		return clients
	}

	swedenClient, err := sweden.NewClient()
	if err != nil {
		logger.Warn("Failed to create Sweden client", "error", err)
		return clients
	}
	clients.sweden = swedenClient
	logger.Info("Sweden client initialized (OAuth2 credentials configured)")
	return clients
}

// close releases all configured clients.
func (c *countryClients) close() {
	c.norway.Close()
	c.denmark.Close()
	c.finland.Close()
	if c.sweden != nil {
		c.sweden.Close()
	}
}

// resolveAuthToken returns the bearer token from the flag, falling back to
// the MCP_AUTH_TOKEN environment variable.
func resolveAuthToken(flagToken string) string {
	if flagToken != "" {
		return flagToken
	}
	return os.Getenv("MCP_AUTH_TOKEN")
}

// buildServer creates the MCP server and registers all tools.
func buildServer(logger *slog.Logger, clients *countryClients) (*mcp.Server, *tools.HandlerRegistry) {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: ServerVersion,
	}, &mcp.ServerOptions{
		Logger: logger,
		// Disable listChanged notifications to prevent a pre-initialize
		// spec violation in go-sdk: when tools are registered before
		// server.Run(), the SDK sends notifications/tools/list_changed
		// before the client completes the initialize handshake. This
		// causes intermittent connection failures in Claude Code CLI
		// when many MCP servers start simultaneously. The client still
		// discovers tools via the tools/list request during handshake.
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{},
		},
		Instructions: serverInstructions,
	})

	registry := tools.NewHandlerRegistry(tools.HandlerRegistryConfig{
		NorwayClient:  clients.norway,
		DenmarkClient: clients.denmark,
		FinlandClient: clients.finland,
		SwedenClient:  clients.sweden,
		Logger:        logger,
	})
	registry.RegisterAll(server)
	return server, registry
}

// runStdioServer runs the MCP server over the stdio transport until a
// shutdown signal is received.
func runStdioServer(server *mcp.Server, logger *slog.Logger) {
	logger.Info("Starting Nordic Registry MCP Server (stdio mode)",
		"name", ServerName,
		"version", ServerVersion,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := <-sigChan
		logger.Info("Shutdown signal received", "signal", sig.String())
		cancel()
	}()

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && err != context.Canceled {
		log.Fatalf("Server error: %v", err)
	}

	logger.Info("Shutdown complete")
}

func main() {
	flags := parseFlags()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if shutdownTracing := setupTracing(logger); shutdownTracing != nil {
		defer shutdownTracing()
	}

	clients := buildClients(logger)
	defer clients.close()

	authToken := resolveAuthToken(flags.bearerToken)
	server, registry := buildServer(logger, clients)

	if flags.httpAddr != "" {
		runHTTPServer(httpServerConfig{
			server:    server,
			logger:    logger,
			flags:     flags,
			authToken: authToken,
			clients:   clients,
			registry:  registry,
		})
		return
	}

	runStdioServer(server, logger)
}

const serverInstructions = `Nordic Registry MCP Server - Access Nordic Business Registries

## Available Countries

Currently supports:
- **Norway** (Brønnøysundregistrene / data.brreg.no) - Norwegian business registry
- **Denmark** (CVR / cvrapi.dk) - Danish business registry
- **Finland** (PRH / avoindata.prh.fi) - Finnish business registry
- **Sweden** (Bolagsverket / api.bolagsverket.se) - Swedish business registry (requires OAuth2 credentials)

## Tool Selection Guide

### Search for companies by name:
"Find Norwegian companies named Equinor"
-> USE: norway_search_companies

### Get company details by org number:
"Get details for company 923609016"
-> USE: norway_get_company

### Get board members and roles:
"Who is on the board of 923609016?"
-> USE: norway_get_roles

### Get branch offices:
"What branches does company 923609016 have?"
-> USE: norway_get_subunits

### Get a specific sub-unit:
"Get details for sub-unit 912345678"
-> USE: norway_get_subunit

### Monitor registry changes:
"What companies changed since yesterday?"
-> USE: norway_get_updates

## Danish Company Lookups

### Search for Danish companies by name:
"Find Danish company Novo Nordisk"
-> USE: denmark_search_companies

### Get Danish company details by CVR:
"Get details for CVR 10150817"
-> USE: denmark_get_company

### Get production units (P-numbers):
"What production units does CVR 10150817 have?"
-> USE: denmark_get_production_units

### IMPORTANT: Danish Search Returns Only ONE Result

The CVR API returns only one company per search. Large companies often have multiple legal entities with similar names. When searching for well-known or international companies, TRY MULTIPLE VARIATIONS:

1. "[Company] Denmark" - Danish subsidiary (e.g., "Tietoevry Denmark")
2. "[Company] A/S" or "[Company] ApS" - with legal form
3. "[Company] DK" - common naming pattern
4. "[Company] Holding" - holding company vs operating company
5. Pre-merger/historical names - companies change names after M&A
6. "[Company] filial" - branch of foreign company

Example: Searching "Tietoevry" returns TIETOEVRY DK A/S (11 employees), but "Tietoevry Denmark" returns TIETOEVRY DENMARK A/S (56 employees) - a completely different legal entity.

Always ask the user to clarify if the first result seems wrong (wrong size, wrong address, wrong industry).

## Finnish Company Lookups

### Search for Finnish companies by name:
"Find Finnish company Nokia"
-> USE: finland_search_companies

### Get Finnish company details by business ID:
"Get details for business ID 0112038-9"
-> USE: finland_get_company

### IMPORTANT: Finnish Search Can Return 900+ Results

Common company names return too many results. To narrow down:

1. Use exact legal name: "Nokia Oyj" instead of "Nokia"
2. Filter by company_form: OY (private) or OYJ (public) for main operating companies
3. Filter by location: city name to narrow geographically
4. Combine filters: company_form=OY AND location=Helsinki

Example: Searching "Nokia" returns 900+ results. Searching "Nokia Oyj" with company_form=OYJ returns just the main company.

## Swedish Company Lookups

Sweden has NO name search in this API - you must have the 10-digit organization number. Ask the user for the org number if not provided.

### Get Swedish company details:
"Get Swedish company 5560125790"
-> USE: sweden_get_company

## Norwegian Organization Numbers

Norwegian org numbers are 9 digits. Spaces and dashes are automatically removed.
Examples: "923609016", "923 609 016", "923-609-016" all work.

## Danish CVR Numbers

Danish CVR numbers are 8 digits. Spaces, dashes, and "DK" prefix are automatically removed.
Examples: "10150817", "DK-10150817", "DK10150817" all work.

## Organization Forms (Norway)

Common codes:
- AS: Aksjeselskap (Limited company)
- ASA: Allmennaksjeselskap (Public limited company)
- ENK: Enkeltpersonforetak (Sole proprietorship)
- NUF: Norsk avdeling av utenlandsk foretak (Norwegian branch of foreign company)
- ANS: Ansvarlig selskap (General partnership)
- DA: Delt ansvar (Limited partnership)
- SA: Samvirkeforetak (Cooperative)
- STI: Stiftelse (Foundation)

## Company Types (Denmark)

Common types:
- A/S: Aktieselskab (Public limited company)
- ApS: Anpartsselskab (Private limited company)
- I/S: Interessentskab (General partnership)
- K/S: Kommanditselskab (Limited partnership)
- P/S: Partnerselskab (Partnership company)
- IVS: Iværksætterselskab (Entrepreneurial company)
- Enkeltmandsvirksomhed (Sole proprietorship)

## Finnish Business IDs (Y-tunnus)

Finnish business IDs are 7 digits + hyphen + check digit (e.g., 0112038-9).
The FI prefix is automatically removed. Examples: "0112038-9", "FI0112038-9" both work.

## Company Forms (Finland)

Common codes:
- OY: Osakeyhtiö (Private limited company)
- OYJ: Julkinen osakeyhtiö (Public limited company)
- Ky: Kommandiittiyhtiö (Limited partnership)
- Ay: Avoin yhtiö (General partnership)
- Tmi: Toiminimi (Sole proprietorship)
- Osk: Osuuskunta (Cooperative)`

// parseCSVList splits a comma-separated string into a slice, trimming
// whitespace and dropping empty entries.
func parseCSVList(s string) []string {
	if s == "" {
		return nil
	}
	var list []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			list = append(list, item)
		}
	}
	return list
}

// registerHTTPRoutes wires the operational endpoints (health, ready, metrics,
// tools, status) plus the secured MCP handler onto a new mux.
func registerHTTPRoutes(cfg httpServerConfig, securedHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler(cfg.clients))
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/tools", toolsHandler(cfg.logger, cfg.registry))
	mux.HandleFunc("/status", statusHandler(cfg.logger, cfg.clients))
	mux.Handle("/", securedHandler)
	return mux
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `{"status":"healthy","server":"%s","version":"%s"}`, ServerName, ServerVersion)
}

func readyHandler(clients *countryClients) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")

		noCBStats := clients.norway.CircuitBreakerStats()
		dkCBStats := clients.denmark.CircuitBreakerStats()
		fiCBStats := clients.finland.CircuitBreakerStats()

		if !allClosed(noCBStats.State, dkCBStats.State, fiCBStats.State) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"not_ready","norway_cb":"%s","denmark_cb":"%s","finland_cb":"%s"}`, noCBStats.State, dkCBStats.State, fiCBStats.State)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ready","countries":["norway","denmark","finland"]}`)
	}
}

// allClosed reports whether every circuit breaker state is "closed".
func allClosed(states ...string) bool {
	for _, s := range states {
		if s != "closed" {
			return false
		}
	}
	return true
}

func toolsHandler(logger *slog.Logger, registry *tools.HandlerRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")

		registeredTools := registry.RegisteredTools()
		toolsByCountry := make(map[string][]map[string]any)
		for _, tool := range registeredTools {
			toolInfo := map[string]any{
				"name":        tool.Name,
				"title":       tool.Title,
				"category":    tool.Category,
				"description": tool.Description,
				"read_only":   tool.ReadOnly,
			}
			toolsByCountry[tool.Country] = append(toolsByCountry[tool.Country], toolInfo)
		}

		response := map[string]any{
			"server":     ServerName,
			"version":    ServerVersion,
			"tool_count": len(registeredTools),
			"countries":  toolsByCountry,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("Failed to encode tools response", "error", err)
		}
	}
}

func statusHandler(logger *slog.Logger, clients *countryClients) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")

		noCBStats := clients.norway.CircuitBreakerStats()
		noDedupStats := clients.norway.DedupStats()
		dkCBStats := clients.denmark.CircuitBreakerStats()
		dkDedupStats := clients.denmark.DedupStats()
		fiCBStats := clients.finland.CircuitBreakerStats()

		response := map[string]any{
			"server":  ServerName,
			"version": ServerVersion,
			"norway": map[string]any{
				"circuit_breaker": map[string]any{
					"state":                noCBStats.State,
					"consecutive_failures": noCBStats.ConsecutiveFails,
					"last_failure":         noCBStats.LastFailure,
				},
				"dedup": map[string]any{
					"inflight_requests": noDedupStats,
				},
			},
			"denmark": map[string]any{
				"circuit_breaker": map[string]any{
					"state":                dkCBStats.State,
					"consecutive_failures": dkCBStats.ConsecutiveFails,
					"last_failure":         dkCBStats.LastFailure,
				},
				"dedup": map[string]any{
					"inflight_requests": dkDedupStats,
				},
			},
			"finland": map[string]any{
				"circuit_breaker": map[string]any{
					"state":                fiCBStats.State,
					"consecutive_failures": fiCBStats.ConsecutiveFails,
					"last_failure":         fiCBStats.LastFailure,
				},
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("Failed to encode status response", "error", err)
		}
	}
}

// serveHTTP starts the HTTP server and blocks until a shutdown signal or a
// fatal server error, then performs a graceful shutdown.
func serveHTTP(httpServer *http.Server, securedHandler *SecurityMiddleware, logger *slog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("HTTP server error: %v", err)
	case sig := <-sigChan:
		logger.Info("Shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("Initiating graceful shutdown...")

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	if securedHandler.rateLimiter != nil {
		securedHandler.rateLimiter.Close()
		logger.Info("Rate limiter stopped")
	}

	logger.Info("Shutdown complete")
}

func runHTTPServer(cfg httpServerConfig) {
	logger := cfg.logger
	addr := cfg.flags.httpAddr
	authToken := cfg.authToken

	mcpHandler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return cfg.server
	}, nil)

	securityConfig := SecurityConfig{
		BearerToken:    authToken,
		AllowedOrigins: parseCSVList(cfg.flags.allowedOrigins),
		RateLimit:      cfg.flags.rateLimit,
		TrustedProxies: parseCSVList(cfg.flags.trustedProxies),
	}
	securedHandler := NewSecurityMiddleware(mcpHandler, logger, securityConfig)

	mux := registerHTTPRoutes(cfg, securedHandler)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("Starting Nordic Registry MCP Server (HTTP mode)",
		"name", ServerName,
		"version", ServerVersion,
		"address", addr,
		"auth_enabled", authToken != "",
		"rate_limit", cfg.flags.rateLimit,
	)

	if authToken == "" {
		logger.Warn("HTTP server running WITHOUT authentication. Set -token flag or MCP_AUTH_TOKEN env var for production use.")
	}
	if !strings.HasPrefix(addr, "127.0.0.1") && !strings.HasPrefix(addr, "localhost") {
		logger.Warn("Server binding to external interface. Ensure you're behind HTTPS proxy in production.")
	}

	serveHTTP(httpServer, securedHandler, logger)
}
