package norway

import "time"

// SearchCompaniesArgs contains parameters for company search
type SearchCompaniesArgs struct {
	Query           string `json:"query" jsonschema:"description=Company name to search for"`
	Page            int    `json:"page,omitempty" jsonschema:"description=Page number (0-indexed)"`
	Size            int    `json:"size,omitempty" jsonschema:"description=Results per page (max 100)"`
	OrgForm         string `json:"org_form,omitempty" jsonschema:"description=Organization form code (AS, ENK, NUF, etc.)"`
	Municipality    string `json:"municipality,omitempty" jsonschema:"description=Municipality number"`
	RegisteredInVAT *bool  `json:"registered_in_vat,omitempty" jsonschema:"description=Filter by VAT registration"`
	Bankrupt        *bool  `json:"bankrupt,omitempty" jsonschema:"description=Filter by bankruptcy status"`
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
	Bankrupt           bool   `json:"bankrupt,omitempty"`
	UnderLiquidation   bool   `json:"under_liquidation,omitempty"`
}

// GetCompanyArgs contains parameters for getting a single company
type GetCompanyArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"required,description=9-digit Norwegian organization number"`
}

// GetCompanyResult is the result of getting a company
type GetCompanyResult struct {
	Company *Company `json:"company"`
}

// GetRolesArgs contains parameters for getting company roles
type GetRolesArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"required,description=9-digit Norwegian organization number"`
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
	ParentOrgNumber string `json:"parent_org_number" jsonschema:"required,description=Parent company organization number"`
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
	OrgNumber string `json:"org_number" jsonschema:"required,description=Sub-unit organization number"`
}

// GetSubUnitResult is the result of getting a sub-unit
type GetSubUnitResult struct {
	SubUnit *SubUnit `json:"sub_unit"`
}

// GetUpdatesArgs contains parameters for getting registry updates
type GetUpdatesArgs struct {
	Since time.Time `json:"since" jsonschema:"required,description=Get updates since this timestamp (ISO 8601)"`
	Size  int       `json:"size,omitempty" jsonschema:"description=Maximum number of updates to return"`
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
