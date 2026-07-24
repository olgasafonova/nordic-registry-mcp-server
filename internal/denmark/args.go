package denmark

// SearchCompaniesArgs contains parameters for company search
type SearchCompaniesArgs struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"Company name to search for"`
}

// SearchCompaniesResult is the result of a company search
type SearchCompaniesResult struct {
	Company *CompanySummary `json:"company,omitempty"`
	Found   bool            `json:"found"`
	Message string          `json:"message,omitempty"`
}

// CompanySummary is a simplified company representation for search results
type CompanySummary struct {
	CVR          string `json:"cvr"`
	Name         string `json:"name"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty"`
	Zipcode      string `json:"zipcode,omitempty"`
	CompanyType  string `json:"company_type,omitempty"`
	IndustryDesc string `json:"industry,omitempty"`
	Employees    string `json:"employees,omitempty"`
	StartDate    string `json:"start_date,omitempty"`
	Status       string `json:"status,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Email        string `json:"email,omitempty"`
}

// GetCompanyArgs contains parameters for getting a company by CVR
type GetCompanyArgs struct {
	CVR  string `json:"cvr" jsonschema:"required" jsonschema_description:"8-digit Danish CVR number"`
	Full bool   `json:"full,omitempty" jsonschema_description:"Return full company details instead of summary (default: false)"`
}

// GetCompanyResult is the result of getting a company
type GetCompanyResult struct {
	Company *Company              `json:"company,omitempty"` // Full company (when full=true)
	Summary *CompanyDetailSummary `json:"summary,omitempty"` // Summary (default)
}

// CompanyDetailSummary is a compact company representation for get_company responses
type CompanyDetailSummary struct {
	CVR             string `json:"cvr"`
	Name            string `json:"name"`
	Address         string `json:"address,omitempty"`
	City            string `json:"city,omitempty"`
	Zipcode         string `json:"zipcode,omitempty"`
	CompanyType     string `json:"company_type,omitempty"`
	Industry        string `json:"industry,omitempty"`
	Employees       int    `json:"employees,omitempty"`
	StartDate       string `json:"start_date,omitempty"`
	Status          string `json:"status,omitempty"`
	Phone           string `json:"phone,omitempty"`
	Email           string `json:"email,omitempty"`
	ProductionUnits int    `json:"production_units_count,omitempty"`
}

// GetProductionUnitsArgs contains parameters for getting production units
type GetProductionUnitsArgs struct {
	CVR  string `json:"cvr" jsonschema:"required" jsonschema_description:"8-digit Danish CVR number"`
	Page int    `json:"page,omitempty" jsonschema_description:"Page number (0-indexed, default: 0)"`
	Size int    `json:"size,omitempty" jsonschema_description:"Results per page (default: 20, max: 100)"`
}

// GetProductionUnitsResult is the result of getting production units
type GetProductionUnitsResult struct {
	ProductionUnits []ProductionUnitSummary `json:"production_units"`
	TotalResults    int                     `json:"total_results"`
	Page            int                     `json:"page"`
	Size            int                     `json:"size"`
	TotalPages      int                     `json:"total_pages"`
	HasMore         bool                    `json:"has_more"`
}

// ProductionUnitSummary is a simplified production unit for MCP responses
type ProductionUnitSummary struct {
	PNumber      string `json:"p_number"`
	Name         string `json:"name"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city,omitempty"`
	Zipcode      string `json:"zipcode,omitempty"`
	IsMain       bool   `json:"is_main"`
	Employees    string `json:"employees,omitempty"`
	IndustryDesc string `json:"industry,omitempty"`
}

// SearchByPhoneArgs contains parameters for searching by phone number
type SearchByPhoneArgs struct {
	Phone string `json:"phone" jsonschema:"required" jsonschema_description:"Phone number to search for"`
}

// SearchByPhoneResult is the result of searching by phone number
type SearchByPhoneResult struct {
	Company *CompanySummary `json:"company,omitempty"`
	Found   bool            `json:"found"`
	Message string          `json:"message,omitempty"`
}

// GetByPNumberArgs contains parameters for getting a company by P-number
type GetByPNumberArgs struct {
	PNumber string `json:"p_number" jsonschema:"required" jsonschema_description:"Production unit P-number"`
}

// GetByPNumberResult is the result of getting a company by P-number
type GetByPNumberResult struct {
	Company *CompanySummary `json:"company,omitempty"`
	Found   bool            `json:"found"`
	Message string          `json:"message,omitempty"`
}

// LogAttrs implementations expose each tool's structured-log attributes so
// the handler layer can log requests and results without per-type dispatch.

// LogAttrs returns structured-log attributes for the search request.
func (a SearchCompaniesArgs) LogAttrs() []any { return []any{"query", a.Query} }

// LogAttrs returns structured-log attributes for the company lookup.
func (a GetCompanyArgs) LogAttrs() []any { return []any{"cvr", a.CVR} }

// LogAttrs returns structured-log attributes for the production-unit listing.
func (a GetProductionUnitsArgs) LogAttrs() []any { return []any{"cvr", a.CVR} }

// LogAttrs returns structured-log attributes for the phone search.
func (a SearchByPhoneArgs) LogAttrs() []any { return []any{"phone", a.Phone} }

// LogAttrs returns structured-log attributes for the P-number lookup.
func (a GetByPNumberArgs) LogAttrs() []any { return []any{"p_number", a.PNumber} }

// LogAttrs returns structured-log attributes for the search result.
func (r SearchCompaniesResult) LogAttrs() []any { return []any{"found", r.Found} }

// LogAttrs returns structured-log attributes for the production-unit result.
func (r GetProductionUnitsResult) LogAttrs() []any {
	return []any{"production_units", len(r.ProductionUnits)}
}

// LogAttrs returns structured-log attributes for the phone-search result.
func (r SearchByPhoneResult) LogAttrs() []any { return []any{"found", r.Found} }

// LogAttrs returns structured-log attributes for the P-number result.
func (r GetByPNumberResult) LogAttrs() []any { return []any{"found", r.Found} }
