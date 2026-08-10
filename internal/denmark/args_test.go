package denmark

import (
	"reflect"
	"testing"
)

// The attribute keys are a logging contract: dashboards and log queries key on
// them, so a rename is a breaking change and belongs in a diff, not a surprise.
func TestLogAttrs(t *testing.T) {
	tests := []struct {
		name string
		got  []any
		want []any
	}{
		{
			"SearchCompaniesArgs",
			SearchCompaniesArgs{Query: "Novo Nordisk A/S"}.LogAttrs(),
			[]any{"query", "Novo Nordisk A/S"},
		},
		{
			"GetCompanyArgs",
			GetCompanyArgs{CVR: "10150817", Full: true}.LogAttrs(),
			[]any{"cvr", "10150817"},
		},
		{
			"GetProductionUnitsArgs",
			GetProductionUnitsArgs{CVR: "10150817", Size: 20}.LogAttrs(),
			[]any{"cvr", "10150817"},
		},
		{
			"SearchByPhoneArgs",
			SearchByPhoneArgs{Phone: "33121212"}.LogAttrs(),
			[]any{"phone", "33121212"},
		},
		{
			"GetByPNumberArgs",
			GetByPNumberArgs{PNumber: "1012345678"}.LogAttrs(),
			[]any{"p_number", "1012345678"},
		},
		{
			"SearchCompaniesResult found",
			SearchCompaniesResult{Company: &CompanySummary{CVR: "10150817"}, Found: true}.LogAttrs(),
			[]any{"found", true},
		},
		{
			"SearchCompaniesResult not found",
			SearchCompaniesResult{Found: false, Message: "no match"}.LogAttrs(),
			[]any{"found", false},
		},
		{
			"GetProductionUnitsResult",
			GetProductionUnitsResult{
				ProductionUnits: []ProductionUnitSummary{{PNumber: "1012345678"}, {PNumber: "1087654321"}},
			}.LogAttrs(),
			[]any{"production_units", 2},
		},
		{
			"SearchByPhoneResult",
			SearchByPhoneResult{Found: true}.LogAttrs(),
			[]any{"found", true},
		},
		{
			"GetByPNumberResult",
			GetByPNumberResult{Found: false}.LogAttrs(),
			[]any{"found", false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("LogAttrs() = %v, want %v", tt.got, tt.want)
			}
		})
	}
}
