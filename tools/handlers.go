package tools

import (
	"context"
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

// NewHandlerRegistry creates a new handler registry with all handlers pre-built.
func NewHandlerRegistry(norwayClient *norway.Client, denmarkClient *denmark.Client, finlandClient *finland.Client, swedenClient *sweden.Client, logger *slog.Logger) *HandlerRegistry {
	h := &HandlerRegistry{
		norwayClient:  norwayClient,
		denmarkClient: denmarkClient,
		finlandClient: finlandClient,
		swedenClient:  swedenClient,
		logger:        logger,
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
func register[Args, Result any](
	h *HandlerRegistry,
	server *mcp.Server,
	tool *mcp.Tool,
	spec ToolSpec,
	method func(context.Context, Args) (Result, error),
) {
	mcp.AddTool(server, tool, func(ctx context.Context, req *mcp.CallToolRequest, args Args) (*mcp.CallToolResult, Result, error) {
		defer h.recoverPanic(spec.Name)

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
		result, err := method(ctx, args)
		duration := time.Since(start).Seconds()

		span.SetAttributes(attribute.Float64("mcp.tool.duration_seconds", duration))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			metrics.RecordRequest(spec.Name, duration, false)
			var zero Result
			return nil, zero, fmt.Errorf("%s failed: %w", spec.Name, err)
		}

		span.SetStatus(codes.Ok, "")
		metrics.RecordRequest(spec.Name, duration, true)
		h.logExecution(spec, args, result)
		return nil, result, nil
	})
}

// recoverPanic recovers from panics in tool handlers.
func (h *HandlerRegistry) recoverPanic(toolName string) {
	if rec := recover(); rec != nil {
		metrics.PanicsRecovered.WithLabelValues(toolName).Inc()
		h.logger.Error("Panic recovered",
			"tool", toolName,
			"panic", rec,
			"stack", string(debug.Stack()))
	}
}

// logExecution logs tool execution details.
func (h *HandlerRegistry) logExecution(spec ToolSpec, args, result any) {
	attrs := []any{"tool", spec.Name, "country", spec.Country}

	// Add extractable fields from args using type assertions
	switch a := args.(type) {
	// Norway args
	case norway.SearchCompaniesArgs:
		attrs = append(attrs, "query", a.Query)
	case norway.GetCompanyArgs:
		attrs = append(attrs, "org_number", a.OrgNumber)
	case norway.GetRolesArgs:
		attrs = append(attrs, "org_number", a.OrgNumber)
	case norway.GetSubUnitsArgs:
		attrs = append(attrs, "parent_org_number", a.ParentOrgNumber)
	case norway.GetSubUnitArgs:
		attrs = append(attrs, "org_number", a.OrgNumber)
	case norway.GetUpdatesArgs:
		attrs = append(attrs, "since", a.Since)
	case norway.SearchSubUnitsArgs:
		attrs = append(attrs, "query", a.Query)
	case norway.ListMunicipalitiesArgs:
		// No args to log
	case norway.ListOrgFormsArgs:
		// No args to log
	case norway.GetSubUnitUpdatesArgs:
		attrs = append(attrs, "since", a.Since)
	case norway.GetSignatureRightsArgs:
		attrs = append(attrs, "org_number", a.OrgNumber)
	case norway.BatchGetCompaniesArgs:
		attrs = append(attrs, "org_numbers_count", len(a.OrgNumbers))
	// Denmark args
	case denmark.SearchCompaniesArgs:
		attrs = append(attrs, "query", a.Query)
	case denmark.GetCompanyArgs:
		attrs = append(attrs, "cvr", a.CVR)
	case denmark.GetProductionUnitsArgs:
		attrs = append(attrs, "cvr", a.CVR)
	case denmark.SearchByPhoneArgs:
		attrs = append(attrs, "phone", a.Phone)
	case denmark.GetByPNumberArgs:
		attrs = append(attrs, "p_number", a.PNumber)
	// Finland args
	case finland.SearchCompaniesArgs:
		attrs = append(attrs, "query", a.Query)
	case finland.GetCompanyArgs:
		attrs = append(attrs, "business_id", a.BusinessID)
	// Sweden args
	case sweden.GetCompanyArgs:
		attrs = append(attrs, "org_number", a.OrgNumber)
	case sweden.GetDocumentListArgs:
		attrs = append(attrs, "org_number", a.OrgNumber)
	case sweden.CheckStatusArgs:
		// No args to log
	case sweden.DownloadDocumentArgs:
		attrs = append(attrs, "document_id", a.DocumentID)
	}

	// Add extractable fields from result
	switch r := result.(type) {
	// Norway results
	case norway.SearchCompaniesResult:
		attrs = append(attrs, "results_count", len(r.Companies), "total_results", r.TotalResults)
	case norway.GetRolesResult:
		attrs = append(attrs, "role_groups", len(r.RoleGroups))
	case norway.GetSubUnitsResult:
		attrs = append(attrs, "subunits", len(r.SubUnits))
	case norway.GetUpdatesResult:
		attrs = append(attrs, "updates", len(r.Updates))
	case norway.SearchSubUnitsResult:
		attrs = append(attrs, "results_count", len(r.SubUnits), "total_results", r.TotalResults)
	case norway.ListMunicipalitiesResult:
		attrs = append(attrs, "municipalities", r.Count)
	case norway.ListOrgFormsResult:
		attrs = append(attrs, "org_forms", r.Count)
	case norway.GetSubUnitUpdatesResult:
		attrs = append(attrs, "updates", len(r.Updates))
	case norway.GetSignatureRightsResult:
		attrs = append(attrs, "signature_rights", len(r.SignatureRights), "prokura", len(r.Prokura))
	case norway.BatchGetCompaniesResult:
		attrs = append(attrs, "companies", len(r.Companies), "not_found", len(r.NotFound))
	// Denmark results
	case denmark.SearchCompaniesResult:
		attrs = append(attrs, "found", r.Found)
	case denmark.GetProductionUnitsResult:
		attrs = append(attrs, "production_units", len(r.ProductionUnits))
	case denmark.SearchByPhoneResult:
		attrs = append(attrs, "found", r.Found)
	case denmark.GetByPNumberResult:
		attrs = append(attrs, "found", r.Found)
	// Finland results
	case finland.SearchCompaniesResult:
		attrs = append(attrs, "results_count", len(r.Companies), "total_results", r.TotalResults)
	// Sweden results
	case sweden.GetCompanyResult:
		if r.Company != nil {
			attrs = append(attrs, "found", true, "name", r.Company.Name)
		} else {
			attrs = append(attrs, "found", false)
		}
	case sweden.GetDocumentListResult:
		attrs = append(attrs, "documents", r.Count)
	case sweden.CheckStatusResult:
		attrs = append(attrs, "available", r.Available)
	case sweden.DownloadDocumentResult:
		attrs = append(attrs, "document_id", r.DocumentID, "size_bytes", r.SizeBytes)
	}

	h.logger.Info("Tool executed", attrs...)
}
