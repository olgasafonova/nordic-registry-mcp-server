# Contributing

Thanks for your interest in contributing to Nordic Registry MCP Server!

## Quick Start

```bash
# Clone the repo
git clone https://github.com/olgasafonova/nordic-registry-mcp-server.git
cd nordic-registry-mcp-server

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o nordic-registry-mcp-server .
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Make Changes

Follow the code organization:

| Directory | Purpose |
|-----------|---------|
| `internal/norway/` | Norway (Brønnøysundregistrene) client |
| `internal/denmark/` | Denmark (CVR) client |
| `internal/finland/` | Finland (PRH) client |
| `internal/infra/` | Shared infrastructure (cache, resilience) |
| `tools/` | MCP tool definitions and handlers |
| `metrics/` | Prometheus metrics |
| `tracing/` | OpenTelemetry tracing |
| `main.go` | Server setup and configuration |

### 3. Test Your Changes

```bash
# Run all tests
go test ./...

# Run with race detection
go test ./... -race

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 4. Lint Your Code

```bash
# Install golangci-lint if needed
brew install golangci-lint

# Run linter
golangci-lint run
```

### 5. Submit a Pull Request

- Push your branch
- Open a PR against `main`
- Wait for CI checks to pass

## Code Style

### Go Conventions

- Use `gofmt` formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Add godoc comments to exported functions

### Error Handling

Return structured errors with context:

```go
// Good
return nil, fmt.Errorf("failed to get company '%s': %w", orgNumber, err)
```

### Adding New Tools

1. Add types to `internal/{country}/args.go` and `internal/{country}/types.go`

2. Implement in `internal/{country}/client.go`

3. Register in `tools/definitions.go` and `tools/handlers.go`

4. Add tests

### Adding a New Country

To add support for a new Nordic country (e.g., Sweden, Iceland):

1. Create `internal/{country}/` directory with:
   - `client.go` - API client
   - `args.go` - Tool argument structs
   - `types.go` - Response types

2. Follow existing patterns from Norway/Denmark/Finland clients

3. Add tool definitions and handlers

4. Update README.md with new tools

## Testing Guidelines

### Table-Driven Tests

```go
func TestValidateOrgNumber(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid 9 digits", "923609016", false},
        {"too short", "12345678", true},
        {"contains letters", "92360901A", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateOrgNumber(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Questions?

Open an issue or start a discussion on GitHub.
