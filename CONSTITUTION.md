# Nordic Registry MCP Server Constitution

This document holds the governance articles for the Nordic Registry MCP Server. These articles are **non-negotiable** and **not subject to per-feature override**. They apply to every commit, pull request, and release regardless of urgency or scope.

This document does not change without an explicit constitutional amendment: a dedicated pull request that modifies only this file, reviewed by the maintainer. A feature pull request that would violate an article does not get an exception; it either changes to comply, or it waits behind an amendment.

**Every article below codifies something the repository already does.** No article invents a new requirement. Each names the file or pattern it is drawn from, and each states honestly whether a linter, a test, or a CI job enforces it, or whether it rests on review alone. An article that claims enforcement it does not have is worse than one that admits it has none, because the false claim stops anyone from adding the missing check.

Written 27-08-2026 against `main` at go-sdk v1.7.0, 23 registered tools.

---

## Article I: Tool registration is declarative and single-entry

Adding a tool means adding one `ToolSpec` to `AllTools` in `tools/definitions.go` and one entry to the handler map built by `initHandlers` in `tools/handlers.go`. Handlers MUST NOT be registered by hand-written boilerplate: they go through the generic `makeHandler`/`register[Args, Result]` pair, which is what attaches panic recovery, the 30-second `ToolTimeout`, OpenTelemetry spans, Prometheus metrics, and structured execution logging to every tool uniformly. A tool wired around `register` gets none of those and MUST NOT be merged.

Every spec MUST carry a non-empty `Name`, `Method`, `Title`, `Category`, `Country`, and `Description`. Names begin with their country prefix (`norway_`, `denmark_`, `finland_`, `sweden_`). Names and methods are unique. Categories come from the fixed set in `TestToolCategories`.

The skip path in `registerTool` (unknown method logs a warning and returns false) is a **designed feature here, not only a hole**: Sweden's four handlers are added to the map only when `swedenClient != nil` (`tools/handlers.go:92-97`), so an unconfigured Bolagsverket OAuth pair means the four `sweden_*` specs are deliberately skipped and `RegisteredTools()` keeps them out of the `/tools` discovery endpoint. That is the intended conditional-registration mechanism, and README "Sweden Setup" documents it.

Codifies: `tools/definitions.go` (`AllTools`), `tools/registry.go` (`ToolSpec`), `tools/handlers.go` (`initHandlers`, `makeHandler`, `register`, `registerTool`, `RegisteredTools`), `CONTRIBUTING.md` "Adding New Tools".

**Enforcement: mechanically checked, with one hole.** `tools/definitions_test.go` runs `TestAllToolsHaveRequiredFields`, `TestToolNamingPrefix`, `TestToolCountries`, `TestToolCategories`, `TestToolCount` (pinned at 23), `TestToolCountByCountry` (12/5/2/4), `TestToolNamesUnique`, `TestToolMethodsUnique`; `tools/handlers_test.go` adds `TestToolSpecMethods` (every spec's `Method` must appear in a hand-maintained known list) and `TestRegisterTool` (the unknown-method skip returns false). All run in CI via `go test -race ./...` in `.github/workflows/ci.yml`. The hole: with all four clients configured, no test asserts that all 23 methods actually have handler-map entries. `TestRegisterAll` asserts only a non-zero count, so a new non-Sweden tool whose `Method` was added to `TestToolSpecMethods` but forgotten in `initHandlers` would ride the Sweden skip path silently. Verified 27-08-2026: full suite green.

---

## Article II: Handlers never panic out; startup may panic loudly

Every tool handler runs behind `defer h.recoverPanic(spec.Name, &err)`, and the dispatcher closure in `register` MUST use **named return values**. Without named returns the deferred reassignment is a no-op and a recovered panic reaches the caller as `(nil, zero, nil)`, which an agent reads as a successful empty response. The panic value and stack are logged server-side; only a correlation ID reaches the MCP caller.

This article is incident-born in this repository: the 17-05-2026 portfolio sweep found exactly this silent-nil dispatcher shape in `tools/handlers.go`, where `recoverPanic` recovered and logged but could not write the error back. Fixed 23-05-2026 in commit `10bba4f` ("silent-nil dispatcher writes panic to errPtr"), and the comments at `tools/handlers.go:206-208` and `266-270` now state the named-return requirement, citing HG-1 in the portfolio's `rules/code-review-prompts.md`.

Panics are permitted in exactly one place: registration-time validation that runs at startup. `headerAnnotatedSchema` panics on a `HeaderParams` entry naming an unknown property or a non-primitive type, because the SDK silently skips malformed annotations and the malformed tool would vanish from `tools/list` over HTTP with no error anywhere (`tools/handlers.go:180-199`).

Codifies: `tools/handlers.go` (`register`, `recoverPanic`, `newCorrelationID`, `headerAnnotatedSchema`).

**Enforcement: partially mechanical.** `TestRecoverPanic` (`tools/handlers_test.go:133`) exercises the recovery path, and `TestHeaderAnnotatedSchemaPanicsOnUnknownProperty` / `TestHeaderAnnotatedSchemaPanicsOnNonPrimitive` (`tools/headerparams_test.go`) pin the startup panics. `TestToolInvocation_Success` and `TestToolInvocation_Error` drive real in-memory client-server sessions through the wrapper. Nothing checks that a future edit keeps the named returns; that rests on `register` being the only registration path (Article I) and on review.

---

## Article III: Anything that does I/O takes `context.Context` first

Every country-client method that performs a network call, and every MCP wrapper (`*MCP` methods in `internal/*/mcp.go`), MUST accept `context.Context` as its first parameter and propagate it to the underlying request. The base client threads it through `DoRequest`, `AcquireSlot` (rate-limit wait aborts on cancellation), and backoff sleeps (`waitBeforeAttempt` selects on `ctx.Done()`). The `register` wrapper caps every call with the 30-second `ToolTimeout`.

The only exempt methods are the in-process accessors that touch no network and block on nothing: `CircuitBreakerStats`, `DedupStats`, `Close`, and `ReleaseSlot` on `base.Client` and the country clients; the constructors and their options; and pure helpers such as `sweden.IsConfigured` and `sweden.NormalizeOrgNumber`. This exemption is exhaustive. A new method that reaches a registry API and does not take a context is a violation, not a new exemption.

Codifies: `internal/base/client.go` (`DoRequest`, `AcquireSlot`, `waitBeforeAttempt`), `internal/norway/client.go:147-315` (eleven API methods context-first, with the twelfth, signature rights, in `internal/norway/mcp_signature.go`), the matching methods in `internal/denmark`, `internal/finland`, `internal/sweden`; `tools/handlers.go:223` (`context.WithTimeout`).

**Enforcement: none mechanical.** No linter in `.golangci.yml` checks parameter ordering. `TestRequestDeduplicator_Do_ContextCancellation` (`internal/infra/resilience_test.go:129`) covers cancellation in the dedup layer, but the context-first convention itself rests on review.

---

## Article IV: Errors are never silently discarded

An operation MUST NOT swallow an error. If an error cannot be handled where it occurs it is logged with enough context to identify the failing object, and propagated or surfaced. A best-effort path that drops an error MUST document, at that line, why dropping it is safe — the existing discards are written as `_, _ =` on probe-endpoint writes (`main.go:439`, `:453`, `:458`) and `_ = f.Close()` in the temp-file writer, each in a context where the reason is visible.

The repository's own receipt is the Article II incident: the silent-nil dispatcher converted a panic into a fake success, and the visible symptom was absent data rather than an error. Same family, same lesson — a swallowed failure presents as a correct empty answer, which is the most expensive thing an agent can receive.

Codifies: commit `10bba4f`; `CHANGELOG.md` v1.2.0 "Unparsed upstream body truncated in error fallbacks" (five error paths made honest without dropping the operator-facing diagnostic); `CONTRIBUTING.md` "Error Handling".

**Enforcement: mechanically checked.** `errcheck` is in the enabled linter list in `.golangci.yml` (`errcheck`, `gosec`, `govet`, `ineffassign`, `misspell`, `staticcheck`, `unused`) and runs on every push and pull request via both `.github/workflows/lint.yml` and the lint job in `ci.yml`. The `gosec` exclusion for `G104` carries the comment "Covered by errcheck", and in this repository that claim is **true** — unlike the sister repo where the same comment sat over a disabled check. Verified 27-08-2026: `golangci-lint run ./...` reports **0 issues**. The bare probe `golangci-lint run --no-config --default=none --enable=errcheck ./...` reports 9 hits, of which 3 are production code — `internal/sweden/client.go:211`, `internal/sweden/client.go:260`, `internal/sweden/client_documents.go:83`, all `defer resp.Body.Close()` — and 6 are test-file session closes. The repo config's `std-error-handling` exclusion preset is what absorbs the deferred-`Close` class, which is a deliberate and conventional carve-out, not a gap in the enabled set.

---

## Article V: Every response is bounded, projected, and paginated

No tool returns a raw upstream payload. API responses are projected into trimmed summary types before they leave the server: `buildCompanySummary` and `buildCompanyDetailSummary` in `internal/norway/mcp.go` flatten the HAL envelope, and each country package does the equivalent for its own wire format. Detail is opt-in (`full=true` on `norway_get_company`, `denmark_get_company`, `finland_get_company`); the compact summary is the default.

The current numeric values are the contract: default page size 20, maximum 100 on every paginated search; Norway batch lookups capped at 2000 org numbers (`internal/norway/client.go:294-295`); upstream response bodies capped at `MaxResponseSize` 10 MB (`internal/base/client.go:28`); error-fallback body interpolation capped at 256 bytes (CHANGELOG v1.2.0, commit `ad16428`); and binary payloads returned as a file handle, never inline — `sweden_download_document` writes the ZIP via `writeDocumentToTempFile` (`internal/sweden/mcp.go:228-243`) and returns path plus `size_bytes` precisely because a 1-10 MB report base64-encodes to ~33% more than its raw size. Raising a default or cap is a change to this article's terms and belongs in an amendment.

Codifies: `internal/norway/mcp.go:58,117`, `internal/norway/validation.go:55-60`, `internal/denmark/args.go:71-81`, `internal/finland/args.go:16-24`, `internal/sweden/mcp.go`, `internal/base/client.go:28`.

**Enforcement: partially mechanical.** The per-country validation suites (`internal/*/args_test.go`, `internal/norway/validation.go` tests) pin the size limits, and `internal/sweden/client_documents_test.go` covers the download path. Nothing asserts that a *new* list tool has a cap or that a new detail field defaults to the compact shape; that is convention plus review, and it is the article most likely to erode quietly.

---

## Article VI: Per-country envelope shapes diverge by design

The four country packages own their argument and result shapes independently, and cross-country uniformity is **not a goal of this repository**. Norway paginates with `Page`/`TotalPages` and rejects `size > 100` with an error; Denmark and Finland clamp oversize values to 100; Denmark's search returns a single result because the CVR API does; Sweden has no name search at all. These are faithful projections of four different national APIs, documented in `docs/ARCHITECTURE.md` "Why Three Separate Clients?" and encoded in the `logAttrsProvider` comment at `tools/handlers.go:307-312`, which names the divergence as deliberate.

A pull request that "normalizes" the envelopes across countries is therefore not a cleanup; it is a violation of this article unless it arrives as an amendment with an argument that the upstream APIs converged. The uniformity that IS required lives at the MCP layer: every tool goes through the same `register` wrapper (Article I), every result is a typed struct (Article VII), and every annotation obeys Article IX. Uniform plumbing, divergent payloads.

Codifies: `docs/ARCHITECTURE.md` "Key Design Decisions", `internal/norway/validation.go:55-60` (reject) versus `internal/denmark/args.go:72` and `internal/finland/args.go:17` (clamp), `tools/handlers.go:307-312`.

**Enforcement: none mechanical.** No test can distinguish deliberate divergence from accidental drift. The article exists precisely so that review has something written to point at when a well-meaning normalization PR shows up.

---

## Article VII: Not-found is data, exit paths are loud, and results are typed

Every tool returns a typed result struct with named, stable fields. A lookup that finds nothing returns `Found: false` as a **valid result, not an error** (`internal/norway/args.go:51`, `internal/denmark/args.go:19,105,117`), and batch lookups report the misses in a `not_found` list rather than failing the call (`internal/norway/args.go:254`). List results carry `count` and pagination fields (`internal/norway/args.go:176,193`, `internal/sweden/args.go:48`) so a caller can tell a full page from the last one.

The process-level exits are equally deliberate: `log.Fatalf` on stdio server error (`main.go:214`), on HTTP server error (`main.go:568`), and — the load-bearing one — a **startup refusal** when the HTTP server would bind a non-loopback address without an auth token (`main.go:637-639`). Fail at boot, loudly, rather than serve an open registry proxy.

**Bounded scope, so the gaps are visible rather than implied.** No tool emits a literal "no matching records" string; an empty search returns an empty list with its count. And the Sweden-not-configured state surfaces as absent tools (Article I) plus a startup log line, not as a runtime error. Both are current behaviour; changing either needs code plus an amendment, not a sentence here.

Codifies: `internal/*/args.go` result types, `docs/ARCHITECTURE.md` "Why MCP Wrappers?" ("Handle not-found as a valid result"), `main.go:214,568,637-639`, `security.go:31` (`isLoopbackAddr`).

**Enforcement: partially mechanical.** The wrapper tests in `internal/*/mcp*_test.go` and `tools/handlers_test.go` pin the typed-result shapes, and `TestIsLoopbackAddr` (`main_test.go:360`) covers the loopback classifier. No test asserts an exit code, and the `log.Fatalf` refusal path itself is untestable in-process; it rests on review.

---

## Article VIII: A tool description is a public contract with the agent

The description on a `ToolSpec` is the only thing an agent reads before deciding whether to call a tool. Changing it changes behaviour for every caller, invisibly, with no version bump and no error.

Descriptions follow the established shape and keep it: what the tool does, `USE WHEN:` trigger conditions, `FAILS WHEN:` failure conditions, and cross-references to the sibling tool an agent would otherwise confuse (`norway_get_roles` versus `norway_get_signature_rights`, `norway_get_subunits` versus `norway_get_subunit` versus `norway_search_subunits`). Operational API quirks are encoded where the agent will see them: the Danish single-result `WARNING` with its name-variation strategy lives in `denmark_search_companies`' description and again in `serverInstructions` (`main.go:252-392`), and the Finnish 900+-results narrowing advice likewise. Dropping a cross-reference or a quirk warning when shortening a description is a breaking change whatever happened to the code behind it.

Codifies: `tools/definitions.go` (all 23 specs), `main.go:252` (`serverInstructions`), CHANGELOG v1.1.0 (all 23 descriptions rewritten against the Wang et al. smell categories).

**Enforcement: partially mechanical.** `TestAllToolsHaveRequiredFields` checks non-emptiness only. Nothing checks that an edit preserved a `USE WHEN`/`FAILS WHEN` section or a cross-reference, and this repository has no eval suite pinning confusable pairs. That is a real gap; until one exists, description edits rest entirely on review.

---

## Article IX: Annotations tell the truth about what a tool does

`ReadOnly`, `Destructive`, `Idempotent`, and `OpenWorld` on a `ToolSpec` become MCP tool hints via `buildTool` (`tools/handlers.go:145-163`), and clients use them to decide whether to prompt a human.

This is a fully read-only server, and the annotations MUST say so: every tool is `ReadOnly`, no tool is `Destructive`, and every tool is `OpenWorld` (all reach external registries). A tool MUST NOT be both `ReadOnly` and `Idempotent`: idempotence carries meaning only for tools that change state, and asserting it on a read misleads a client reasoning about retry safety — the reasoning is written into the test itself. A future write-capable tool would need to change the test deliberately, which is exactly the friction intended.

Codifies: `tools/definitions.go` (every spec), `tools/handlers.go` (`buildTool`), `SECURITY.md` "This is a read-only server".

**Enforcement: mechanically checked.** `TestToolAnnotations` (`tools/definitions_test.go:100`) asserts all four properties on every tool, and `TestBuildTool` / `TestBuildTool_DestructiveHint` (`tools/handlers_test.go`) pin the spec-to-hint mapping. All run in CI. This is the best-enforced article in this document.

---

## Article X: Header annotations are validated at startup and mirrored in CORS

`ToolSpec.HeaderParams` declares SEP-2243 `x-mcp-header` passthrough (currently one: `norway_get_company`'s `org_number` travels as `Mcp-Param-Org-Number`). Two rules keep it honest. First, the annotation is validated at registration: `headerAnnotatedSchema` panics on an unknown property or non-primitive type, because the SDK silently skips malformed annotations and the failure would otherwise surface as the whole tool missing from `tools/list` over HTTP (Article II). Second, every new header suffix MUST be added to the CORS `Access-Control-Allow-Headers` list in `security.go:353`, or browser-based clients lose the tool — the requirement is written into the `HeaderParams` doc comment in `tools/registry.go`.

Codifies: `tools/registry.go` (`HeaderParams` and its doc comment), `tools/handlers.go:180-199`, `tools/definitions.go:26`, `security.go:353`.

**Enforcement: mechanical at startup, review for the CORS mirror.** `tools/headerparams_test.go` runs `TestHeaderAnnotatedSchemaAttachesAnnotation`, both panic tests, and `TestAllToolsHeaderParamsAreWellFormed` (every declared `HeaderParams` entry across `AllTools` must build). No test cross-checks `security.go`'s header list against the declared suffixes; that sync rests on the doc comment and review.

---

## Article XI: Outbound HTTP clients refuse redirects

Every HTTP client in this repository sets `CheckRedirect` to refuse all redirects (`http.ErrUseLastResponse`). The registry APIs are pinned single-host upstreams with hardcoded base URLs; legitimate operation never requires following a 3xx, and Go's default policy of following up to 10 redirects would bypass any origin-side validation — for the Sweden client it would re-POST OAuth client credentials and Bearer tokens to whatever host a 307/308 `Location` names.

This article is incident-born: CHANGELOG v1.2.0 records the Sweden credential-leak vector and its fix (commit `27834ae`), and the base-client comment at `internal/base/client.go:315-327` carries the full reasoning for Norway/Denmark/Finland. A new client — or a `WithHTTPClient` override — that ships without the refusal is a violation, not a style choice.

Codifies: `internal/base/client.go:315-328` (`newHTTPClient`), `internal/sweden/client.go:119`, `CHANGELOG.md` v1.2.0.

**Enforcement: mechanically checked for Sweden, review for the base client.** `internal/sweden/redirect_test.go` both attacks the policy with a live redirecting test server and asserts `TestHTTPClient_ProductionConstructor_HasCheckRedirect` on the production constructor. No equivalent test pins `newHTTPClient` in `internal/base`; its refusal is currently guaranteed by code plus comment only. Writing that test is the smallest change that would finish this article's enforcement.

---

## Article XII: The HTTP surface is stateless, and one auth path guards everything but the probes

The Streamable HTTP handler runs `Stateless: true` — required to serve protocol revision 2026-07-28, under which a stateful handler rejects every request with HTTP 400 (`main.go:602-606`, with the SEP-2567 reasoning in the comment). The SEP-2549 `ttlMs` gap is closed by the `mcpcache` receiving middleware stamping one-hour TTLs on `tools/list` and `server/discover` (`main.go:162-180`), because the SDK's own default is 0, meaning immediately stale.

Exactly two endpoints are public: `/health` and `/ready`, so orchestrators can probe. Everything else — `/metrics`, `/tools`, `/status`, and the MCP handler — sits behind the single `SecurityMiddleware` (bearer auth, per-IP rate limiting, CORS, trusted-proxy resolution, request-size caps), wired by `registerSecuredRoutes` feeding `registerHTTPRoutes` (`main.go:414-433`). And per Article VII, a non-loopback bind without a token refuses to start. This routing is itself incident-shaped: commit `fb3877c` exists because the diagnostics endpoints had been mounted beside the auth middleware rather than behind it.

Codifies: `main.go:162-180,414-433,602-606,637-639`, `security.go` (the whole middleware), commit `fb3877c`.

**Enforcement: mechanically checked.** `TestRegisterHTTPRoutesSecuresDiagnostics` (`main_test.go:386`) pins that `/metrics`, `/tools`, and `/status` require the token while `/health` and `/ready` do not — the regression test for `fb3877c`. The middleware itself carries two suites (`security_test.go`, 13 tests; `main_test.go`, 19 tests) covering auth, CORS, rate-limit isolation per IP, trusted proxies, and body limits. The stateless flag has no test; a revert would only surface as live HTTP 400s.

---

## Article XIII: The supply chain is verified on every pull request

CI MUST verify, on every push to `main` and every pull request: that neither `go.mod` nor `go.sum` drifts from `go mod tidy` (the two-file diff is deliberate — a `go get` before the import records a stale `// indirect` that a `go.sum`-only diff cannot see), that tests pass with the race detector, that `golangci-lint` reports no issues, and that `gosec` and `govulncheck` have run.

Codifies: `.github/workflows/ci.yml` (jobs: lint, test, security, go-mod-tidy, build matrix), `.github/workflows/lint.yml`, `.golangci.yml`.

**Enforcement: mechanically checked, with two named exceptions.** First, `govulncheck` runs with `|| echo "::warning::..."`, so a known vulnerability produces a warning annotation and a green build — the stated rationale (stdlib findings resolve with Go patch updates) is real, and the cost is that a vulnerable direct dependency also cannot fail the build. Second, there is **no `go mod verify` step**: module checksums are not explicitly verified in CI, so that half of the portfolio's standard four-step chain is absent here. Adding it is a one-line job step. Coverage upload is `fail_ci_if_error: false`, which is correct for a reporting step. Verified locally 27-08-2026: `golangci-lint run ./...` 0 issues, `go test -race ./...` green across all 11 packages.

---

## Articles considered and rejected

**Semantic versioning, and the changelog is part of the change.** Rejected on the evidence, not the principle. The repository publishes tagged releases, GHCR images, and an MCP Registry entry, and `CHANGELOG.md` is genuinely kept — but three version strings currently disagree: `main.go:32` hardcodes `ServerVersion = "1.0.0"` as a const, `server.json` says `1.0.2`, and the latest git tag is `v1.2.0`. The Makefile even injects `-X main.Version=$(VERSION)` into a variable that does not exist in `main.go`, so the ldflags stamp is silently dropped and every released binary self-reports 1.0.0. A constitution codifies practice; this practice is not yet held. Fix the version plumbing first, then add the article by amendment.

**Test-first development, or a numeric coverage bar.** Rejected because neither is what this repository does. Coverage is real but uneven — `internal/infra` and `security.go` are thoroughly tested, while `main.go`'s HTTP paths gained their test files only in the recent security wave — and `CONTRIBUTING.md` asks for tests without naming a number. Writing ">80%" here would invent a requirement no tooling checks.

**Operations that grant durable access fail closed.** Rejected as not applicable: no tool shares, invites, grants, or writes anything — all 23 are read-only, and Article IX's `TestToolAnnotations` mechanically forbids a `Destructive` tool from appearing quietly. The fail-closed instances this repository actually has are the startup token refusal and the redirect refusal, which stand as Articles XII and XI.

**Fixtures are captured from live responses, with a live-probe workflow.** Rejected because the machinery does not exist here: there is no `api-tracking/` directory and no scheduled probe. The repository's substitute discipline is encoding verified upstream quirks where the agent reads them — the Danish single-result warning and Finnish narrowing advice in the descriptions and `serverInstructions` (Article VIII). An article claiming probe-backed fixture provenance would describe a workflow this repo does not run.

**No credentials in version control as a standalone article.** Rejected as claiming more than anything enforces. The intent is real — Bolagsverket credentials and `MCP_AUTH_TOKEN` are env-only, `server.json` marks them `isSecret`, and there are no token files to leak — but no secret scanner runs in CI and `gosec`'s `G101` is excluded in `.golangci.yml` (documented false positives on URL constants and test fixtures). The enforceable pieces already live in Article XII; a broader article would rest on nothing but review while implying otherwise.

**Structured logging with `log/slog` everywhere.** Rejected as written because the mix is deliberate: `slog` throughout the request path and clients, `log.Fatalf` at the three startup/shutdown sites named in Article VII where crashing loudly is the point. An article would need to carve out exactly those sites, at which point it constrains nothing.

---

## Amendment log

| Date | Change |
|------|--------|
| 27-08-2026 | Ratified. Thirteen articles, adapted from the `CONSTITUTION.md` in `gridctl/gridctl` (Apache-2.0, github.com/gridctl/gridctl). |
