# Architecture

## Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        MCP Clients                                   │
│         (Claude Desktop, Claude Code, Cursor, etc.)                 │
└─────────────────────────────────────────────────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    │  stdio or HTTP/SSE    │
                    └───────────┬───────────┘
                                │
┌─────────────────────────────────────────────────────────────────────┐
│                         main.go                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────┐ │
│  │ MCP Server  │  │ HTTP Server │  │ Security Middleware         │ │
│  │ (go-sdk)    │  │ (optional)  │  │ - Bearer Auth               │ │
│  └──────┬──────┘  └──────┬──────┘  │ - Rate Limiting             │ │
│         │                │         │ - CORS                      │ │
│         └────────┬───────┘         │ - Request Size Limits       │ │
│                  │                 └─────────────────────────────┘ │
└──────────────────┼─────────────────────────────────────────────────┘
                   │
┌──────────────────┼─────────────────────────────────────────────────┐
│                  │    tools/                                        │
│  ┌───────────────┴───────────────┐                                 │
│  │       HandlerRegistry         │                                 │
│  │  - Type-safe tool registration│                                 │
│  │  - Metrics & tracing wrapper  │                                 │
│  │  - Panic recovery             │                                 │
│  │  - Timeout enforcement        │                                 │
│  └───────────────┬───────────────┘                                 │
│                  │                                                  │
│  ┌───────────────┴───────────────┐                                 │
│  │         AllTools              │                                 │
│  │  Tool specifications          │                                 │
│  │  (name, description, hints)   │                                 │
│  └───────────────────────────────┘                                 │
└────────────────────────────────────────────────────────────────────┘
                   │
     ┌─────────────┼─────────────┬─────────────┐
     │             │             │             │
     ▼             ▼             ▼             ▼
┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐
│ Norway  │  │ Denmark │  │ Finland │  │    base     │
│ Client  │  │ Client  │  │ Client  │  │   Client    │
└────┬────┘  └────┬────┘  └────┬────┘  └──────┬──────┘
     │            │            │              │
     └────────────┴────────────┴──────────────┘
                         │
           ┌─────────────┼─────────────┐
           │             │             │
           ▼             ▼             ▼
      ┌─────────┐  ┌─────────┐  ┌─────────────┐
      │  Cache  │  │Circuit  │  │   Dedup     │
      │  (LRU)  │  │Breaker  │  │  Requests   │
      └─────────┘  └─────────┘  └─────────────┘
```

---

## Package Structure

```
nordic-registry-mcp-server/
├── main.go                     # Entry point, transports, HTTP server
├── internal/
│   ├── base/                   # Shared HTTP client infrastructure
│   │   └── client.go           # Retries, circuit breaker, rate limiting
│   ├── infra/                  # Shared infrastructure
│   │   ├── cache.go            # LRU cache with TTL
│   │   └── resilience.go       # Circuit breaker, request deduplication
│   ├── errors/                 # Shared error types
│   │   └── errors.go           # NotFoundError, ValidationError
│   ├── norway/                 # Norwegian registry client
│   │   ├── client.go           # API methods
│   │   ├── types.go            # Response types (HAL format)
│   │   ├── args.go             # MCP argument/result types
│   │   ├── mcp.go              # MCP wrappers (args → result)
│   │   └── validation.go       # Org number validation
│   ├── denmark/                # Danish registry client
│   │   └── ... (same structure)
│   └── finland/                # Finnish registry client
│       └── ... (same structure)
├── tools/
│   ├── registry.go             # ToolSpec type definition
│   ├── definitions.go          # AllTools list with metadata
│   └── handlers.go             # HandlerRegistry, registration logic
├── metrics/
│   └── metrics.go              # Prometheus metrics
├── tracing/
│   └── tracing.go              # OpenTelemetry setup
└── docs/                       # Documentation (you are here)
```

---

## Request Flow

### 1. Tool Invocation

```
MCP Client → MCP Server → HandlerRegistry.register()
```

The generic `register()` function wraps each tool handler with:

1. **Panic recovery** - Prevents crashes, logs stack trace
2. **Timeout** - 30 second maximum execution time
3. **Tracing** - OpenTelemetry span creation
4. **Metrics** - Request count, duration, in-flight gauge
5. **Logging** - Tool name, arguments, result summary

### 2. Country Client Execution

```
Handler → Country Client (norway/denmark/finland) → Base Client
```

Each country client:
1. Validates input (org number format)
2. Checks cache (returns cached if valid)
3. Checks request deduplication (coalesces identical concurrent requests)
4. Delegates to base client for HTTP

### 3. HTTP Request

```
Base Client → Circuit Breaker → Semaphore → HTTP Client → External API
```

The base client:
1. **Circuit breaker check** - Fails fast if API is down
2. **Semaphore acquisition** - Limits to 5 concurrent requests
3. **Request creation** - Adds headers (Accept, User-Agent)
4. **Retry loop** - Up to 3 attempts with exponential backoff
5. **Response handling** - Rate limit detection (429), server errors (5xx)

### 4. Response Processing

```
External API → Base Client → Country Client → MCP Wrapper → MCP Server
```

1. **Base client** - Returns raw JSON bytes + status code
2. **Country client** - Parses JSON, caches result, records circuit breaker success
3. **MCP wrapper** - Transforms API types to MCP result types
4. **Handler** - Logs execution, returns to MCP server

---

## Key Design Decisions

### Why Three Separate Clients?

Each Nordic country has a different API:

| Country | Base URL | Format | Auth | Quirks |
|---------|----------|--------|------|--------|
| Norway | data.brreg.no | HAL/JSON | None | Pagination, org numbers |
| Denmark | cvrapi.dk | JSON | None | Single result, CVR format |
| Finland | avoindata.prh.fi | JSON | None | Business ID format |

Separate clients allow:
- Country-specific validation
- Different caching strategies
- Independent circuit breakers
- Clear code organization

### Why Embed Base Client?

The base client provides:
- HTTP transport with optimized settings
- Circuit breaker integration
- Request deduplication
- Semaphore-based rate limiting

Embedding (not wrapping) allows country clients to directly access these facilities while adding country-specific logic.

### Why MCP Wrappers?

The MCP SDK expects specific argument and result types. The wrappers:
1. Accept MCP argument structs
2. Call the underlying API method
3. Transform responses to MCP result structs
4. Handle not-found as a valid result (not error)

This separation keeps API clients reusable outside MCP context.

### Why Generic Registration?

```go
func register[Args, Result any](
    h *HandlerRegistry,
    server *mcp.Server,
    tool *mcp.Tool,
    spec ToolSpec,
    method func(context.Context, Args) (Result, error),
)
```

This generic function:
- Provides type safety (compile-time argument/result checking)
- Centralizes cross-cutting concerns (metrics, tracing, logging)
- Eliminates boilerplate in individual handlers

The tradeoff is a large type switch in `HandlerRegistry.register()` that maps `any` to concrete types.

---

## Resilience Patterns

### Circuit Breaker

State machine with three states:

```
         success
    ┌────────────────┐
    │                │
    ▼                │
┌────────┐  5 failures  ┌────────┐
│ CLOSED │─────────────▶│  OPEN  │
└────────┘              └────┬───┘
    ▲                        │
    │     30s timeout        │
    │                        ▼
    │                   ┌──────────┐
    └───────────────────│HALF-OPEN │
         success        └──────────┘
                             │ failure
                             │
                             ▼
                        (back to OPEN)
```

Configuration:
- Failure threshold: 5 consecutive failures
- Reset timeout: 30 seconds
- Half-open max: 2 test requests

### Request Deduplication

When multiple goroutines request the same data:

```
Goroutine A ─┐
             ├──▶ Single API call ──▶ Result shared to all
Goroutine B ─┤
Goroutine C ─┘
```

Key mechanism:
1. First request creates an `inflightRequest` with a `done` channel
2. Subsequent requests find the inflight entry and wait on the channel
3. When complete, all waiters receive the same result
4. The entry is deleted, allowing future requests

### LRU Cache

Bounded cache with time-based and size-based eviction:

```
┌─────────────────────────────────────┐
│ sync.Map (concurrent-safe)          │
│  key → CacheEntry {                 │
│    Data       interface{}           │
│    ExpiresAt  time.Time             │
│    AccessedAt time.Time (for LRU)   │
│  }                                  │
└─────────────────────────────────────┘
```

Eviction triggers:
1. **TTL expiration** - Checked on access + background cleanup
2. **Size limit** - LRU eviction when > 1000 entries

---

## HTTP Mode Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        HTTP Server (main.go)                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  /                    ──▶  MCP Streamable HTTP Handler               │
│                            └──▶ SecurityMiddleware                   │
│                                 └──▶ Bearer Auth                     │
│                                 └──▶ Rate Limiting                   │
│                                 └──▶ CORS                            │
│                                                                      │
│  /health              ──▶  Always 200 (liveness)                    │
│  /ready               ──▶  503 if any circuit breaker open          │
│  /metrics             ──▶  Prometheus handler                        │
│  /tools               ──▶  Tool discovery (cached 1h)               │
│  /status              ──▶  Circuit breaker + dedup stats            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Security Middleware Flow

```
Request ──▶ Check body size (max 2MB)
        ──▶ Check rate limit (token bucket per IP)
        ──▶ Check origin (CORS allowlist)
        ──▶ Check auth (Bearer token comparison)
        ──▶ Add security headers
        ──▶ Pass to handler
```

---

## Metrics

Prometheus metrics with namespace `nordic_registry_mcp`:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `requests_total` | Counter | tool, success | Total tool invocations |
| `request_duration_seconds` | Histogram | tool | Request latency |
| `request_in_flight` | Gauge | tool | Currently executing requests |
| `panics_recovered_total` | Counter | tool | Panics caught by recovery |

---

## Tracing

OpenTelemetry spans created for each tool invocation:

```
mcp.tool.{tool_name}
├── Attributes:
│   ├── mcp.tool.name
│   ├── mcp.tool.category
│   ├── mcp.tool.country
│   ├── mcp.tool.readonly
│   └── mcp.tool.duration_seconds
└── Status: Ok or Error
```

Configure via environment:
- `OTEL_EXPORTER_OTLP_ENDPOINT` - Collector endpoint
- `OTEL_SERVICE_NAME` - Service name override
