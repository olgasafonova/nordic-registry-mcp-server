package norway

import "time"

// Tag convention (verified against google/jsonschema-go, which the go-sdk uses
// to derive input schemas): the `jsonschema:"..."` tag VALUE becomes the
// property description, and a field is REQUIRED unless its `json` tag carries
// `,omitempty`. There is no `required` keyword — `jsonschema:"required"` merely
// sets the description to the literal word "required". Enum, minimum, maximum
// and pattern constraints are not derivable from tags; they are enforced in the
// handlers (see validation.go) and described in the tag text.

// SearchCompaniesArgs contains parameters for company search
type SearchCompaniesArgs struct {
	Query                 string `json:"query" jsonschema:"Company name to search for; partial and case-insensitive matches, 2-500 characters"`
	Page                  int    `json:"page,omitempty" jsonschema:"Page number, 0-indexed (default 0)"`
	Size                  int    `json:"size,omitempty" jsonschema:"Results per page, 1-100 (default 20); values above 100 are rejected"`
	OrgForm               string `json:"org_form,omitempty" jsonschema:"Organization form code to filter by: AS (limited company), ENK (sole proprietorship), NUF (foreign branch), and others. Use norway_list_org_forms for the full list"`
	Municipality          string `json:"municipality,omitempty" jsonschema:"4-digit Norwegian municipality code to filter by, e.g. 0301 for Oslo. Use norway_list_municipalities to look up codes"`
	RegisteredInVAT       *bool  `json:"registered_in_vat,omitempty" jsonschema:"Filter by VAT-register (Merverdiavgiftsregisteret) membership; omit for no filter"`
	Bankrupt              *bool  `json:"bankrupt,omitempty" jsonschema:"Filter by bankruptcy status; true returns only bankrupt companies, false only solvent ones, omit for no filter"`
	RegisteredInVoluntary *bool  `json:"registered_in_voluntary,omitempty" jsonschema:"Filter for voluntary/non-profit organizations registered in Frivillighetsregisteret; omit for no filter"`
}

// SearchCompaniesResult is the result of a company search
type SearchCompaniesResult struct {
	Companies    []CompanySummary `json:"companies"`
	TotalResults int              `json:"total_results"`
	Page         int              `json:"page"`
	TotalPages   int              `json:"total_pages"`
}

// CompanySummary is a simplified company representation for search results
type CompanySummary struct {
	OrganizationNumber string `json:"organization_number"`
	Name               string `json:"name"`
	OrganizationForm   string `json:"organization_form,omitempty"`
	PostalAddress      string `json:"postal_address,omitempty"`
	BusinessAddress    string `json:"business_address,omitempty"`
	Status             string `json:"status,omitempty"` // ACTIVE, BANKRUPT, LIQUIDATING
}

// GetCompanyArgs contains parameters for getting a single company
type GetCompanyArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"9-digit Norwegian organization number; spaces and dashes are stripped automatically, e.g. 923609016 or 923 609 016"`
	Full      bool   `json:"full,omitempty" jsonschema:"Return the full company record (all addresses, industry codes, capital) instead of the compact summary (default false)"`
}

// GetCompanyResult is the result of getting a company
type GetCompanyResult struct {
	Found   bool                  `json:"found"`             // Whether the company was found
	Message string                `json:"message,omitempty"` // Message when not found
	Company *Company              `json:"company,omitempty"` // Full company (when full=true)
	Summary *CompanyDetailSummary `json:"summary,omitempty"` // Summary (default)
}

// CompanyDetailSummary is a compact company representation for get_company responses
type CompanyDetailSummary struct {
	OrganizationNumber        string   `json:"organization_number"`
	Name                      string   `json:"name"`
	OrganizationForm          string   `json:"organization_form,omitempty"`
	BusinessAddress           string   `json:"business_address,omitempty"`
	PostalAddress             string   `json:"postal_address,omitempty"`
	RegistrationDate          string   `json:"registration_date,omitempty"`
	EmployeeCount             int      `json:"employee_count,omitempty"`
	Industry                  string   `json:"industry,omitempty"`
	Website                   string   `json:"website,omitempty"`
	VATRegistered             bool     `json:"vat_registered,omitempty"`
	Status                    string   `json:"status,omitempty"` // ACTIVE, BANKRUPT, LIQUIDATING
	RegisteredInVoluntary     bool     `json:"registered_in_voluntary,omitempty"`
	VoluntaryRegistrationDate string   `json:"voluntary_registration_date,omitempty"` // When registered in Frivillighetsregisteret
	Activity                  []string `json:"activity,omitempty"`                    // Activity description for voluntary orgs
}

// GetRolesArgs contains parameters for getting company roles
type GetRolesArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"9-digit Norwegian organization number whose board members, CEO and auditors should be listed; spaces and dashes are stripped automatically"`
}

// GetRolesResult is the result of getting company roles
type GetRolesResult struct {
	RoleGroups []RoleGroupSummary `json:"role_groups"`
}

// RoleGroupSummary is a simplified role group for MCP responses
type RoleGroupSummary struct {
	Type         string        `json:"type"`
	Description  string        `json:"description"`
	LastModified string        `json:"last_modified,omitempty"`
	Roles        []RoleSummary `json:"roles"`
}

// RoleSummary is a simplified role for MCP responses
type RoleSummary struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Name        string `json:"name,omitempty"`          // Person or entity name
	BirthDate   string `json:"birth_date,omitempty"`    // For persons
	Resigned    bool   `json:"resigned,omitempty"`      // Whether resigned
	EntityOrgNr string `json:"entity_org_nr,omitempty"` // For corporate roles
}

// GetSubUnitsArgs contains parameters for getting sub-units
type GetSubUnitsArgs struct {
	ParentOrgNumber string `json:"parent_org_number" jsonschema:"9-digit organization number of the parent company whose branch offices (underenheter) should be listed"`
}

// GetSubUnitsResult is the result of getting sub-units
type GetSubUnitsResult struct {
	SubUnits     []SubUnitSummary `json:"sub_units"`
	TotalResults int              `json:"total_results"`
}

// SubUnitSummary is a simplified sub-unit for MCP responses
type SubUnitSummary struct {
	OrganizationNumber string `json:"organization_number"`
	Name               string `json:"name"`
	ParentOrgNumber    string `json:"parent_org_number"`
	BusinessAddress    string `json:"business_address,omitempty"`
	EmployeeCount      int    `json:"employee_count,omitempty"`
}

// GetSubUnitArgs contains parameters for getting a single sub-unit
type GetSubUnitArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"9-digit organization number of the sub-unit (branch office) itself, not of its parent company"`
}

// GetSubUnitResult is the result of getting a sub-unit
type GetSubUnitResult struct {
	SubUnit *SubUnit `json:"sub_unit"`
}

// GetUpdatesArgs contains parameters for getting registry updates
type GetUpdatesArgs struct {
	Since time.Time `json:"since" jsonschema:"Return main-unit registry changes recorded after this instant. ISO 8601 datetime with timezone, e.g. 2024-01-08T00:00:00Z"`
	Size  int       `json:"size,omitempty" jsonschema:"Maximum number of update entries to return; omit to let the registry apply its own default"`
}

// GetUpdatesResult is the result of getting updates
type GetUpdatesResult struct {
	Updates []UpdateSummary `json:"updates"`
}

// UpdateSummary is a simplified update entry for MCP responses
type UpdateSummary struct {
	UpdateID           int       `json:"update_id"`
	OrganizationNumber string    `json:"organization_number"`
	UpdatedAt          time.Time `json:"updated_at"`
	ChangeType         string    `json:"change_type,omitempty"`
}

// SearchSubUnitsArgs contains parameters for sub-unit search
type SearchSubUnitsArgs struct {
	Query        string `json:"query" jsonschema:"Sub-unit (branch office) name to search for; partial and case-insensitive matches, 2-500 characters"`
	Page         int    `json:"page,omitempty" jsonschema:"Page number, 0-indexed (default 0)"`
	Size         int    `json:"size,omitempty" jsonschema:"Results per page, 1-100 (default 20); values above 100 are rejected"`
	Municipality string `json:"municipality,omitempty" jsonschema:"4-digit Norwegian municipality code to filter by, e.g. 0301 for Oslo. Use norway_list_municipalities to look up codes"`
}

// SearchSubUnitsResult is the result of a sub-unit search
type SearchSubUnitsResult struct {
	SubUnits     []SubUnitSummary `json:"sub_units"`
	TotalResults int              `json:"total_results"`
	Page         int              `json:"page"`
	TotalPages   int              `json:"total_pages"`
}

// ListMunicipalitiesArgs contains parameters for listing municipalities
type ListMunicipalitiesArgs struct {
	// No parameters needed - returns all municipalities
}

// ListMunicipalitiesResult is the result of listing municipalities
type ListMunicipalitiesResult struct {
	Municipalities []MunicipalitySummary `json:"municipalities"`
	Count          int                   `json:"count"`
}

// MunicipalitySummary is a simplified municipality for MCP responses
type MunicipalitySummary struct {
	Number string `json:"number"`
	Name   string `json:"name"`
}

// ListOrgFormsArgs contains parameters for listing organization forms
type ListOrgFormsArgs struct {
	// No parameters needed - returns all organization forms
}

// ListOrgFormsResult is the result of listing organization forms
type ListOrgFormsResult struct {
	OrgForms []OrgFormSummary `json:"org_forms"`
	Count    int              `json:"count"`
}

// OrgFormSummary is a simplified organization form for MCP responses
type OrgFormSummary struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// GetSubUnitUpdatesArgs contains parameters for getting sub-unit updates
type GetSubUnitUpdatesArgs struct {
	Since time.Time `json:"since" jsonschema:"Return sub-unit (branch office) registry changes recorded after this instant. ISO 8601 datetime with timezone, e.g. 2024-01-08T00:00:00Z"`
	Size  int       `json:"size,omitempty" jsonschema:"Maximum number of update entries to return; omit to let the registry apply its own default"`
}

// GetSubUnitUpdatesResult is the result of getting sub-unit updates
type GetSubUnitUpdatesResult struct {
	Updates []SubUnitUpdateSummary `json:"updates"`
}

// SubUnitUpdateSummary is a simplified sub-unit update entry for MCP responses
type SubUnitUpdateSummary struct {
	UpdateID           int       `json:"update_id"`
	OrganizationNumber string    `json:"organization_number"`
	UpdatedAt          time.Time `json:"updated_at"`
	ChangeType         string    `json:"change_type,omitempty"`
}

// GetSignatureRightsArgs contains parameters for getting signature rights
type GetSignatureRightsArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"9-digit Norwegian organization number whose signature rights (signaturrett) and prokura holders should be returned"`
}

// GetSignatureRightsResult is the result of getting signature rights
type GetSignatureRightsResult struct {
	OrganizationNumber string           `json:"organization_number"`
	CompanyName        string           `json:"company_name,omitempty"`
	SignatureRights    []SignatureRight `json:"signature_rights"`
	Prokura            []SignatureRight `json:"prokura"`
	Summary            string           `json:"summary"`
}

// SignatureRight represents a person or entity with signing authority
type SignatureRight struct {
	Type        string `json:"type"`                    // SIGN, PROK, etc.
	Description string `json:"description"`             // Norwegian description
	Name        string `json:"name"`                    // Person or entity name
	BirthDate   string `json:"birth_date,omitempty"`    // For persons
	EntityOrgNr string `json:"entity_org_nr,omitempty"` // For corporate signatories
	Resigned    bool   `json:"resigned,omitempty"`
}

// BatchGetCompaniesArgs contains parameters for batch company lookup
type BatchGetCompaniesArgs struct {
	OrgNumbers []string `json:"org_numbers" jsonschema:"9-digit Norwegian organization numbers to look up in one request, max 2000. Entries that are not 9 digits are skipped and reported under not_found rather than failing the call"`
}

// BatchGetCompaniesResult is the result of batch company lookup
type BatchGetCompaniesResult struct {
	Companies    []CompanySummary `json:"companies"`
	TotalResults int              `json:"total_results"`
	NotFound     []string         `json:"not_found,omitempty"` // Org numbers that were not found
}

// LogAttrs implementations expose each tool's structured-log attributes so
// the handler layer can log requests and results without per-type dispatch.

// LogAttrs returns structured-log attributes for the search request.
func (a SearchCompaniesArgs) LogAttrs() []any { return []any{"query", a.Query} }

// LogAttrs returns structured-log attributes for the company lookup.
func (a GetCompanyArgs) LogAttrs() []any { return []any{"org_number", a.OrgNumber} }

// LogAttrs returns structured-log attributes for the roles lookup.
func (a GetRolesArgs) LogAttrs() []any { return []any{"org_number", a.OrgNumber} }

// LogAttrs returns structured-log attributes for the sub-unit listing.
func (a GetSubUnitsArgs) LogAttrs() []any { return []any{"parent_org_number", a.ParentOrgNumber} }

// LogAttrs returns structured-log attributes for the sub-unit lookup.
func (a GetSubUnitArgs) LogAttrs() []any { return []any{"org_number", a.OrgNumber} }

// LogAttrs returns structured-log attributes for the updates feed request.
func (a GetUpdatesArgs) LogAttrs() []any { return []any{"since", a.Since} }

// LogAttrs returns structured-log attributes for the sub-unit search.
func (a SearchSubUnitsArgs) LogAttrs() []any { return []any{"query", a.Query} }

// LogAttrs marks the type as recognized; there are no arguments to log.
func (ListMunicipalitiesArgs) LogAttrs() []any { return []any{} }

// LogAttrs marks the type as recognized; there are no arguments to log.
func (ListOrgFormsArgs) LogAttrs() []any { return []any{} }

// LogAttrs returns structured-log attributes for the sub-unit updates feed.
func (a GetSubUnitUpdatesArgs) LogAttrs() []any { return []any{"since", a.Since} }

// LogAttrs returns structured-log attributes for the signature-rights lookup.
func (a GetSignatureRightsArgs) LogAttrs() []any { return []any{"org_number", a.OrgNumber} }

// LogAttrs returns structured-log attributes for the batch lookup.
func (a BatchGetCompaniesArgs) LogAttrs() []any { return []any{"org_numbers_count", len(a.OrgNumbers)} }

// LogAttrs returns structured-log attributes for the search result.
func (r SearchCompaniesResult) LogAttrs() []any {
	return []any{"results_count", len(r.Companies), "total_results", r.TotalResults}
}

// LogAttrs returns structured-log attributes for the roles result.
func (r GetRolesResult) LogAttrs() []any { return []any{"role_groups", len(r.RoleGroups)} }

// LogAttrs returns structured-log attributes for the sub-unit listing result.
func (r GetSubUnitsResult) LogAttrs() []any { return []any{"subunits", len(r.SubUnits)} }

// LogAttrs returns structured-log attributes for the updates result.
func (r GetUpdatesResult) LogAttrs() []any { return []any{"updates", len(r.Updates)} }

// LogAttrs returns structured-log attributes for the sub-unit search result.
func (r SearchSubUnitsResult) LogAttrs() []any {
	return []any{"results_count", len(r.SubUnits), "total_results", r.TotalResults}
}

// LogAttrs returns structured-log attributes for the municipality listing.
func (r ListMunicipalitiesResult) LogAttrs() []any { return []any{"municipalities", r.Count} }

// LogAttrs returns structured-log attributes for the org-form listing.
func (r ListOrgFormsResult) LogAttrs() []any { return []any{"org_forms", r.Count} }

// LogAttrs returns structured-log attributes for the sub-unit updates result.
func (r GetSubUnitUpdatesResult) LogAttrs() []any { return []any{"updates", len(r.Updates)} }

// LogAttrs returns structured-log attributes for the signature-rights result.
func (r GetSignatureRightsResult) LogAttrs() []any {
	return []any{"signature_rights", len(r.SignatureRights), "prokura", len(r.Prokura)}
}

// LogAttrs returns structured-log attributes for the batch lookup result.
func (r BatchGetCompaniesResult) LogAttrs() []any {
	return []any{"companies", len(r.Companies), "not_found", len(r.NotFound)}
}
