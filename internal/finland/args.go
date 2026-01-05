package finland

// SearchCompaniesArgs contains parameters for company search
type SearchCompaniesArgs struct {
	Query       string `json:"query" jsonschema:"required" jsonschema_description:"Company name to search for"`
	Location    string `json:"location,omitempty" jsonschema_description:"Town or city to filter by"`
	CompanyForm string `json:"company_form,omitempty" jsonschema_description:"Company form code (OY, OYJ, Ky, etc.)"`
	Page        int    `json:"page,omitempty" jsonschema_description:"Page number for pagination"`
}

// SearchCompaniesResult is the result of a company search
type SearchCompaniesResult struct {
	Companies    []CompanySummary `json:"companies"`
	TotalResults int              `json:"total_results"`
	Page         int              `json:"page"`
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
	BusinessID string `json:"business_id" jsonschema:"required" jsonschema_description:"Finnish business ID (Y-tunnus), e.g., 0112038-9"`
}

// GetCompanyResult is the result of getting a company
type GetCompanyResult struct {
	Company *CompanyDetails `json:"company"`
}

// CompanyDetails contains full company information
type CompanyDetails struct {
	BusinessID       string            `json:"business_id"`
	EUID             string            `json:"eu_id,omitempty"`
	Name             string            `json:"name"`
	PreviousNames    []string          `json:"previous_names,omitempty"`
	AuxiliaryNames   []string          `json:"auxiliary_names,omitempty"`
	CompanyForm      string            `json:"company_form,omitempty"`
	CompanyFormDesc  string            `json:"company_form_desc,omitempty"`
	Industry         string            `json:"industry,omitempty"`
	IndustryCode     string            `json:"industry_code,omitempty"`
	Website          string            `json:"website,omitempty"`
	StreetAddress    *AddressSummary   `json:"street_address,omitempty"`
	PostalAddress    *AddressSummary   `json:"postal_address,omitempty"`
	RegistrationDate string            `json:"registration_date,omitempty"`
	BusinessIDDate   string            `json:"business_id_date,omitempty"`
	Status           string            `json:"status,omitempty"`
	StatusDesc       string            `json:"status_desc,omitempty"`
	Situations       []SituationInfo   `json:"situations,omitempty"`
	Registrations    []RegistrationInfo `json:"registrations,omitempty"`
	LastModified     string            `json:"last_modified,omitempty"`
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
