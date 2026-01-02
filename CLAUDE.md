# CLAUDE.md

Import @README.md

## Build & Test
```bash
go build .              # Build binary
go test ./...           # Run tests
```

## Project Structure
```
internal/
├── infra/              # Infrastructure (circuit breaker, cache, dedup)
│   ├── resilience.go   # CircuitBreaker, RequestDeduplicator
│   └── cache.go        # LRU cache with TTL
└── norway/             # Norway registry client (Brønnøysundregistrene)
    ├── client.go       # API client with resilience
    ├── types.go        # Response types (HAL format)
    ├── args.go         # MCP argument types
    └── mcp.go          # MCP result wrappers

tools/                  # MCP tool definitions
├── registry.go         # ToolSpec type
├── definitions.go      # Tool metadata (AllTools)
└── handlers.go         # Tool registration

metrics/                # Prometheus metrics (namespace: nordic_registry_mcp)
tracing/                # OpenTelemetry tracing
```

## Adding a New Country
1. Create `internal/{country}/` with client, types, args, mcp files
2. Add tool specs to `tools/definitions.go` (follow Norway pattern)
3. Register handlers in `tools/handlers.go`
4. Update README.md supported countries table

## Norway API
- Base URL: `https://data.brreg.no/enhetsregisteret/api`
- No authentication required
- HAL-formatted JSON responses
- Org numbers are 9 digits (spaces/dashes auto-stripped)
