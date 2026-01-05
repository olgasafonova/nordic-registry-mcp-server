# Nordic Registry MCP Server

Query Nordic business registries with AI. Search companies, get details, find board members across Norway, Denmark, and Finland.

[![CI](https://github.com/olgasafonova/nordic-registry-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/olgasafonova/nordic-registry-mcp-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/olgasafonova/nordic-registry-mcp-server)](https://goreportcard.com/report/github.com/olgasafonova/nordic-registry-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Works with:** Claude Desktop, Claude Code, Cursor, and any MCP-compatible tool.

---

## Supported Countries

| Country | Registry | Tools | ID Format |
|---------|----------|-------|-----------|
| Norway | [Brønnøysundregistrene](https://data.brreg.no) | 6 | 9 digits (e.g., `923609016`) |
| Denmark | [CVR](https://datacvr.virk.dk) | 3 | 8 digits (e.g., `10150817`) |
| Finland | [PRH](https://avoindata.prh.fi) | 2 | 7+1 digits (e.g., `0112038-9`) |

All APIs are free and require no authentication.

---

## What Can You Do?

Once connected, just ask your AI:

| You say... | What happens |
|------------|--------------|
| *"Find Norwegian companies named Equinor"* | Searches Brønnøysundregistrene |
| *"Get details for org number 923609016"* | Returns full company info |
| *"Who is on the board of 923609016?"* | Lists board members, CEO, roles |
| *"Find Danish company Novo Nordisk"* | Searches CVR registry |
| *"Look up Finnish company Nokia"* | Searches PRH registry |
| *"Get company 0112038-9 from Finland"* | Returns Nokia's full details |

---

## Quick Start

### Option 1: Download Binary

Go to [Releases](https://github.com/olgasafonova/nordic-registry-mcp-server/releases) and download for your platform.

### Option 2: Build from Source

```bash
git clone https://github.com/olgasafonova/nordic-registry-mcp-server.git
cd nordic-registry-mcp-server
go build .
```

Requires Go 1.21+

---

## Setup

### Claude Code CLI

```bash
claude mcp add nordic-registry ./nordic-registry-mcp-server
```

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nordic-registry": {
      "command": "/path/to/nordic-registry-mcp-server"
    }
  }
}
```

Restart Claude Desktop after changes.

---

## All Tools

### Norway (Brønnøysundregistrene)

| Tool | Description |
|------|-------------|
| `norway_search_companies` | Search companies by name |
| `norway_get_company` | Get company details by org number |
| `norway_get_roles` | Get board members, CEO, auditors |
| `norway_get_subunits` | List branch offices |
| `norway_get_subunit` | Get specific branch office details |
| `norway_get_updates` | Get recent registry changes |

### Denmark (CVR)

| Tool | Description |
|------|-------------|
| `denmark_search_companies` | Search companies by name |
| `denmark_get_company` | Get company details by CVR number |
| `denmark_get_production_units` | List production units (P-numbers) |

### Finland (PRH)

| Tool | Description |
|------|-------------|
| `finland_search_companies` | Search companies by name |
| `finland_get_company` | Get company details by business ID |

---

## Example Prompts

### Company Search
- *"Find Norwegian companies named Telenor"*
- *"Search for AS companies in Oslo"*
- *"Find Danish companies named Carlsberg"*
- *"Look up Finnish company Kone"*

### Company Details
- *"Get details for Norwegian org 923609016"*
- *"Look up CVR 10150817"*
- *"Get Finnish company 0112038-9"*

### Board & Roles (Norway only)
- *"Who is on the board of 923609016?"*
- *"Find the CEO of Equinor"*
- *"List all directors for org 914778271"*

### Branch Offices
- *"What branches does 923609016 have?"*
- *"List production units for CVR 10150817"*

### Registry Updates (Norway only)
- *"What companies changed since yesterday?"*
- *"Get recent registry updates"*

---

## HTTP Mode

For remote access or integration with other tools:

```bash
# Start HTTP server
./nordic-registry-mcp-server -http :8080

# With authentication
./nordic-registry-mcp-server -http :8080 -token "your-secret-token"
```

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `/` | MCP protocol (Streamable HTTP) |
| `/health` | Liveness check |
| `/ready` | Readiness check (verifies API connectivity) |
| `/tools` | List all tools by country |
| `/status` | Circuit breaker stats |
| `/metrics` | Prometheus metrics |

---

## Architecture

```
nordic-registry-mcp-server/
├── main.go                 # Entry point, HTTP/stdio transport
├── internal/
│   ├── infra/             # Shared infrastructure
│   │   ├── cache.go       # LRU cache with TTL
│   │   └── resilience.go  # Circuit breaker, request deduplication
│   ├── norway/            # Norwegian registry client
│   ├── denmark/           # Danish registry client
│   └── finland/           # Finnish registry client
├── tools/
│   ├── definitions.go     # Tool specifications
│   └── handlers.go        # MCP tool registration
├── metrics/               # Prometheus metrics
└── tracing/               # OpenTelemetry tracing
```

---

## Resilience Features

- **LRU Cache**: 5-15 minute TTL depending on endpoint
- **Circuit Breaker**: Automatic failover after consecutive failures
- **Request Deduplication**: Prevents duplicate concurrent requests
- **Rate Limiting**: Semaphore-based concurrency control
- **Retry with Backoff**: Exponential backoff on transient failures

---

## Why Not Sweden?

Sweden's Bolagsverket doesn't offer a free public API like the other Nordic countries. Options exist (Eniro API, OpenCorporates) but require payment or have restrictions. We may add Sweden if Bolagsverket releases a proper open data API.

---

## Development

```bash
# Build
go build .

# Test
go test ./...

# Lint (requires golangci-lint)
golangci-lint run
```

---

## License

MIT License

## Credits

- Built with [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
- Data from [Brønnøysundregistrene](https://data.brreg.no), [CVR](https://datacvr.virk.dk), [PRH](https://avoindata.prh.fi)
