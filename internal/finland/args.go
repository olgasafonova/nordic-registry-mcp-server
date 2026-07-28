package finland

// Tag convention (verified against google/jsonschema-go, which the go-sdk uses
// to derive input schemas): the `jsonschema:"..."` tag VALUE becomes the
// property description, and a field is REQUIRED unless its `json` tag carries
// `,omitempty`. There is no `required` keyword — `jsonschema:"required"` merely
// sets the description to the literal word "required". Enum, minimum, maximum
// and pattern constraints are not derivable from tags; they are enforced in the
// handlers (see validation.go) and described in the tag text.

// SearchCompaniesArgs contains parameters for company search
type SearchCompaniesArgs struct {
	Query       string `json:"query" jsonschema:"Finnish company name to search for, 2-500 characters; partial and case-insensitive matches. Common names return 900+ hits, so prefer the exact legal name such as 'Nokia Oyj' or narrow with company_form and location"`
	Location    string `json:"location,omitempty" jsonschema:"Town or city to filter by, e.g. Helsinki or Espoo"`
	CompanyForm string `json:"company_form,omitempty" jsonschema:"Company form code to filter by: OY (private limited), OYJ (public limited), KY (limited partnership), AY (general partnership), and others"`
	Page        int    `json:"page,omitempty" jsonschema:"Page number, 0-indexed (default 0)"`
	Size        int    `json:"size,omitempty" jsonschema:"Results per page, 1-100 (default 20); values above 100 are clamped to 100"`
}

// SearchCompaniesResult is the result of a company search
type SearchCompaniesResult struct {
	Companies    []CompanySummary `json:"companies"`
	TotalResults int              `json:"total_results"`
	Page         int              `json:"page"`
	Size         int              `json:"size"`
	HasMore      bool             `json:"has_more"`
}

// CompanySummary is a simplified company representation for search results
type CompanySummary struct {
	BusinessID       string `json:"business_id"`
	Name             string `json:"name"`
	CompanyForm      string `json:"company_form,omitempty"`
	CompanyFormDesc  string `json:"company_form_desc,omitempty"`
	City             string `json:"city,omitempty"`
	PostCode         string `json:"post_code,omitempty"`
	StreetAddress    string `json:"street_address,omitempty"`
	Industry         string `json:"industry,omitempty"`
	IndustryCode     string `json:"industry_code,omitempty"`
	Website          string `json:"website,omitempty"`
	RegistrationDate string `json:"registration_date,omitempty"`
	Status           string `json:"status,omitempty"`
}

// GetCompanyArgs contains parameters for getting a company by business ID
type GetCompanyArgs struct {
	BusinessID string `json:"business_id" jsonschema:"Finnish business ID (Y-tunnus): 7 digits, a hyphen, then a mod-11 check digit, e.g. 0112038-9. A leading FI prefix is stripped automatically and the check digit is verified"`
	Full       bool   `json:"full,omitempty" jsonschema:"Return the full company record (previous names, auxiliary names, registry entries, situations) instead of the compact summary (default false)"`
}

// GetCompanyResult is the result of getting a company
type GetCompanyResult struct {
	Company *CompanyDetails       `json:"company,omitempty"` // Full company (when full=true)
	Summary *CompanyDetailSummary `json:"summary,omitempty"` // Summary (default)
}

// CompanyDetailSummary is a compact company representation for get_company responses
type CompanyDetailSummary struct {
	BusinessID       string `json:"business_id"`
	Name             string `json:"name"`
	CompanyForm      string `json:"company_form,omitempty"`
	Industry         string `json:"industry,omitempty"`
	City             string `json:"city,omitempty"`
	StreetAddress    string `json:"street_address,omitempty"`
	Website          string `json:"website,omitempty"`
	RegistrationDate string `json:"registration_date,omitempty"`
	Status           string `json:"status,omitempty"`
}

// CompanyDetails contains full company information
type CompanyDetails struct {
	BusinessID       string             `json:"business_id"`
	EUID             string             `json:"eu_id,omitempty"`
	Name             string             `json:"name"`
	PreviousNames    []string           `json:"previous_names,omitempty"`
	AuxiliaryNames   []string           `json:"auxiliary_names,omitempty"`
	CompanyForm      string             `json:"company_form,omitempty"`
	CompanyFormDesc  string             `json:"company_form_desc,omitempty"`
	Industry         string             `json:"industry,omitempty"`
	IndustryCode     string             `json:"industry_code,omitempty"`
	Website          string             `json:"website,omitempty"`
	StreetAddress    *AddressSummary    `json:"street_address,omitempty"`
	PostalAddress    *AddressSummary    `json:"postal_address,omitempty"`
	RegistrationDate string             `json:"registration_date,omitempty"`
	BusinessIDDate   string             `json:"business_id_date,omitempty"`
	Status           string             `json:"status,omitempty"`
	StatusDesc       string             `json:"status_desc,omitempty"`
	Situations       []SituationInfo    `json:"situations,omitempty"`
	Registrations    []RegistrationInfo `json:"registrations,omitempty"`
	LastModified     string             `json:"last_modified,omitempty"`
}

// AddressSummary is a simplified address for MCP responses
type AddressSummary struct {
	Street   string `json:"street,omitempty"`
	PostCode string `json:"post_code,omitempty"`
	City     string `json:"city,omitempty"`
}

// SituationInfo describes company situations (liquidation, bankruptcy, etc.)
type SituationInfo struct {
	Type      string `json:"type"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
}

// RegistrationInfo describes registry entries
type RegistrationInfo struct {
	Register  string `json:"register"`
	Status    string `json:"status"`
	Authority string `json:"authority,omitempty"`
	Date      string `json:"date,omitempty"`
}

// LogAttrs implementations expose each tool's structured-log attributes so
// the handler layer can log requests and results without per-type dispatch.

// LogAttrs returns structured-log attributes for the search request.
func (a SearchCompaniesArgs) LogAttrs() []any { return []any{"query", a.Query} }

// LogAttrs returns structured-log attributes for the company lookup.
func (a GetCompanyArgs) LogAttrs() []any { return []any{"business_id", a.BusinessID} }

// LogAttrs returns structured-log attributes for the search result.
func (r SearchCompaniesResult) LogAttrs() []any {
	return []any{"results_count", len(r.Companies), "total_results", r.TotalResults}
}
