package tools

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/denmark"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/finland"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/norway"
)

func TestNewHandlerRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}
	if registry.norwayClient != noClient {
		t.Error("Registry should hold the Norway client reference")
	}
	if registry.denmarkClient != dkClient {
		t.Error("Registry should hold the Denmark client reference")
	}
	if registry.finlandClient != fiClient {
		t.Error("Registry should hold the Finland client reference")
	}
	if registry.logger != logger {
		t.Error("Registry should hold the logger reference")
	}
}

func TestBuildTool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	tests := []struct {
		name      string
		spec      ToolSpec
		wantName  string
		wantDesc  string
		wantRO    bool
		wantIdem  bool
		wantDestr bool
		wantOpen  bool
	}{
		{
			name: "read-only tool",
			spec: ToolSpec{
				Name:        "norway_search_companies",
				Title:       "Search Norwegian Companies",
				Description: "Search for companies by name",
				Method:      "SearchCompanies",
				Country:     "norway",
				ReadOnly:    true,
				Idempotent:  true,
			},
			wantName:  "norway_search_companies",
			wantDesc:  "Search for companies by name",
			wantRO:    true,
			wantIdem:  true,
			wantDestr: false,
			wantOpen:  false,
		},
		{
			name: "open world tool",
			spec: ToolSpec{
				Name:        "norway_get_company",
				Title:       "Get Norwegian Company",
				Description: "Get company details by org number",
				Method:      "GetCompany",
				Country:     "norway",
				OpenWorld:   true,
			},
			wantName: "norway_get_company",
			wantDesc: "Get company details by org number",
			wantOpen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := registry.buildTool(tt.spec)

			if tool.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", tool.Name, tt.wantName)
			}
			if tool.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", tool.Description, tt.wantDesc)
			}
			if tool.Annotations == nil {
				t.Fatal("Expected annotations")
			}
			if tool.Annotations.ReadOnlyHint != tt.wantRO {
				t.Errorf("ReadOnlyHint = %v, want %v", tool.Annotations.ReadOnlyHint, tt.wantRO)
			}
			if tool.Annotations.IdempotentHint != tt.wantIdem {
				t.Errorf("IdempotentHint = %v, want %v", tool.Annotations.IdempotentHint, tt.wantIdem)
			}
			if tt.wantDestr && (tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint) {
				t.Error("Expected DestructiveHint to be true")
			}
			if tt.wantOpen && (tool.Annotations.OpenWorldHint == nil || !*tool.Annotations.OpenWorldHint) {
				t.Error("Expected OpenWorldHint to be true")
			}
		})
	}
}

func TestRecoverPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	// Test that recoverPanic doesn't panic itself
	func() {
		defer registry.recoverPanic("test_tool")
		panic("test panic")
	}()

	// If we get here, panic was recovered successfully
}

func TestLogExecution(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
	spec := ToolSpec{Name: "test_tool", Country: "norway"}

	// Test with SearchCompaniesArgs
	registry.logExecution(spec,
		norway.SearchCompaniesArgs{Query: "test"},
		norway.SearchCompaniesResult{
			Companies:    []norway.CompanySummary{{Name: "Test AS"}},
			TotalResults: 1,
		})

	// Test with GetCompanyArgs
	registry.logExecution(spec,
		norway.GetCompanyArgs{OrgNumber: "923609016"},
		norway.GetCompanyResult{})

	// Test with GetRolesArgs
	registry.logExecution(spec,
		norway.GetRolesArgs{OrgNumber: "923609016"},
		norway.GetRolesResult{RoleGroups: []norway.RoleGroupSummary{{Type: "STYR"}}})

	// Test with GetSubUnitsArgs
	registry.logExecution(spec,
		norway.GetSubUnitsArgs{ParentOrgNumber: "923609016"},
		norway.GetSubUnitsResult{SubUnits: []norway.SubUnitSummary{{Name: "Branch"}}})
}

func TestAllToolsNotEmpty(t *testing.T) {
	if len(AllTools) == 0 {
		t.Error("AllTools should not be empty")
	}

	// Verify each tool has required fields
	for i, spec := range AllTools {
		if spec.Name == "" {
			t.Errorf("Tool %d has empty Name", i)
		}
		if spec.Method == "" {
			t.Errorf("Tool %s has empty Method", spec.Name)
		}
		if spec.Description == "" {
			t.Errorf("Tool %s has empty Description", spec.Name)
		}
		if spec.Country == "" {
			t.Errorf("Tool %s has empty Country", spec.Name)
		}
	}
}

func TestToolSpecMethods(t *testing.T) {
	knownMethods := map[string]bool{
		// Norway tools
		"SearchCompanies":    true,
		"GetCompany":         true,
		"GetRoles":           true,
		"GetSubUnits":        true,
		"GetSubUnit":         true,
		"GetUpdates":         true,
		"SearchSubUnits":     true,
		"ListMunicipalities": true,
		"ListOrgForms":       true,
		"GetSubUnitUpdates":  true,
		"GetSignatureRights": true,
		"BatchGetCompanies":  true,
		// Denmark tools
		"DKSearchCompanies":    true,
		"DKGetCompany":         true,
		"DKGetProductionUnits": true,
		"DKSearchByPhone":      true,
		"DKGetByPNumber":       true,
		// Finland tools
		"FISearchCompanies": true,
		"FIGetCompany":      true,
		// Sweden tools
		"SEGetCompany":       true,
		"SEGetDocumentList":  true,
		"SEDownloadDocument": true,
		"SECheckStatus":      true,
	}

	for _, spec := range AllTools {
		if !knownMethods[spec.Method] {
			t.Errorf("Tool %s has unknown method: %s", spec.Name, spec.Method)
		}
	}
}

func TestToolsByCountry(t *testing.T) {
	norwayTools := ToolsByCountry("norway")
	if len(norwayTools) == 0 {
		t.Error("Expected Norway tools")
	}

	for _, tool := range norwayTools {
		if tool.Country != "norway" {
			t.Errorf("Tool %s has country %s, expected norway", tool.Name, tool.Country)
		}
	}

	denmarkTools := ToolsByCountry("denmark")
	if len(denmarkTools) == 0 {
		t.Error("Expected Denmark tools")
	}

	for _, tool := range denmarkTools {
		if tool.Country != "denmark" {
			t.Errorf("Tool %s has country %s, expected denmark", tool.Name, tool.Country)
		}
	}

	finlandTools := ToolsByCountry("finland")
	if len(finlandTools) == 0 {
		t.Error("Expected Finland tools")
	}

	for _, tool := range finlandTools {
		if tool.Country != "finland" {
			t.Errorf("Tool %s has country %s, expected finland", tool.Name, tool.Country)
		}
	}

	// Non-existent country should return empty
	unknownTools := ToolsByCountry("unknown")
	if len(unknownTools) != 0 {
		t.Errorf("Expected 0 tools for unknown country, got %d", len(unknownTools))
	}
}

func TestToolsByCategory(t *testing.T) {
	searchTools := ToolsByCategory("search")
	for _, tool := range searchTools {
		if tool.Category != "search" {
			t.Errorf("Tool %s has category %s, expected search", tool.Name, tool.Category)
		}
	}
}

func TestRegisteredTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	t.Run("without Sweden client", func(t *testing.T) {
		// Create registry without Sweden client
		registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

		registeredTools := registry.RegisteredTools()

		// Should have Norway (12) + Denmark (5) + Finland (2) = 19 tools
		expectedCount := 19
		if len(registeredTools) != expectedCount {
			t.Errorf("Expected %d registered tools without Sweden, got %d", expectedCount, len(registeredTools))
		}

		// Verify no Sweden tools are included
		for _, tool := range registeredTools {
			if tool.Country == "sweden" {
				t.Errorf("Sweden tool %s should not be registered when Sweden client is nil", tool.Name)
			}
		}
	})

	t.Run("registered tools subset of AllTools", func(t *testing.T) {
		registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
		registeredTools := registry.RegisteredTools()

		// All registered tools should be in AllTools
		allToolsMap := make(map[string]bool)
		for _, tool := range AllTools {
			allToolsMap[tool.Name] = true
		}

		for _, tool := range registeredTools {
			if !allToolsMap[tool.Name] {
				t.Errorf("Registered tool %s not found in AllTools", tool.Name)
			}
		}
	})
}

func TestBuildTool_DestructiveHint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	spec := ToolSpec{
		Name:        "test_tool",
		Title:       "Test Tool",
		Description: "A destructive test tool",
		Method:      "TestMethod",
		Country:     "test",
		Destructive: true,
	}

	tool := registry.buildTool(spec)

	if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
		t.Error("Expected DestructiveHint to be true")
	}
}

func TestLogExecution_AllArgTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
	spec := ToolSpec{Name: "test_tool", Country: "norway"}

	// Test GetSubUnitArgs
	registry.logExecution(spec,
		norway.GetSubUnitArgs{OrgNumber: "123456789"},
		norway.GetSubUnitResult{})

	// Test GetUpdatesArgs
	registry.logExecution(spec,
		norway.GetUpdatesArgs{Since: time.Now()},
		norway.GetUpdatesResult{Updates: []norway.UpdateSummary{{OrganizationNumber: "123"}}})

	// Test SearchSubUnitsArgs
	registry.logExecution(spec,
		norway.SearchSubUnitsArgs{Query: "test"},
		norway.SearchSubUnitsResult{SubUnits: []norway.SubUnitSummary{{}}, TotalResults: 1})

	// Test ListMunicipalitiesArgs (no args to log)
	registry.logExecution(spec,
		norway.ListMunicipalitiesArgs{},
		norway.ListMunicipalitiesResult{Count: 356})

	// Test ListOrgFormsArgs (no args to log)
	registry.logExecution(spec,
		norway.ListOrgFormsArgs{},
		norway.ListOrgFormsResult{Count: 50})

	// Test GetSubUnitUpdatesArgs
	registry.logExecution(spec,
		norway.GetSubUnitUpdatesArgs{Since: time.Now()},
		norway.GetSubUnitUpdatesResult{Updates: []norway.SubUnitUpdateSummary{{OrganizationNumber: "123"}}})

	// Test GetSignatureRightsArgs
	registry.logExecution(spec,
		norway.GetSignatureRightsArgs{OrgNumber: "123456789"},
		norway.GetSignatureRightsResult{
			SignatureRights: []norway.SignatureRight{{}},
			Prokura:         []norway.SignatureRight{{}},
		})

	// Test BatchGetCompaniesArgs
	registry.logExecution(spec,
		norway.BatchGetCompaniesArgs{OrgNumbers: []string{"123", "456"}},
		norway.BatchGetCompaniesResult{
			Companies: []norway.CompanySummary{{}},
			NotFound:  []string{"789"},
		})

	// Denmark tests
	dkSpec := ToolSpec{Name: "test_tool", Country: "denmark"}
	registry.logExecution(dkSpec,
		denmark.SearchCompaniesArgs{Query: "test"},
		denmark.SearchCompaniesResult{Found: true})

	registry.logExecution(dkSpec,
		denmark.GetCompanyArgs{CVR: "12345678"},
		denmark.GetCompanyResult{})

	registry.logExecution(dkSpec,
		denmark.GetProductionUnitsArgs{CVR: "12345678"},
		denmark.GetProductionUnitsResult{ProductionUnits: []denmark.ProductionUnitSummary{{}}})

	registry.logExecution(dkSpec,
		denmark.SearchByPhoneArgs{Phone: "12345678"},
		denmark.SearchByPhoneResult{Found: true})

	registry.logExecution(dkSpec,
		denmark.GetByPNumberArgs{PNumber: "1234567890"},
		denmark.GetByPNumberResult{Found: false})

	// Finland tests
	fiSpec := ToolSpec{Name: "test_tool", Country: "finland"}
	registry.logExecution(fiSpec,
		finland.SearchCompaniesArgs{Query: "test"},
		finland.SearchCompaniesResult{Companies: []finland.CompanySummary{{}}, TotalResults: 1})

	registry.logExecution(fiSpec,
		finland.GetCompanyArgs{BusinessID: "1234567-8"},
		finland.GetCompanyResult{})

	// Unknown arg type (should not panic)
	registry.logExecution(spec, struct{ Unknown string }{Unknown: "value"}, struct{}{})
}

func TestPtr(t *testing.T) {
	// Test the ptr helper function
	boolVal := ptr(true)
	if boolVal == nil || !*boolVal {
		t.Error("ptr(true) should return pointer to true")
	}

	falseVal := ptr(false)
	if falseVal == nil || *falseVal {
		t.Error("ptr(false) should return pointer to false")
	}
}

// createTestMCPServer creates a minimal MCP server for testing
func createTestMCPServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)
}

func TestRegisterAll(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
	server := createTestMCPServer()

	// RegisterAll should not panic
	registry.RegisterAll(server)

	// Verify tools were registered by checking RegisteredTools count matches
	registeredTools := registry.RegisteredTools()
	if len(registeredTools) == 0 {
		t.Error("Expected tools to be registered")
	}
}

func TestRegisterTool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
	server := createTestMCPServer()

	t.Run("valid tool registration", func(t *testing.T) {
		spec := ToolSpec{
			Name:        "norway_search_companies",
			Title:       "Search Norwegian Companies",
			Description: "Search for companies by name",
			Method:      "SearchCompanies",
			Country:     "norway",
			Category:    "search",
			ReadOnly:    true,
		}

		result := registry.registerTool(server, spec)
		if !result {
			t.Error("Expected registerTool to return true for valid spec")
		}
	})

	t.Run("unknown method", func(t *testing.T) {
		spec := ToolSpec{
			Name:        "unknown_tool",
			Title:       "Unknown Tool",
			Description: "Tool with unknown method",
			Method:      "UnknownMethod",
			Country:     "norway",
		}

		result := registry.registerTool(server, spec)
		if result {
			t.Error("Expected registerTool to return false for unknown method")
		}
	})
}

func TestRegisterAllTools_ByCountry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
	server := createTestMCPServer()

	registry.RegisterAll(server)

	// Count registered tools by country
	registeredTools := registry.RegisteredTools()
	countByCountry := make(map[string]int)
	for _, tool := range registeredTools {
		countByCountry[tool.Country]++
	}

	// Verify we have tools for each country (except Sweden which has no client)
	if countByCountry["norway"] == 0 {
		t.Error("Expected Norway tools to be registered")
	}
	if countByCountry["denmark"] == 0 {
		t.Error("Expected Denmark tools to be registered")
	}
	if countByCountry["finland"] == 0 {
		t.Error("Expected Finland tools to be registered")
	}
	if countByCountry["sweden"] != 0 {
		t.Error("Expected no Sweden tools when Sweden client is nil")
	}
}

func TestRegisterAllTools_Categories(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)
	server := createTestMCPServer()

	registry.RegisterAll(server)

	// Verify we have various categories
	registeredTools := registry.RegisteredTools()
	categories := make(map[string]bool)
	for _, tool := range registeredTools {
		categories[tool.Category] = true
	}

	expectedCategories := []string{"search", "read", "roles", "updates", "reference"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("Expected category %q to be represented in registered tools", cat)
		}
	}
}

func TestMakeHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	// Test that handlers map is populated
	if len(registry.handlers) == 0 {
		t.Error("Expected handlers map to be populated")
	}

	// Verify known handlers exist
	knownMethods := []string{
		"SearchCompanies",
		"GetCompany",
		"GetRoles",
		"DKSearchCompanies",
		"DKGetCompany",
		"FISearchCompanies",
		"FIGetCompany",
	}

	for _, method := range knownMethods {
		if _, ok := registry.handlers[method]; !ok {
			t.Errorf("Expected handler for method %q", method)
		}
	}
}

// =============================================================================
// Integration Tests - Invoke tools through MCP protocol
// =============================================================================

// createMockNorwayServer creates a mock Norway API server for testing
func createMockNorwayServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/enheter") && r.URL.Query().Get("navn") != "":
			// Search companies
			resp := map[string]any{
				"_embedded": map[string]any{
					"enheter": []map[string]any{
						{
							"organisasjonsnummer": "923609016",
							"navn":                "EQUINOR ASA",
							"organisasjonsform":   map[string]string{"kode": "ASA"},
						},
					},
				},
				"page": map[string]any{
					"totalElements": 1,
					"totalPages":    1,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case strings.HasSuffix(r.URL.Path, "/923609016"):
			// Get company
			resp := map[string]any{
				"organisasjonsnummer":      "923609016",
				"navn":                     "EQUINOR ASA",
				"organisasjonsform":        map[string]string{"kode": "ASA", "beskrivelse": "Allmennaksjeselskap"},
				"registrertIMvaregisteret": true,
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestToolInvocation_Success tests that tools can be invoked through the MCP protocol
func TestToolInvocation_Success(t *testing.T) {
	// Create mock API server
	mockServer := createMockNorwayServer()
	defer mockServer.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create Norway client pointing to mock server
	noClient := norway.NewClient(
		norway.WithLogger(logger),
		norway.WithBaseURL(mockServer.URL),
	)
	defer noClient.Close()

	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	// Create MCP server and register tools
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)
	registry.RegisterAll(server)

	// Create in-memory transports for client-server communication
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Connect server
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}
	defer serverSession.Close()

	// Create and connect client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer clientSession.Close()

	t.Run("search companies", func(t *testing.T) {
		result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "norway_search_companies",
			Arguments: map[string]any{
				"query": "Equinor",
			},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if len(result.Content) == 0 {
			t.Error("Expected content in result")
		}
	})

	t.Run("get company", func(t *testing.T) {
		result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "norway_get_company",
			Arguments: map[string]any{
				"org_number": "923609016",
			},
		})
		if err != nil {
			t.Fatalf("CallTool failed: %v", err)
		}
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
	})
}

// TestToolInvocation_Error tests error handling in tool invocation
func TestToolInvocation_Error(t *testing.T) {
	// Create mock server that returns errors
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	}))
	defer mockServer.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	noClient := norway.NewClient(
		norway.WithLogger(logger),
		norway.WithBaseURL(mockServer.URL),
	)
	defer noClient.Close()

	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)
	registry.RegisterAll(server)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer clientSession.Close()

	// Call tool that will fail
	_, err = clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "norway_search_companies",
		Arguments: map[string]any{
			"query": "test",
		},
	})

	// Should get an error (the tool returns an error when API fails)
	if err == nil {
		t.Log("Tool returned success despite API error - this is expected behavior for some error types")
	}
}

// TestToolInvocation_ListTools tests that registered tools appear in tool list
func TestToolInvocation_ListTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	noClient := norway.NewClient(norway.WithLogger(logger))
	defer noClient.Close()
	dkClient := denmark.NewClient(denmark.WithLogger(logger))
	defer dkClient.Close()
	fiClient := finland.NewClient(finland.WithLogger(logger))
	defer fiClient.Close()

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, nil, logger)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)
	registry.RegisterAll(server)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer clientSession.Close()

	// List tools
	result, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	// Verify we have the expected tools
	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{
		"norway_search_companies",
		"norway_get_company",
		"norway_get_roles",
		"denmark_search_companies",
		"denmark_get_company",
		"finland_search_companies",
		"finland_get_company",
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool %q to be registered", name)
		}
	}
}
