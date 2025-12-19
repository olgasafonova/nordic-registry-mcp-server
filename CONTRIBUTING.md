# Contributing

Thanks for your interest in contributing to MediaWiki MCP Server!

## Quick Start

```bash
# Clone the repo
git clone https://github.com/olgasafonova/mediawiki-mcp-server.git
cd mediawiki-mcp-server

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o mediawiki-mcp-server .
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
| `wiki/read.go` | Page reading operations |
| `wiki/write.go` | Page editing operations |
| `wiki/search.go` | Search operations |
| `wiki/links.go` | Link operations |
| `wiki/history.go` | Revision history |
| `wiki/quality.go` | Content quality checks |
| `wiki/security.go` | SSRF protection |
| `wiki/types.go` | Type definitions |
| `wiki/errors.go` | Error handling |
| `main.go` | Tool registration |

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

### 5. Commit

Write clear commit messages:

```
Add fuzzy title matching to resolve_title

- Implement Jaccard similarity for title comparison
- Add suggestions when exact match not found
- Include similarity scores in response
```

### 6. Submit a Pull Request

- Push your branch
- Open a PR against `main`
- Fill out the PR template
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
return nil, fmt.Errorf("failed to get page '%s': %w", title, err)

// Better - use custom error types
return nil, &ValidationError{
    Field:      "title",
    Message:    "page not found",
    Suggestion: "Try using resolve_title to find similar pages",
}
```

### Adding New Tools

1. Add types to `wiki/types.go`:
   ```go
   type MyToolArgs struct {
       Title string `json:"title"`
   }

   type MyToolResult struct {
       Success bool   `json:"success"`
       Message string `json:"message"`
   }
   ```

2. Implement in the appropriate `wiki/*.go` file:
   ```go
   func (c *Client) MyTool(ctx context.Context, args MyToolArgs) (MyToolResult, error) {
       // Implementation
   }
   ```

3. Register in `main.go`:
   ```go
   registerToolHandler(server, &ToolDefinition{
       Name:        "mediawiki_my_tool",
       Description: "Does something useful",
       // ...
   }, func(ctx, req, args) { return client.MyTool(ctx, args) })
   ```

4. Add tests in `wiki/*_test.go`

## Testing Guidelines

### Unit Tests

Test individual functions:

```go
func TestMyFunction(t *testing.T) {
    result := myFunction("input")
    if result != "expected" {
        t.Errorf("got %s, want %s", result, "expected")
    }
}
```

### Table-Driven Tests

For multiple cases:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty", "", ""},
        {"normal", "hello", "HELLO"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := myFunction(tt.input)
            if got != tt.expected {
                t.Errorf("got %s, want %s", got, tt.expected)
            }
        })
    }
}
```

## Security

When contributing security-sensitive code:

- Review [SECURITY.md](SECURITY.md) for security practices
- Never log credentials or tokens
- Validate all user input
- Use SSRF protection for external requests
- Run `gosec ./...` to check for vulnerabilities

## Questions?

Open an issue or start a discussion on GitHub.
