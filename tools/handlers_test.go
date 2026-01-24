package tools

import (
	"log/slog"
	"os"
	"testing"

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

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, logger)

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

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, logger)

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

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, logger)

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

	registry := NewHandlerRegistry(noClient, dkClient, fiClient, logger)
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
		// Denmark tools
		"DKSearchCompanies":    true,
		"DKGetCompany":         true,
		"DKGetProductionUnits": true,
		"DKSearchByPhone":      true,
		"DKGetByPNumber":       true,
		// Finland tools
		"FISearchCompanies": true,
		"FIGetCompany":      true,
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
