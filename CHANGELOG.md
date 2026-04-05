# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

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
