package norway

import (
	"reflect"
	"testing"
	"time"
)

// The attribute keys are a logging contract: dashboards and log queries key on
// them, so a rename is a breaking change and belongs in a diff, not a surprise.
func TestLogAttrs(t *testing.T) {
	since := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		got  []any
		want []any
	}{
		{
			"SearchCompaniesArgs",
			SearchCompaniesArgs{Query: "Equinor", Size: 20}.LogAttrs(),
			[]any{"query", "Equinor"},
		},
		{
			"GetCompanyArgs",
			GetCompanyArgs{OrgNumber: "923609016", Full: true}.LogAttrs(),
			[]any{"org_number", "923609016"},
		},
		{
			"GetRolesArgs",
			GetRolesArgs{OrgNumber: "923609016"}.LogAttrs(),
			[]any{"org_number", "923609016"},
		},
		{
			"GetSubUnitsArgs",
			GetSubUnitsArgs{ParentOrgNumber: "923609016"}.LogAttrs(),
			[]any{"parent_org_number", "923609016"},
		},
		{
			"GetSubUnitArgs",
			GetSubUnitArgs{OrgNumber: "912345678"}.LogAttrs(),
			[]any{"org_number", "912345678"},
		},
		{
			"GetUpdatesArgs",
			GetUpdatesArgs{Since: since, Size: 10}.LogAttrs(),
			[]any{"since", since},
		},
		{
			"SearchSubUnitsArgs",
			SearchSubUnitsArgs{Query: "Equinor avd Bergen"}.LogAttrs(),
			[]any{"query", "Equinor avd Bergen"},
		},
		{
			"ListMunicipalitiesArgs",
			ListMunicipalitiesArgs{}.LogAttrs(),
			[]any{},
		},
		{
			"ListOrgFormsArgs",
			ListOrgFormsArgs{}.LogAttrs(),
			[]any{},
		},
		{
			"GetSubUnitUpdatesArgs",
			GetSubUnitUpdatesArgs{Since: since}.LogAttrs(),
			[]any{"since", since},
		},
		{
			"GetSignatureRightsArgs",
			GetSignatureRightsArgs{OrgNumber: "923609016"}.LogAttrs(),
			[]any{"org_number", "923609016"},
		},
		{
			"BatchGetCompaniesArgs",
			BatchGetCompaniesArgs{OrgNumbers: []string{"923609016", "912345678"}}.LogAttrs(),
			[]any{"org_numbers_count", 2},
		},
		{
			"BatchGetCompaniesArgs empty",
			BatchGetCompaniesArgs{}.LogAttrs(),
			[]any{"org_numbers_count", 0},
		},
		{
			"SearchCompaniesResult",
			SearchCompaniesResult{
				Companies:    []CompanySummary{{OrganizationNumber: "923609016"}},
				TotalResults: 42,
			}.LogAttrs(),
			[]any{"results_count", 1, "total_results", 42},
		},
		{
			"GetRolesResult",
			GetRolesResult{RoleGroups: []RoleGroupSummary{{Type: "STYR"}, {Type: "DAGL"}}}.LogAttrs(),
			[]any{"role_groups", 2},
		},
		{
			"GetSubUnitsResult",
			GetSubUnitsResult{SubUnits: []SubUnitSummary{{OrganizationNumber: "912345678"}}}.LogAttrs(),
			[]any{"subunits", 1},
		},
		{
			"GetUpdatesResult",
			GetUpdatesResult{Updates: []UpdateSummary{{UpdateID: 1}, {UpdateID: 2}}}.LogAttrs(),
			[]any{"updates", 2},
		},
		{
			"SearchSubUnitsResult",
			SearchSubUnitsResult{
				SubUnits:     []SubUnitSummary{{OrganizationNumber: "912345678"}},
				TotalResults: 7,
			}.LogAttrs(),
			[]any{"results_count", 1, "total_results", 7},
		},
		{
			"ListMunicipalitiesResult",
			ListMunicipalitiesResult{Count: 356}.LogAttrs(),
			[]any{"municipalities", 356},
		},
		{
			"ListOrgFormsResult",
			ListOrgFormsResult{Count: 43}.LogAttrs(),
			[]any{"org_forms", 43},
		},
		{
			"GetSubUnitUpdatesResult",
			GetSubUnitUpdatesResult{Updates: []SubUnitUpdateSummary{{UpdateID: 1}}}.LogAttrs(),
			[]any{"updates", 1},
		},
		{
			"GetSignatureRightsResult",
			GetSignatureRightsResult{
				SignatureRights: []SignatureRight{{Type: "SIGN"}, {Type: "SIGN"}},
				Prokura:         []SignatureRight{{Type: "PROK"}},
			}.LogAttrs(),
			[]any{"signature_rights", 2, "prokura", 1},
		},
		{
			"BatchGetCompaniesResult",
			BatchGetCompaniesResult{
				Companies: []CompanySummary{{OrganizationNumber: "923609016"}},
				NotFound:  []string{"000000000", "111111111"},
			}.LogAttrs(),
			[]any{"companies", 1, "not_found", 2},
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
