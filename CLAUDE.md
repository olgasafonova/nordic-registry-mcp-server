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


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
