# MediaWiki MCP Server - Remaining Improvements

This document tracks improvements identified during the code review session on 2025-12-19.

## Completed (v1.17.4 - v1.17.6)

| Version | Improvement | Description |
|---------|-------------|-------------|
| v1.17.4 | DNS Rebinding Fix | Added `safeDialer` with connection-time IP validation |
| v1.17.5 | Fail-Closed DNS | DNS errors now block requests instead of allowing |
| v1.17.5 | Structured Error Codes | Added `SSRFError` type with programmatic codes |
| v1.17.6 | Unicode Normalization | NFC normalization for page titles and content validation |

## Remaining Improvements

### Medium Priority (Code Quality)

#### 1. Split methods.go (~4 hours)
**Current state:** 4,174 lines with 52 methods on `Client` struct

**Proposed structure:**
```
wiki/
├── client.go         (auth, caching, core HTTP - keep as is)
├── search.go         (Search, FindSimilarPages, CompareTopic, SearchInPage, SearchInFile)
├── read.go           (GetPage, GetSections, GetPageInfo, GetRelated, GetImages, ListPages)
├── write.go          (EditPage, FindReplace, BulkReplace, ApplyFormatting, UploadFile)
├── history.go        (GetRevisions, CompareRevisions, GetUserContributions, GetRecentChanges)
├── links.go          (GetExternalLinks, CheckLinks, GetBacklinks, FindBrokenInternalLinks, FindOrphanedPages)
├── categories.go     (ListCategories, GetCategoryMembers)
├── quality.go        (CheckTerminology, CheckTranslations, FindSimilarTerms)
├── types.go          (keep as is)
├── errors.go         (keep as is)
└── *_test.go         (split tests to match)
```

**Benefits:**
- Easier navigation and code review
- Logical grouping by functionality
- Smaller, focused files

---

#### 2. Tool Registration Abstraction (~6 hours)
**Current state:** 33 repetitive tool registration blocks in `main.go` (~1,100 lines)

**Each registration looks like:**
```go
registerToolHandler(server, &ToolDefinition{
    Name: "mediawiki_search",
    Description: "...",
    // ... 20+ lines of metadata
}, func(ctx context.Context, req *mcp.CallToolRequest, args wiki.SearchArgs) (*mcp.CallToolResult, wiki.SearchResult, error) {
    // handler
})
```

**Proposed solution:** Metadata-driven registry
```go
type ToolSpec struct {
    Name        string
    Method      string  // e.g., "Search"
    Description string
    Category    string
    ReadOnly    bool
}

var toolRegistry = []ToolSpec{
    {"mediawiki_search", "Search", "Full-text search", "search", true},
    {"mediawiki_get_page", "GetPage", "Get page content", "read", true},
    // ... all tools defined declaratively
}

func registerAllTools(server *mcp.Server, client *wiki.Client) {
    for _, spec := range toolRegistry {
        registerTool(server, spec, client) // uses reflection
    }
}
```

**Benefits:**
- Reduce main.go by ~800 lines
- Easier to add new tools
- Consistent tool metadata
- Single source of truth

---

#### 3. Add Godoc Comments (~2 hours)
**Current state:** ~30% of exported functions lack documentation

**Priority files:**
- `wiki/methods.go` - 52 methods need godoc
- `wiki/client.go` - Some public methods missing docs

**Example fix:**
```go
// Before
func (c *Client) Search(ctx context.Context, args SearchArgs) (SearchResult, error) {

// After
// Search performs full-text search across the wiki.
// It returns pages matching the query, sorted by relevance.
// Use SearchArgs.Limit to control result count (default 20, max 500).
func (c *Client) Search(ctx context.Context, args SearchArgs) (SearchResult, error) {
```

---

#### 4. Bump Test Coverage (~8 hours)
**Current state:**
- `converter/`: 82.2% ✓
- `wiki/`: 26.6% → target 40%
- `main`: 12.7%

**Priority test additions:**
1. Cache eviction under load
2. Concurrent request handling
3. CSRF token refresh on expiry
4. Rate limiter race conditions
5. Content validation edge cases

---

### Low Priority

#### 5. Audit Logging for Writes (~3 hours)
Log all page edits with:
- Timestamp
- Page title
- User (bot account)
- Content hash (for tracking changes)
- Edit summary

Useful for wiki admins to track bot activity.

---

## How to Use This Document

1. Pick an improvement from the list
2. Create a branch: `git checkout -b refactor/split-methods`
3. Implement the change
4. Run tests: `go test ./...`
5. Commit and push
6. Create PR or release

## Session Notes

The codebase evaluation found the server to be **production-quality** with:
- Strong security practices (SSRF, XSS, rate limiting)
- Excellent error handling with recovery suggestions
- Minimal dependencies (only MCP SDK)
- Good Go idioms (concurrency, error wrapping)

Main improvement areas are organizational (file splitting) and reducing boilerplate (tool registration).
