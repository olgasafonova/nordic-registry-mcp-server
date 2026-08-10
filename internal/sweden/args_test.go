package sweden

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
			"GetCompanyArgs",
			GetCompanyArgs{OrgNumber: "5560125790"}.LogAttrs(),
			[]any{"org_number", "5560125790"},
		},
		{
			"GetDocumentListArgs",
			GetDocumentListArgs{OrgNumber: "5560125790"}.LogAttrs(),
			[]any{"org_number", "5560125790"},
		},
		{
			"CheckStatusArgs",
			CheckStatusArgs{}.LogAttrs(),
			[]any{},
		},
		{
			"DownloadDocumentArgs",
			DownloadDocumentArgs{DocumentID: "abc-123"}.LogAttrs(),
			[]any{"document_id", "abc-123"},
		},
		{
			"GetCompanyResult found",
			GetCompanyResult{Company: &CompanySummary{Name: "Volvo AB"}}.LogAttrs(),
			[]any{"found", true, "name", "Volvo AB"},
		},
		{
			"GetCompanyResult not found",
			GetCompanyResult{}.LogAttrs(),
			[]any{"found", false},
		},
		{
			"GetDocumentListResult",
			GetDocumentListResult{Count: 3}.LogAttrs(),
			[]any{"documents", 3},
		},
		{
			"CheckStatusResult",
			CheckStatusResult{Available: true, CircuitBreakerStatus: "closed"}.LogAttrs(),
			[]any{"available", true},
		},
		{
			"DownloadDocumentResult",
			DownloadDocumentResult{DocumentID: "abc-123", SizeBytes: 1048576}.LogAttrs(),
			[]any{"document_id", "abc-123", "size_bytes", 1048576},
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
