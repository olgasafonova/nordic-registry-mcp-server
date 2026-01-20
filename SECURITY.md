# Security Guide for Nordic Registry MCP Server

This document outlines the security architecture and considerations for the Nordic Registry MCP Server.

## Security Profile

**This is a read-only server** that queries public business registry APIs. It does not:
- Perform write operations
- Handle user authentication
- Store sensitive data
- Process user credentials

The APIs queried (Brønnøysundregistrene, CVR, PRH) are public and require no authentication.

## Current Security Protections

### 1. Rate Limiting

Each country client implements rate limiting to respect API quotas:

| Country | Concurrent Requests | Retry Strategy |
|---------|---------------------|----------------|
| Norway | 5 | Exponential backoff |
| Denmark | 5 | Exponential backoff |
| Finland | 5 | Exponential backoff |

### 2. Circuit Breaker

Protects against cascading failures when APIs are unavailable:

- **Failure threshold**: 5 consecutive failures opens circuit
- **Half-open state**: Allows 2 test requests
- **Recovery**: Automatic after successful requests

### 3. Request Deduplication

Identical concurrent requests are deduplicated to reduce API load and improve response times.

### 4. Response Caching

Responses are cached to reduce API calls:

| Country | Cache TTL |
|---------|-----------|
| Norway | 5 minutes |
| Denmark | 5 minutes |
| Finland | 15 minutes |

### 5. Input Validation

Tool parameters are validated before API calls (see issue tracker for planned improvements).

### 6. HTTPS Only

All external API calls use HTTPS exclusively.

## Data Handling

### What Data Flows Through

- **Company names** and search queries
- **Organization numbers** (public identifiers)
- **Public business information** (addresses, board members, registration dates)

### What Does NOT Flow Through

- User credentials
- Personal identification numbers
- Financial data
- Private business information

### Data Sensitivity

All data returned is **public record** from official government registries. However, users should be aware that:

- Board member names are personal data under GDPR
- Company details may reveal business relationships
- Historical data shows business changes over time

## HTTP Server Mode

When running in HTTP server mode (`--http`):

### Authentication

- **Optional**: Set `AUTH_TOKEN` environment variable
- **Warning logged**: Server warns if running without authentication
- **Recommendation**: Always use authentication in production

```bash
AUTH_TOKEN=your-secret-token ./nordic-registry-mcp-server --http :8080
```

### Network Exposure

- Bind to `localhost` for local-only access
- Use a reverse proxy (nginx, Caddy) for production exposure
- Consider firewall rules to restrict access

## Threat Model

### Mitigated Threats

| Threat | Protection |
|--------|------------|
| API abuse | Rate limiting, circuit breaker |
| Duplicate requests | Request deduplication |
| Network failures | Retry with backoff |
| Slow responses | Request timeouts |

### Out of Scope

| Threat | Reason |
|--------|--------|
| Data tampering | Read-only server, no writes |
| Authentication bypass | No auth required for public APIs |
| Injection attacks | No database, no user input in queries beyond search terms |
| SSRF | Only calls predefined API endpoints |

## Deployment Recommendations

### Local Development

```bash
./nordic-registry-mcp-server
```

No special security configuration needed.

### Production HTTP Mode

```bash
# Set authentication token
export AUTH_TOKEN=$(openssl rand -hex 32)

# Run with explicit bind address
./nordic-registry-mcp-server --http 127.0.0.1:8080
```

### Container Deployment

```dockerfile
FROM scratch
COPY nordic-registry-mcp-server /
USER 1000:1000
ENTRYPOINT ["/nordic-registry-mcp-server"]
```

- Run as non-root user
- Use minimal base image
- Set resource limits

## Reporting Security Issues

If you discover a security vulnerability:

1. **Do not** open a public GitHub issue
2. Email the maintainer directly
3. Include steps to reproduce
4. Allow reasonable time for a fix before public disclosure

## Version History

| Version | Date | Security Changes |
|---------|------|------------------|
| 1.0 | 2024-12 | Initial release with rate limiting and circuit breaker |
