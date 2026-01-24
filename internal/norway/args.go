package norway

import "time"

// SearchCompaniesArgs contains parameters for company search
type SearchCompaniesArgs struct {
	Query           string `json:"query" jsonschema:"required" jsonschema_description:"Company name to search for"`
	Page            int    `json:"page,omitempty" jsonschema_description:"Page number (0-indexed)"`
	Size            int    `json:"size,omitempty" jsonschema_description:"Results per page (max 100)"`
	OrgForm         string `json:"org_form,omitempty" jsonschema_description:"Organization form code (AS, ENK, NUF, etc.)"`
	Municipality    string `json:"municipality,omitempty" jsonschema_description:"Municipality number"`
	RegisteredInVAT *bool  `json:"registered_in_vat,omitempty" jsonschema_description:"Filter by VAT registration"`
	Bankrupt        *bool  `json:"bankrupt,omitempty" jsonschema_description:"Filter by bankruptcy status"`
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
	Status             string `json:"status,omitempty"`           // ACTIVE, BANKRUPT, LIQUIDATING
	Bankrupt           bool   `json:"bankrupt,omitempty"`         // Deprecated: use Status
	UnderLiquidation   bool   `json:"under_liquidation,omitempty"` // Deprecated: use Status
}

// GetCompanyArgs contains parameters for getting a single company
type GetCompanyArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"required" jsonschema_description:"9-digit Norwegian organization number"`
	Full      bool   `json:"full,omitempty" jsonschema_description:"Return full company details instead of summary (default: false)"`
}

// GetCompanyResult is the result of getting a company
type GetCompanyResult struct {
	Company *Company              `json:"company,omitempty"` // Full company (when full=true)
	Summary *CompanyDetailSummary `json:"summary,omitempty"` // Summary (default)
}

// CompanyDetailSummary is a compact company representation for get_company responses
type CompanyDetailSummary struct {
	OrganizationNumber string `json:"organization_number"`
	Name               string `json:"name"`
	OrganizationForm   string `json:"organization_form,omitempty"`
	BusinessAddress    string `json:"business_address,omitempty"`
	PostalAddress      string `json:"postal_address,omitempty"`
	RegistrationDate   string `json:"registration_date,omitempty"`
	EmployeeCount      int    `json:"employee_count,omitempty"`
	Industry           string `json:"industry,omitempty"`
	Website            string `json:"website,omitempty"`
	VATRegistered      bool   `json:"vat_registered,omitempty"`
	Status             string `json:"status,omitempty"`           // ACTIVE, BANKRUPT, LIQUIDATING
	Bankrupt           bool   `json:"bankrupt,omitempty"`         // Deprecated: use Status
	UnderLiquidation   bool   `json:"under_liquidation,omitempty"` // Deprecated: use Status
}

// GetRolesArgs contains parameters for getting company roles
type GetRolesArgs struct {
	OrgNumber string `json:"org_number" jsonschema:"required" jsonschema_description:"9-digit Norwegian organization number"`
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
	ParentOrgNumber string `json:"parent_org_number" jsonschema:"required" jsonschema_description:"Parent company organization number"`
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
	OrgNumber string `json:"org_number" jsonschema:"required" jsonschema_description:"Sub-unit organization number"`
}

// GetSubUnitResult is the result of getting a sub-unit
type GetSubUnitResult struct {
	SubUnit *SubUnit `json:"sub_unit"`
}

// GetUpdatesArgs contains parameters for getting registry updates
type GetUpdatesArgs struct {
	Since time.Time `json:"since" jsonschema:"required" jsonschema_description:"Get updates since this timestamp (ISO 8601)"`
	Size  int       `json:"size,omitempty" jsonschema_description:"Maximum number of updates to return"`
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
	Query        string `json:"query" jsonschema:"required" jsonschema_description:"Sub-unit name to search for"`
	Page         int    `json:"page,omitempty" jsonschema_description:"Page number (0-indexed)"`
	Size         int    `json:"size,omitempty" jsonschema_description:"Results per page (max 100)"`
	Municipality string `json:"municipality,omitempty" jsonschema_description:"Municipality number to filter by"`
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
	Since time.Time `json:"since" jsonschema:"required" jsonschema_description:"Get updates since this timestamp (ISO 8601)"`
	Size  int       `json:"size,omitempty" jsonschema_description:"Maximum number of updates to return"`
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
	OrgNumber string `json:"org_number" jsonschema:"required" jsonschema_description:"9-digit Norwegian organization number"`
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
