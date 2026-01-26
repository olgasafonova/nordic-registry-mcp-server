# Setup Guide

Get the Nordic Registry MCP Server running in under 5 minutes.

## Prerequisites

- Go 1.21+ (only if building from source)
- Claude Desktop, Claude Code, Cursor, or any MCP-compatible client

## Installation

### Option 1: Download Binary (Recommended)

Download the latest release for your platform from [Releases](https://github.com/olgasafonova/nordic-registry-mcp-server/releases).

```bash
# macOS (Apple Silicon)
curl -L -o nordic-registry-mcp-server \
  https://github.com/olgasafonova/nordic-registry-mcp-server/releases/latest/download/nordic-registry-mcp-server-darwin-arm64
chmod +x nordic-registry-mcp-server

# macOS (Intel)
curl -L -o nordic-registry-mcp-server \
  https://github.com/olgasafonova/nordic-registry-mcp-server/releases/latest/download/nordic-registry-mcp-server-darwin-amd64
chmod +x nordic-registry-mcp-server

# Linux (x64)
curl -L -o nordic-registry-mcp-server \
  https://github.com/olgasafonova/nordic-registry-mcp-server/releases/latest/download/nordic-registry-mcp-server-linux-amd64
chmod +x nordic-registry-mcp-server

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/olgasafonova/nordic-registry-mcp-server/releases/latest/download/nordic-registry-mcp-server-windows-amd64.exe" -OutFile "nordic-registry-mcp-server.exe"
```

### Option 2: Build from Source

```bash
git clone https://github.com/olgasafonova/nordic-registry-mcp-server.git
cd nordic-registry-mcp-server
go build .
```

## Configuration

### Claude Code CLI

```bash
claude mcp add nordic-registry /path/to/nordic-registry-mcp-server
```

### Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "nordic-registry": {
      "command": "/path/to/nordic-registry-mcp-server"
    }
  }
}
```

Restart Claude Desktop after saving.

### Cursor

Add to your MCP configuration in Cursor settings.

## Sweden Setup (Optional)

Sweden's Bolagsverket API requires OAuth2 credentials. This is free but requires registration.

1. Go to [Bolagsverket API Registration](https://bolagsverket.se/apierochoppnadata/vardefulladatamangder/kundanmalantillapiforvardefulladatamangder.5528.html)
2. Submit the customer registration form (kundanmälan)
3. Wait for approval (usually 1-2 business days)
4. Access the [Developer Portal](https://portal.api.bolagsverket.se/devportal/) to get credentials

Set environment variables:

```bash
export BOLAGSVERKET_CLIENT_ID="your-client-id"
export BOLAGSVERKET_CLIENT_SECRET="your-client-secret"
```

Or in Claude Desktop config:

```json
{
  "mcpServers": {
    "nordic-registry": {
      "command": "/path/to/nordic-registry-mcp-server",
      "env": {
        "BOLAGSVERKET_CLIENT_ID": "your-client-id",
        "BOLAGSVERKET_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

Without Sweden credentials, all other countries (Norway, Denmark, Finland) work normally. Sweden tools simply won't appear.

## HTTP Mode (Remote Access)

For server deployments or multi-client access:

```bash
# Basic
./nordic-registry-mcp-server -http :8080

# With authentication (recommended)
./nordic-registry-mcp-server -http :8080 -token "your-secret-token"

# Production setup
./nordic-registry-mcp-server -http :8080 \
  -token "$MCP_AUTH_TOKEN" \
  -origins "https://your-app.com" \
  -rate-limit 60 \
  -trusted-proxies "10.0.0.0/8"
```

### HTTP Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-http` | HTTP listen address | (stdio mode) |
| `-token` | Bearer token for auth | (none) |
| `-origins` | Allowed CORS origins | (all) |
| `-rate-limit` | Requests/minute per IP | 60 |
| `-trusted-proxies` | CIDR ranges to trust X-Forwarded-For | (none) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `MCP_AUTH_TOKEN` | Bearer token (alternative to `-token` flag) |
| `BOLAGSVERKET_CLIENT_ID` | Sweden OAuth2 client ID |
| `BOLAGSVERKET_CLIENT_SECRET` | Sweden OAuth2 client secret |

## Verify Installation

After configuration, test in your AI client:

```
You: Find Norwegian companies named Equinor
```

Expected: The AI should return search results from Brønnøysundregistrene.

## Troubleshooting

### "Tool not found" errors

- Restart your MCP client after configuration changes
- Check that the binary path is correct and executable
- Run the binary directly to check for startup errors

### Sweden tools not appearing

- Verify both `BOLAGSVERKET_CLIENT_ID` and `BOLAGSVERKET_CLIENT_SECRET` are set
- Check server logs for OAuth2 errors
- Use `sweden_check_status` tool to verify API connectivity

### Connection refused (HTTP mode)

- Check firewall rules
- Verify the port is not in use
- Check `-http` flag format (e.g., `:8080` not `8080`)

### Rate limiting

- Default is 60 requests/minute per IP
- Increase with `-rate-limit` flag if needed
- Check `/status` endpoint for circuit breaker state

## Next Steps

- [API Reference](API.md) - Complete tool documentation
- [Architecture](ARCHITECTURE.md) - System design details
- [Production Guide](PRODUCTION.md) - Deployment checklist
