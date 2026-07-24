package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/denmark"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/finland"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/norway"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/sweden"
	"github.com/olgasafonova/nordic-registry-mcp-server/metrics"
	"github.com/olgasafonova/nordic-registry-mcp-server/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// registrationFunc is a function that registers a tool with the MCP server.
type registrationFunc func(server *mcp.Server, tool *mcp.Tool, spec ToolSpec)

// HandlerRegistry provides type-safe tool registration by mapping
// tool names to their concrete handler implementations.
type HandlerRegistry struct {
	norwayClient  *norway.Client
	denmarkClient *denmark.Client
	finlandClient *finland.Client
	swedenClient  *sweden.Client // May be nil if OAuth2 credentials not configured
	logger        *slog.Logger
	handlers      map[string]registrationFunc // Method name -> registration function
}

// HandlerRegistryConfig bundles the per-country clients and the logger
// supplied to NewHandlerRegistry. SwedenClient may be nil when Bolagsverket
// OAuth credentials are not configured.
type HandlerRegistryConfig struct {
	NorwayClient  *norway.Client
	DenmarkClient *denmark.Client
	FinlandClient *finland.Client
	SwedenClient  *sweden.Client
	Logger        *slog.Logger
}

// NewHandlerRegistry creates a new handler registry with all handlers pre-built.
func NewHandlerRegistry(cfg HandlerRegistryConfig) *HandlerRegistry {
	h := &HandlerRegistry{
		norwayClient:  cfg.NorwayClient,
		denmarkClient: cfg.DenmarkClient,
		finlandClient: cfg.FinlandClient,
		swedenClient:  cfg.SwedenClient,
		logger:        cfg.Logger,
		handlers:      make(map[string]registrationFunc),
	}
	h.initHandlers()
	return h
}

// initHandlers builds the handler map once at startup.
// This eliminates runtime switch statements during registration.
func (h *HandlerRegistry) initHandlers() {
	// Norway tools
	h.handlers["SearchCompanies"] = makeHandler(h, h.norwayClient.SearchCompaniesMCP)
	h.handlers["GetCompany"] = makeHandler(h, h.norwayClient.GetCompanyMCP)
	h.handlers["GetRoles"] = makeHandler(h, h.norwayClient.GetRolesMCP)
	h.handlers["GetSubUnits"] = makeHandler(h, h.norwayClient.GetSubUnitsMCP)
	h.handlers["GetSubUnit"] = makeHandler(h, h.norwayClient.GetSubUnitMCP)
	h.handlers["GetUpdates"] = makeHandler(h, h.norwayClient.GetUpdatesMCP)
	h.handlers["SearchSubUnits"] = makeHandler(h, h.norwayClient.SearchSubUnitsMCP)
	h.handlers["ListMunicipalities"] = makeHandler(h, h.norwayClient.ListMunicipalitiesMCP)
	h.handlers["ListOrgForms"] = makeHandler(h, h.norwayClient.ListOrgFormsMCP)
	h.handlers["GetSubUnitUpdates"] = makeHandler(h, h.norwayClient.GetSubUnitUpdatesMCP)
	h.handlers["GetSignatureRights"] = makeHandler(h, h.norwayClient.GetSignatureRightsMCP)
	h.handlers["BatchGetCompanies"] = makeHandler(h, h.norwayClient.BatchGetCompaniesMCP)

	// Denmark tools
	h.handlers["DKSearchCompanies"] = makeHandler(h, h.denmarkClient.SearchCompaniesMCP)
	h.handlers["DKGetCompany"] = makeHandler(h, h.denmarkClient.GetCompanyMCP)
	h.handlers["DKGetProductionUnits"] = makeHandler(h, h.denmarkClient.GetProductionUnitsMCP)
	h.handlers["DKSearchByPhone"] = makeHandler(h, h.denmarkClient.SearchByPhoneMCP)
	h.handlers["DKGetByPNumber"] = makeHandler(h, h.denmarkClient.GetByPNumberMCP)

	// Finland tools
	h.handlers["FISearchCompanies"] = makeHandler(h, h.finlandClient.SearchCompaniesMCP)
	h.handlers["FIGetCompany"] = makeHandler(h, h.finlandClient.GetCompanyMCP)

	// Sweden tools (only if client configured)
	if h.swedenClient != nil {
		h.handlers["SEGetCompany"] = makeHandler(h, h.swedenClient.GetCompanyMCP)
		h.handlers["SEGetDocumentList"] = makeHandler(h, h.swedenClient.GetDocumentListMCP)
		h.handlers["SECheckStatus"] = makeHandler(h, h.swedenClient.CheckStatusMCP)
		h.handlers["SEDownloadDocument"] = makeHandler(h, h.swedenClient.DownloadDocumentMCP)
	}
}

// makeHandler creates a registration function for a typed handler method.
// This is the only place where type-specific code is needed.
func makeHandler[Args, Result any](h *HandlerRegistry, method func(context.Context, Args) (Result, error)) registrationFunc {
	return func(server *mcp.Server, tool *mcp.Tool, spec ToolSpec) {
		register(h, server, tool, spec, method)
	}
}

// RegisterAll registers all tools with the MCP server.
func (h *HandlerRegistry) RegisterAll(server *mcp.Server) {
	registered := 0
	for _, spec := range AllTools {
		if h.registerTool(server, spec) {
			registered++
		}
	}
	h.logger.Info("Registered all tools", "count", registered)
}

// registerTool uses the pre-built handler map for O(1) dispatch.
func (h *HandlerRegistry) registerTool(server *mcp.Server, spec ToolSpec) bool {
	handler, ok := h.handlers[spec.Method]
	if !ok {
		h.logger.Warn("No handler for method, skipping tool", "method", spec.Method, "tool", spec.Name)
		return false
	}

	tool := h.buildTool(spec)
	handler(server, tool, spec)
	return true
}

// RegisteredTools returns only the tools that have registered handlers.
// Use this for discovery endpoints to avoid showing unavailable tools.
func (h *HandlerRegistry) RegisteredTools() []ToolSpec {
	registered := make([]ToolSpec, 0, len(AllTools))
	for _, spec := range AllTools {
		if _, ok := h.handlers[spec.Method]; ok {
			registered = append(registered, spec)
		}
	}
	return registered
}

// buildTool creates an mcp.Tool from a ToolSpec.
func (h *HandlerRegistry) buildTool(spec ToolSpec) *mcp.Tool {
	annotations := &mcp.ToolAnnotations{
		Title:          spec.Title,
		ReadOnlyHint:   spec.ReadOnly,
		IdempotentHint: spec.Idempotent,
	}
	if spec.Destructive {
		annotations.DestructiveHint = ptr(true)
	}
	if spec.OpenWorld {
		annotations.OpenWorldHint = ptr(true)
	}

	return &mcp.Tool{
		Name:        spec.Name,
		Description: spec.Description,
		Annotations: annotations,
	}
}

// ToolTimeout is the maximum time allowed for a tool to complete
const ToolTimeout = 30 * time.Second

// register is a generic helper that registers a tool with the MCP server.
// It wraps the client method with panic recovery, metrics, tracing, and logging.
//
// The dispatcher closure uses NAMED return values so the deferred recoverPanic
// can reassign `err` on panic. Without named returns, Go cannot mutate the
// return values from a deferred function and a panic-then-recover would surface
// as `(nil, zero, nil)` to the MCP caller — looking like a successful empty
// response. See HG-1 in rules/code-review-prompts.md.
func register[Args, Result any](
	h *HandlerRegistry,
	server *mcp.Server,
	tool *mcp.Tool,
	spec ToolSpec,
	method func(context.Context, Args) (Result, error),
) {
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, args Args) (res *mcp.CallToolResult, out Result, err error) {
		defer h.recoverPanic(spec.Name, &err)

		// Add timeout to prevent hanging requests
		ctx, cancel := context.WithTimeout(ctx, ToolTimeout)
		defer cancel()

		// Start trace span
		ctx, span := tracing.StartSpan(ctx, "mcp.tool."+spec.Name)
		defer span.End()

		span.SetAttributes(
			attribute.String("mcp.tool.name", spec.Name),
			attribute.String("mcp.tool.category", spec.Category),
			attribute.String("mcp.tool.country", spec.Country),
			attribute.Bool("mcp.tool.readonly", spec.ReadOnly),
		)

		// Track in-flight requests
		metrics.RequestInFlight.WithLabelValues(spec.Name).Inc()
		defer metrics.RequestInFlight.WithLabelValues(spec.Name).Dec()

		start := time.Now()
		result, methodErr := method(ctx, args)
		duration := time.Since(start).Seconds()

		span.SetAttributes(attribute.Float64("mcp.tool.duration_seconds", duration))

		if methodErr != nil {
			span.RecordError(methodErr)
			span.SetStatus(codes.Error, methodErr.Error())
			metrics.RecordRequest(spec.Name, duration, false)
			var zero Result
			return nil, zero, fmt.Errorf("%s failed: %w", spec.Name, methodErr)
		}

		span.SetStatus(codes.Ok, "")
		metrics.RecordRequest(spec.Name, duration, true)
		h.logExecution(spec, args, result)
		return nil, result, nil
	})
}

// recoverPanic recovers from panics in tool handlers and converts them into a
// structured error with a correlation ID. The panic value and stack are logged
// server-side; only the correlation ID reaches the MCP caller.
//
// MUST be called as `defer h.recoverPanic(spec.Name, &err)` from a function
// with NAMED return values. Without named returns the deferred reassignment
// is a no-op and panics surface as silent fake-success responses — i.e.
// (nil, zero, nil) reaches the MCP caller as a successful empty result.
// See HG-1 in rules/code-review-prompts.md.
func (h *HandlerRegistry) recoverPanic(toolName string, errPtr *error) {
	rec := recover()
	if rec == nil {
		return
	}
	corrID := newCorrelationID()
	metrics.PanicsRecovered.WithLabelValues(toolName).Inc()
	h.logger.Error("Panic recovered",
		"tool", toolName,
		"correlation_id", corrID,
		"panic", rec,
		"stack", string(debug.Stack()))
	if errPtr != nil {
		*errPtr = fmt.Errorf("%s: internal error (correlation_id=%s)", toolName, corrID)
	}
}

// newCorrelationID returns a short hex string for log correlation. Falls back
// to a timestamp-based ID if crypto/rand is unavailable (vanishingly rare).
func newCorrelationID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// logExecution logs tool execution details.
func (h *HandlerRegistry) logExecution(spec ToolSpec, args, result any) {
	attrs := []any{"tool", spec.Name, "country", spec.Country}
	attrs = appendLogAttrs(attrs, args)
	attrs = appendLogAttrs(attrs, result)
	h.logger.Info("Tool executed", attrs...)
}

// logAttrsProvider is implemented by arg and result types that expose the
// structured-log attributes for the "Tool executed" log line. Each country
// package owns its types' attribute shapes (matches the per-country
// divergence documented in memory/project_nordic_registry_weird_by_design.md).
type logAttrsProvider interface {
	LogAttrs() []any
}

// appendLogAttrs appends v's structured-log attributes when v declares them;
// unrecognized types leave attrs unchanged.
func appendLogAttrs(attrs []any, v any) []any {
	if p, ok := v.(logAttrsProvider); ok {
		return append(attrs, p.LogAttrs()...)
	}
	return attrs
}
