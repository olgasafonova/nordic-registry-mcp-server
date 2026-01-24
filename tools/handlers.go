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
	"github.com/olgasafonova/nordic-registry-mcp-server/metrics"
	"github.com/olgasafonova/nordic-registry-mcp-server/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// HandlerRegistry provides type-safe tool registration by mapping
// tool names to their concrete handler implementations.
type HandlerRegistry struct {
	norwayClient  *norway.Client
	denmarkClient *denmark.Client
	finlandClient *finland.Client
	logger        *slog.Logger
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(norwayClient *norway.Client, denmarkClient *denmark.Client, finlandClient *finland.Client, logger *slog.Logger) *HandlerRegistry {
	return &HandlerRegistry{
		norwayClient:  norwayClient,
		denmarkClient: denmarkClient,
		finlandClient: finlandClient,
		logger:        logger,
	}
}

// RegisterAll registers all tools with the MCP server.
func (h *HandlerRegistry) RegisterAll(server *mcp.Server) {
	for _, spec := range AllTools {
		h.registerByName(server, spec)
	}
	h.logger.Info("Registered all tools", "count", len(AllTools))
}

// registerByName dispatches to the correct typed registration function.
func (h *HandlerRegistry) registerByName(server *mcp.Server, spec ToolSpec) {
	tool := h.buildTool(spec)

	switch spec.Method {
	// Norway tools
	case "SearchCompanies":
		h.register(server, tool, spec, h.norwayClient.SearchCompaniesMCP)
	case "GetCompany":
		h.register(server, tool, spec, h.norwayClient.GetCompanyMCP)
	case "GetRoles":
		h.register(server, tool, spec, h.norwayClient.GetRolesMCP)
	case "GetSubUnits":
		h.register(server, tool, spec, h.norwayClient.GetSubUnitsMCP)
	case "GetSubUnit":
		h.register(server, tool, spec, h.norwayClient.GetSubUnitMCP)
	case "GetUpdates":
		h.register(server, tool, spec, h.norwayClient.GetUpdatesMCP)
	case "SearchSubUnits":
		h.register(server, tool, spec, h.norwayClient.SearchSubUnitsMCP)
	case "ListMunicipalities":
		h.register(server, tool, spec, h.norwayClient.ListMunicipalitiesMCP)
	case "ListOrgForms":
		h.register(server, tool, spec, h.norwayClient.ListOrgFormsMCP)
	case "GetSubUnitUpdates":
		h.register(server, tool, spec, h.norwayClient.GetSubUnitUpdatesMCP)

	// Denmark tools
	case "DKSearchCompanies":
		h.register(server, tool, spec, h.denmarkClient.SearchCompaniesMCP)
	case "DKGetCompany":
		h.register(server, tool, spec, h.denmarkClient.GetCompanyMCP)
	case "DKGetProductionUnits":
		h.register(server, tool, spec, h.denmarkClient.GetProductionUnitsMCP)
	case "DKSearchByPhone":
		h.register(server, tool, spec, h.denmarkClient.SearchByPhoneMCP)
	case "DKGetByPNumber":
		h.register(server, tool, spec, h.denmarkClient.GetByPNumberMCP)

	// Finland tools
	case "FISearchCompanies":
		h.register(server, tool, spec, h.finlandClient.SearchCompaniesMCP)
	case "FIGetCompany":
		h.register(server, tool, spec, h.finlandClient.GetCompanyMCP)

	default:
		h.logger.Error("Unknown method, tool not registered", "method", spec.Method, "tool", spec.Name)
	}
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
	}

	h.logger.Info("Tool executed", attrs...)
}

// Convenience function to call the generic register with method receiver
func (h *HandlerRegistry) register(server *mcp.Server, tool *mcp.Tool, spec ToolSpec, method any) {
	switch m := method.(type) {
	// Norway tools
	case func(context.Context, norway.SearchCompaniesArgs) (norway.SearchCompaniesResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.GetCompanyArgs) (norway.GetCompanyResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.GetRolesArgs) (norway.GetRolesResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.GetSubUnitsArgs) (norway.GetSubUnitsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.GetSubUnitArgs) (norway.GetSubUnitResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.GetUpdatesArgs) (norway.GetUpdatesResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.SearchSubUnitsArgs) (norway.SearchSubUnitsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.ListMunicipalitiesArgs) (norway.ListMunicipalitiesResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.ListOrgFormsArgs) (norway.ListOrgFormsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, norway.GetSubUnitUpdatesArgs) (norway.GetSubUnitUpdatesResult, error):
		register(h, server, tool, spec, m)

	// Denmark tools
	case func(context.Context, denmark.SearchCompaniesArgs) (denmark.SearchCompaniesResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, denmark.GetCompanyArgs) (denmark.GetCompanyResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, denmark.GetProductionUnitsArgs) (denmark.GetProductionUnitsResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, denmark.SearchByPhoneArgs) (denmark.SearchByPhoneResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, denmark.GetByPNumberArgs) (denmark.GetByPNumberResult, error):
		register(h, server, tool, spec, m)

	// Finland tools
	case func(context.Context, finland.SearchCompaniesArgs) (finland.SearchCompaniesResult, error):
		register(h, server, tool, spec, m)
	case func(context.Context, finland.GetCompanyArgs) (finland.GetCompanyResult, error):
		register(h, server, tool, spec, m)

	default:
		h.logger.Error("Unknown method type, tool not registered", "tool", spec.Name)
	}
}
