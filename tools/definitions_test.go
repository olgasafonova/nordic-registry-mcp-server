package tools

import (
	"strings"
	"testing"
)

func TestDefinitionsNotEmpty(t *testing.T) {
	if len(AllTools) == 0 {
		t.Error("AllTools should not be empty")
	}
}

func TestAllToolsHaveRequiredFields(t *testing.T) {
	for i, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if tool.Name == "" {
				t.Errorf("tool[%d]: Name is required", i)
			}
			if tool.Method == "" {
				t.Errorf("tool[%d] %s: Method is required", i, tool.Name)
			}
			if tool.Title == "" {
				t.Errorf("tool[%d] %s: Title is required", i, tool.Name)
			}
			if tool.Description == "" {
				t.Errorf("tool[%d] %s: Description is required", i, tool.Name)
			}
			if tool.Category == "" {
				t.Errorf("tool[%d] %s: Category is required", i, tool.Name)
			}
			if tool.Country == "" {
				t.Errorf("tool[%d] %s: Country is required", i, tool.Name)
			}
		})
	}
}

func TestToolNamingPrefix(t *testing.T) {
	validPrefixes := map[string]string{
		"norway":  "norway_",
		"denmark": "denmark_",
		"finland": "finland_",
		"sweden":  "sweden_",
	}

	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			prefix, ok := validPrefixes[tool.Country]
			if !ok {
				t.Errorf("tool %q has unknown country %q", tool.Name, tool.Country)
				return
			}
			if !strings.HasPrefix(tool.Name, prefix) {
				t.Errorf("tool %q in country %q should have prefix %q", tool.Name, tool.Country, prefix)
			}
		})
	}
}

func TestToolCountries(t *testing.T) {
	validCountries := map[string]bool{
		"norway":  true,
		"denmark": true,
		"finland": true,
		"sweden":  true,
	}

	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if !validCountries[tool.Country] {
				t.Errorf("tool %q has unknown country %q", tool.Name, tool.Country)
			}
		})
	}
}

func TestToolCategories(t *testing.T) {
	validCategories := map[string]bool{
		"search":    true,
		"read":      true,
		"roles":     true,
		"batch":     true,
		"subunits":  true,
		"updates":   true,
		"reference": true,
		"documents": true,
		"status":    true,
	}

	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if !validCategories[tool.Category] {
				t.Errorf("tool %q has unknown category %q", tool.Name, tool.Category)
			}
		})
	}
}

func TestToolAnnotations(t *testing.T) {
	for _, tool := range AllTools {
		t.Run(tool.Name, func(t *testing.T) {
			if !tool.ReadOnly {
				t.Errorf("tool %q should be ReadOnly (all registries are read-only)", tool.Name)
			}
			if tool.Destructive {
				t.Errorf("tool %q should not be Destructive", tool.Name)
			}
			if !tool.Idempotent {
				t.Errorf("tool %q should be Idempotent (registry lookups are idempotent)", tool.Name)
			}
			if !tool.OpenWorld {
				t.Errorf("tool %q should be OpenWorld (accesses external registries)", tool.Name)
			}
		})
	}
}

func TestToolCount(t *testing.T) {
	expectedCount := 23
	if len(AllTools) != expectedCount {
		t.Errorf("expected %d tools, got %d", expectedCount, len(AllTools))
	}
}

func TestToolCountByCountry(t *testing.T) {
	expected := map[string]int{
		"norway":  12,
		"denmark": 5,
		"finland": 2,
		"sweden":  4,
	}

	for country, want := range expected {
		t.Run(country, func(t *testing.T) {
			got := len(ToolsByCountry(country))
			if got != want {
				t.Errorf("country %q: expected %d tools, got %d", country, want, got)
			}
		})
	}
}

func TestToolNamesUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range AllTools {
		if seen[tool.Name] {
			t.Errorf("duplicate tool name: %q", tool.Name)
		}
		seen[tool.Name] = true
	}
}

func TestToolMethodsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range AllTools {
		if seen[tool.Method] {
			t.Errorf("duplicate method: %q", tool.Method)
		}
		seen[tool.Method] = true
	}
}

func TestDefinitionsByCountryHelper(t *testing.T) {
	tools := ToolsByCountry("norway")
	for _, tool := range tools {
		if tool.Country != "norway" {
			t.Errorf("ToolsByCountry(\"norway\") returned tool %q with country %q", tool.Name, tool.Country)
		}
	}

	empty := ToolsByCountry("iceland")
	if len(empty) != 0 {
		t.Errorf("ToolsByCountry(\"iceland\") should return empty, got %d tools", len(empty))
	}
}

func TestDefinitionsByCategoryHelper(t *testing.T) {
	tools := ToolsByCategory("search")
	for _, tool := range tools {
		if tool.Category != "search" {
			t.Errorf("ToolsByCategory(\"search\") returned tool %q with category %q", tool.Name, tool.Category)
		}
	}
	if len(tools) == 0 {
		t.Error("ToolsByCategory(\"search\") should return at least one tool")
	}

	empty := ToolsByCategory("nonexistent")
	if len(empty) != 0 {
		t.Errorf("ToolsByCategory(\"nonexistent\") should return empty, got %d tools", len(empty))
	}
}
