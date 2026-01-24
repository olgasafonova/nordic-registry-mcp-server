# Nordic Registry MCP Server

Query Nordic business registries with AI. Search companies, get details, find board members, and access annual reports across Norway, Denmark, Finland, and Sweden.

[![CI](https://github.com/olgasafonova/nordic-registry-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/olgasafonova/nordic-registry-mcp-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/olgasafonova/nordic-registry-mcp-server)](https://goreportcard.com/report/github.com/olgasafonova/nordic-registry-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Works with:** Claude Desktop, Claude Code, Cursor, and any MCP-compatible tool.

---

## Supported Countries

| Country | Registry | Tools | ID Format |
|---------|----------|-------|-----------|
| Norway | [Brønnøysundregistrene](https://data.brreg.no) | 12 | 9 digits (e.g., `923609016`) |
| Denmark | [CVR](https://datacvr.virk.dk) | 5 | 8 digits (e.g., `10150817`) |
| Finland | [PRH](https://avoindata.prh.fi) | 2 | 7+1 digits (e.g., `0112038-9`) |
| Sweden | [Bolagsverket](https://bolagsverket.se) | 3 | 10 digits (e.g., `5560125790`) |

All APIs are free. Norway, Denmark, and Finland require no authentication. Sweden uses the [värdefulla datamängder](https://bolagsverket.se/apierochoppnadata/hamtaforetagsinformation/vardefulladatamangder.5294.html) API which requires OAuth2 credentials ([free registration](https://bolagsverket.se/apierochoppnadata/vardefulladatamangder/kundanmalantillapiforvardefulladatamangder.5528.html)).

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
| *"Get Swedish company 5560125790"* | Returns company info from Bolagsverket |
| *"List annual reports for Swedish company X"* | Lists available årsredovisningar |

---

## Use Cases

### Case Processing: Verify Applicant Companies

When processing permit applications, license renewals, or grant requests, verify the applying company exists and is in good standing.

```
You: "A company applied for an environmental permit. Org number 923609016.
      Check if they're active and who can sign on their behalf."

AI: Let me verify the company and check signing authority.
→ norway_get_company("923609016")
→ norway_get_signature_rights("923609016")

Result: Equinor ASA - Active since 1972, registered for VAT, 21,200 employees.
Authorized signatories: [Name 1] (CEO), [Name 2] + [Name 3] (joint signature).
The application signature matches an authorized signatory.
```

### Procurement: Validate Vendors Before Contract Award

Before awarding a municipal contract, verify that bidding companies are legitimate and not bankrupt.

```
You: "We received bids from three Danish companies for the IT contract.
      CVR numbers: 10150817, 25063783, 37553472. Are they all legitimate?"

AI: I'll verify each company's status.
→ denmark_get_company("10150817")
→ denmark_get_company("25063783")
→ denmark_get_company("37553472")

Result:
- 10150817: Novo Nordisk A/S - Active, 47,000+ employees ✓
- 25063783: Mærsk A/S - Active, established company ✓
- 37553472: [Company name] - BANKRUPT ✗

Recommendation: Exclude 37553472 from consideration.
```

### Grant Administration: Verify Non-Profit Status

When processing grant applications from voluntary organizations, verify their registration in Frivillighetsregisteret.

```
You: "An organization applied for cultural funding.
      They claim to be a registered non-profit. Org number 971274808."

AI: Let me check their voluntary organization status.
→ norway_get_company("971274808")

Result: Norges Røde Kors - Registered in Frivillighetsregisteret since 2009.
Organization form: FLI (Forening/lag/innretning).
Activity: Humanitarian aid and emergency response.
Confirmed as eligible non-profit organization.
```

### Regulatory Oversight: Monitor Industry Changes

Track new company registrations or status changes in regulated sectors.

```
You: "I need to monitor new companies in the financial sector.
      What relevant registrations happened this week?"

AI: Let me check recent registry updates.
→ norway_get_updates(since="2024-01-08T00:00:00Z", size=100)

Result: 12 new registrations in financial services sector:
- 3 new AS companies (aksjeselskap)
- 2 new NUF branches of foreign financial institutions
- 7 updates to existing financial companies
[List with org numbers for follow-up]
```

### Cross-Border Coordination: Track Multinational Presence

When coordinating with other Nordic authorities, identify a company's presence across jurisdictions.

```
You: "We're investigating a company that may operate across Nordic countries.
      Check if Volvo has registered entities in Norway and Denmark."

AI: Let me search both registries.
→ norway_search_companies("Volvo")
→ denmark_search_companies("Volvo")

Result: Found registered entities:
- Norway: Volvo Norge AS (org 923609016) - 450 employees, automotive
- Denmark: Volvo Danmark A/S (CVR 12345678) - 320 employees
Both are subsidiaries of Swedish parent company.
```

### Data Quality: Validate Registry Submissions

When receiving bulk data submissions referencing company identifiers, validate them against official registries.

```
You: "A ministry submitted a list of 50 org numbers for a new subsidy program.
      Validate which ones are current and active."

AI: I'll validate them against Brønnøysundregistrene.
→ norway_batch_get_companies([list of 50 org numbers])

Result:
- 47 companies: Active and valid ✓
- 2 companies: Marked as bankrupt - may need review
- 1 org number: Not found in registry - possibly incorrect

Flagged entries exported for manual review.
```

### Statistics: Company Demographics by Municipality

Support policy analysis by examining business composition in specific areas.

```
You: "How many AS companies are registered in Oslo municipality?"

AI: Let me search with municipality filter.
→ norway_search_companies(query="*", org_form="AS", municipality="0301")

Result: Found 45,231 AS companies registered in Oslo (municipality 0301).
Paginated results available for detailed analysis.
```

### Due Diligence: Verify Board Composition

Before approving state aid or contracts above threshold, verify company governance.

```
You: "Check the board composition of company 914778271.
      We need this for the state aid application review."

AI: Let me retrieve their roles.
→ norway_get_company("914778271")
→ norway_get_roles("914778271")

Result: [Company name] - AS registered 2015
Board members:
- [Name 1] - Styreleder (Chair)
- [Name 2] - Styremedlem
- [Name 3] - Styremedlem
- [Name 4] - Daglig leder (CEO)
Auditor: [Audit firm name]
```

### Document Retrieval: Access Swedish Annual Reports

When reviewing Swedish companies for cross-border matters, check available financial documentation.

```
You: "A Swedish company applied for recognition in Norway.
      What financial reports are available for org 5560125790?"

AI: Let me check Bolagsverket for filed documents.
→ sweden_get_company("5560125790")
→ sweden_get_document_list("5560125790")

Result: [Company name] - Active Swedish AB
Available årsredovisningar: 8 annual reports (2016-2023)
Most recent: Financial year ending 2023-06-30, filed 2023-08-15.
```

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
| `norway_get_signature_rights` | Get signature rights and prokura |
| `norway_batch_get_companies` | Look up multiple companies at once |
| `norway_get_subunits` | List branch offices for a company |
| `norway_get_subunit` | Get specific branch office details |
| `norway_search_subunits` | Search branch offices by name |
| `norway_get_updates` | Get recent registry changes |
| `norway_get_subunit_updates` | Get recent branch office changes |
| `norway_list_municipalities` | List municipality codes |
| `norway_list_org_forms` | List organization form codes (AS, ENK, etc.) |

### Denmark (CVR)

| Tool | Description |
|------|-------------|
| `denmark_search_companies` | Search companies by name |
| `denmark_get_company` | Get company details by CVR number |
| `denmark_get_production_units` | List production units (P-numbers) |
| `denmark_search_by_phone` | Find company by phone number |
| `denmark_get_by_pnumber` | Get company by P-number |

### Finland (PRH)

| Tool | Description |
|------|-------------|
| `finland_search_companies` | Search companies by name |
| `finland_get_company` | Get company details by business ID |

### Sweden (Bolagsverket)

Requires OAuth2 credentials. Set `BOLAGSVERKET_CLIENT_ID` and `BOLAGSVERKET_CLIENT_SECRET` environment variables.

| Tool | Description |
|------|-------------|
| `sweden_get_company` | Get company details by organization number |
| `sweden_get_document_list` | List annual reports (årsredovisningar) |
| `sweden_check_status` | Check API availability and OAuth2 status |

---

## Example Prompts

### Company Search
- *"Find Norwegian companies named Telenor"*
- *"Search for AS companies in Oslo"*
- *"Find Danish companies named Carlsberg"*
- *"Look up Finnish company Kone"*
- *"Find voluntary organizations named Røde Kors"*

### Company Details
- *"Get details for Norwegian org 923609016"*
- *"Look up CVR 10150817"*
- *"Get Finnish company 0112038-9"*

### Board & Roles (Norway only)
- *"Who is on the board of 923609016?"*
- *"Find the CEO of Equinor"*
- *"List all directors for org 914778271"*

### Signature Rights (Norway only)
- *"Who can sign for company 923609016?"*
- *"Get signature rights for Equinor"*
- *"Who has prokura for 914778271?"*

### Branch Offices
- *"What branches does 923609016 have?"*
- *"Search for branch offices named Equinor"*
- *"List production units for CVR 10150817"*

### Registry Updates (Norway only)
- *"What companies changed since yesterday?"*
- *"Get recent registry updates"*
- *"What branch offices changed recently?"*

### Batch Operations (Norway only)
- *"Look up these companies: 923609016, 914778271, 985399077"*
- *"Validate these org numbers from my spreadsheet"*
- *"Get details for multiple Norwegian companies at once"*

### Reference Data (Norway only)
- *"List all Norwegian municipalities"*
- *"What is Oslo's municipality code?"*
- *"What does AS mean?"*
- *"List organization form codes"*

### Phone & P-Number Lookup (Denmark)
- *"Find company with phone 33121212"*
- *"Look up production unit P-number 1234567890"*

### Sweden Lookups
- *"Get Swedish company 5560125790"*
- *"Look up Volvo's organization number in Sweden"*
- *"What annual reports are available for 5560125790?"*
- *"List årsredovisningar for Swedish company X"*
- *"Is the Swedish API working?"*

---

## Sweden Setup

Sweden's Bolagsverket API requires OAuth2 authentication (free).

1. [Register for the värdefulla datamängder API](https://bolagsverket.se/apierochoppnadata/vardefulladatamangder/kundanmalantillapiforvardefulladatamangder.5528.html) (submit customer registration form)
2. Access the [Developer Portal](https://portal.api.bolagsverket.se/devportal/) to get your OAuth2 credentials
3. Set environment variables:
   ```bash
   export BOLAGSVERKET_CLIENT_ID="your-client-id"
   export BOLAGSVERKET_CLIENT_SECRET="your-client-secret"
   ```

The server will log whether Sweden credentials are configured on startup. If not configured, Sweden tools are simply not registered.

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
│   ├── finland/           # Finnish registry client
│   └── sweden/            # Swedish registry client (OAuth2)
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

## Documentation

| Document | Description |
|----------|-------------|
| [API Reference](docs/API.md) | Complete reference for all 22 tools with parameters, return values, and examples |
| [Architecture](docs/ARCHITECTURE.md) | System design, request flow, resilience patterns |
| [Production Readiness](docs/PRODUCTION.md) | Deployment checklist, monitoring, known limitations |

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
- Data from [Brønnøysundregistrene](https://data.brreg.no), [CVR](https://datacvr.virk.dk), [PRH](https://avoindata.prh.fi), [Bolagsverket](https://bolagsverket.se)
