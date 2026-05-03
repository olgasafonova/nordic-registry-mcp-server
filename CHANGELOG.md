# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

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
