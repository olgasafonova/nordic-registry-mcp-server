package finland

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
			SearchCompaniesArgs{Query: "Nokia Oyj", Location: "Espoo", Size: 20}.LogAttrs(),
			[]any{"query", "Nokia Oyj"},
		},
		{
			"GetCompanyArgs",
			GetCompanyArgs{BusinessID: "0112038-9", Full: true}.LogAttrs(),
			[]any{"business_id", "0112038-9"},
		},
		{
			"SearchCompaniesResult",
			SearchCompaniesResult{
				Companies:    []CompanySummary{{BusinessID: "0112038-9"}, {BusinessID: "1234567-8"}},
				TotalResults: 900,
			}.LogAttrs(),
			[]any{"results_count", 2, "total_results", 900},
		},
		{
			"SearchCompaniesResult empty",
			SearchCompaniesResult{}.LogAttrs(),
			[]any{"results_count", 0, "total_results", 0},
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
