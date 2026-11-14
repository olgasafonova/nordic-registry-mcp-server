# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Fixed

- **`sweden_download_document` removes the ZIP files it writes.** Each call left a 1-10 MB file in the system temp directory and nothing ever deleted it. Before writing a new file, the server now removes its own downloads that are older than 24 hours. The returned `path` works as before for that period; copy the file if you need it longer. (#77)

## [v2.0.0] - 2026-11-14

### Upgrade notes

This release changes behaviour that existing callers and deployments can depend on. Check these four points before upgrading.

- **`sweden_download_document` returns a file path.** The `content_base64` field is gone. The tool writes the ZIP to a temporary file on the machine where the server runs and returns `path` and `size_bytes`. A caller that reads `content_base64` must read the file at `path` instead. In HTTP or container deployments the path is on the server's filesystem, not the caller's.
- **HTTP mode refuses to start on a non-loopback address without a token.** The Docker image starts with `-http :8080`, so a container run without `MCP_AUTH_TOKEN` (or `-token`) now exits at startup. Loopback binds (`127.0.0.1`, `localhost`, `::1`) still start without a token and log a warning.
- **`/metrics`, `/status` and `/tools` require the bearer token when one is set.** Add the token to Prometheus scrape configs and dashboards. `/health` and `/ready` stay public.
- **Streamable HTTP is stateless.** Session IDs are ignored, and `GET` and `DELETE` on the MCP endpoint return 405. Clients that negotiate `2025-11-25` through `initialize` keep working.

### Fixed (security)

- **Docker image refreshes CA certificates on every build.** The eMudhra intermediate CA keys leaked on 2026-10-19 and Alpine published `ca-certificates` 20261020-r0 the next day. The Docker workflow builds with a GitHub Actions layer cache, which could keep an old `apk add ca-certificates` layer and ship the old certificate bundle. A `CERT_REFRESH` build argument, set to the workflow run id, now sits directly before the certificate install so that layer is rebuilt on every run. The runtime base moves from `alpine:3.20`, which is past end of life, to `alpine:3.24`. The `latest` image published with v1.2.0 predates the new bundle, and this release replaces it. (#75)
- **Diagnostics endpoints sit behind the auth middleware.** In HTTP mode `/metrics`, `/status` and `/tools` were registered outside the security middleware that guards the MCP handler. They now share its auth path. Binding to a non-loopback address without a token is refused at startup, where it used to log a warning. (`fb3877c`)
- **Base HTTP client refuses all redirects.** The client shared by the Norway, Denmark and Finland registries did not set `CheckRedirect`, so Go's default policy followed up to 10 redirects away from the hardcoded base URL. It now matches the Sweden client, and an unexpected 3xx response is returned as a non-retryable error. (`bad56f9`)
- **Bearer token comparison no longer short-circuits on length.** A length check in front of `subtle.ConstantTimeCompare` returned early on a length mismatch, which let response time reveal whether a guessed token had the right length. (#61)
- Bumped the indirect dependency `google.golang.org/grpc` from v1.80.0 to v1.83.2, covering GHSA-hrxh-6v49-42gf and GHSA-2v4p-qf9q-27wj. (#49, #68)

### Added

- **MCP protocol revision `2026-07-28` over Streamable HTTP.** go-sdk v1.7.0 added the revision and the HTTP handler now runs stateless, which the SDK requires for it. CORS allows the `Mcp-Protocol-Version`, `Mcp-Method` and `Mcp-Name` request headers and no longer allows `DELETE` or `Mcp-Session-Id`. (#52, #53)
- **One-hour `ttlMs` on `tools/list` and `server/discover` results**, on both stdio and HTTP. The SDK default was `ttlMs: 0`. (#55)
- **`Mcp-Param-Org-Number` header for `norway_get_company`** (SEP-2243). The tool declares `org_number` as a header parameter so an HTTP intermediary can route on it without reading the request body. For HTTP clients that negotiate `2026-07-28` or later, the header is mandatory whenever `org_number` is present and must match the body. Stdio is unaffected. (#59)

### Changed

- `sweden_download_document` writes the ZIP to a temporary file and returns `path` and `size_bytes` instead of `content_base64`. See the upgrade notes. (#34)
- All 23 tools now report `idempotentHint: false`, where 1.2.0 reported `true`. The hint was removed from the tool specs on the reasoning that it only carries meaning for tools that modify state, and go-sdk now serializes the unset hint as `false`. `readOnlyHint` stays `true` on every tool. (#57)
- The three tools that need Sweden OAuth2 credentials say so in their descriptions and point at `sweden_check_status`. (#31)
- Upgraded go-sdk v1.5.0 → v1.8.0 and OpenTelemetry v1.43.0 → v1.47.0. Builds use Go 1.26, and `go.mod` now requires Go 1.26.
- Internal refactors with no intended behaviour change, added tests, CI maintenance (a dedicated lint workflow, an `all-checks` aggregator job, release and artifact actions pinned to commit SHAs) and routine dependency bumps.

### Fixed

- **Parameter descriptions.** 20 input fields shipped the literal string `required` as their description and 21 shipped none, because the struct tags used a convention the schema library does not read. All 41 fields across the four countries now carry a real description. The set of required parameters did not change for any tool. (#50)
- **A panic in a tool handler returns an error.** The dispatcher recovered and logged the panic but returned an empty successful result to the caller. It now returns an error that carries the tool name and a correlation ID. The panic value and stack stay in the server log. (`10bba4f`)
- **Version reporting.** Release binaries and the Docker image are stamped with the release version at build time. Earlier binaries reported 1.0.0 whatever the tag was. (`18de5a3`)
- `make build`, `make install` and `docker-compose.yml` name this server. They were copied from another project and built a binary under that project's name. (#60)

## [v1.2.0] - 2026-05-03

### Fixed (security)

- **Sweden client refuses all redirects** — closes 307/308 cross-origin credential-leak vector. The client carries OAuth client credentials (POST to the Bolagsverket token endpoint) and Bearer tokens (Authorization header on authenticated data calls). Without `CheckRedirect`, Go's default policy follows redirects and preserves method+body for 307/308 across origins, so a wiki-or-proxy returning `307 Location: https://attacker/` would re-POST credentials to the attacker. (`27834ae`)
- **Unparsed upstream body truncated in error fallbacks** — five sites (Sweden client.go:198 token endpoint, Sweden :269/:417 data API, Norway :414, Denmark :249) embedded the raw upstream response body verbatim into Go errors that propagate to MCP callers. Fixes restore HG-2 compliance: each site already followed a "parse country-specific JSON envelope first, fall back to raw body" pattern; the fallback path is now capped at 256 bytes with a `...` suffix. The parsed-envelope paths (Bolagsverket Problem Details, Brønnøysundregistrene `apiErr.Message`, CVR `apiErr.String()`) are deliberately preserved as the operator-facing diagnostic. The Sweden token-endpoint developer-portal prose is preserved verbatim (only the body interpolation is truncated). (`ad16428`)

### Changed

- Sweden client error UX on rare 3xx responses (which Bolagsverket does not normally return): now surfaces as a JSON-parse error rather than re-POSTing credentials to the redirect target. Fail-safer.

## [v1.1.0] - 2026-04-05

### Changed
- Improved all 23 tool descriptions with return value descriptions, trigger conditions (USE WHEN), and failure conditions (FAILS WHEN) based on Wang et al. smell category research
- Upgraded go-sdk v1.4.0 → v1.4.1
- Upgraded Go 1.24 → 1.25
- Upgraded semconv to v1.40.0

### Fixed
- Security: GO-2026-4773 (cross-site tool execution for HTTP servers without authorization)
- Security: GO-2026-4770 (improper handling of null Unicode character in JSON parsing)
- Suppressed gosec G117 false positives in Sweden OAuth test fixtures

### CI
- Added go.sum integrity check and govulncheck to CI pipeline
- Made govulncheck advisory (warn, don't fail)
- Bumped Docker actions (login v4, metadata v6, build-push v7, setup-buildx v4)
- Bumped codecov/codecov-action v5 → v6
- Bumped actions/upload-artifact v6 → v7, download-artifact v7 → v8
- Bumped Dockerfile to golang:1.25-alpine
- Use go-version-file instead of hardcoded Go version
