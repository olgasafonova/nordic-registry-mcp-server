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
	CVR              string `json:"cvr"`
	Name             string `json:"name"`
	Address          string `json:"address,omitempty"`
	City             string `json:"city,omitempty"`
	Zipcode          string `json:"zipcode,omitempty"`
	CompanyType      string `json:"company_type,omitempty"`
	Industry         string `json:"industry,omitempty"`
	Employees        int    `json:"employees,omitempty"`
	StartDate        string `json:"start_date,omitempty"`
	Status           string `json:"status,omitempty"`
	Phone            string `json:"phone,omitempty"`
	Email            string `json:"email,omitempty"`
	ProductionUnits  int    `json:"production_units_count,omitempty"`
}

// GetProductionUnitsArgs contains parameters for getting production units
type GetProductionUnitsArgs struct {
	CVR string `json:"cvr" jsonschema:"required" jsonschema_description:"8-digit Danish CVR number"`
}

// GetProductionUnitsResult is the result of getting production units
type GetProductionUnitsResult struct {
	ProductionUnits []ProductionUnitSummary `json:"production_units"`
	TotalResults    int                     `json:"total_results"`
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
