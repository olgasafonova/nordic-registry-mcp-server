// Package tools provides a metadata-driven registry for MCP tool definitions.
// It reduces boilerplate in main.go by defining tools declaratively and
// using type-safe handlers to register them.
package tools

// ToolSpec defines a tool's metadata for declarative registration.
// Each spec maps to a registry client method with matching Args/Result types.
type ToolSpec struct {
	// Name is the MCP tool name (e.g., "norway_search_companies")
	Name string

	// Method is the client method name (e.g., "SearchCompanies")
	Method string

	// Description is the tool description shown to LLMs
	Description string

	// Title is the human-readable tool title for annotations
	Title string

	// Category groups tools logically (search, read, roles, etc.)
	Category string

	// Country indicates which Nordic country this tool is for
	Country string

	// ReadOnly indicates the tool doesn't modify registry state
	ReadOnly bool

	// Destructive indicates the tool can delete or overwrite data
	Destructive bool

	// Idempotent indicates repeated calls have the same effect
	Idempotent bool

	// OpenWorld indicates the tool accesses external resources
	OpenWorld bool

	// HeaderParams maps an input-schema property name to the Mcp-Param-*
	// header suffix that mirrors it over Streamable HTTP (SEP-2243
	// x-mcp-header). Example: {"org_number": "Org-Number"} declares that the
	// org_number argument travels as the Mcp-Param-Org-Number header, letting
	// an intermediary route or observe per-entity traffic without reading the
	// JSON-RPC body. Only primitive (string, integer, boolean) top-level
	// properties are valid; registration panics on anything else because the
	// SDK silently ignores malformed annotations. Enforcement is strict for
	// clients negotiating >= 2026-07-28 over HTTP: when the annotated
	// argument is present, the header must be present and agree, or the
	// transport rejects the call with -32020 HeaderMismatch. New header
	// suffixes must also be added to the CORS allowlist in security.go.
	HeaderParams map[string]string
}

// ptr is a helper to create a pointer to a value.
func ptr[T any](v T) *T {
	return &v
}
