package norway

import (
	"context"
	"fmt"
	"strings"

	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
)

// MCP Tool wrapper methods
// These methods wrap the client methods with Args/Result types for MCP integration.

// SearchCompaniesMCP is the MCP wrapper for SearchCompanies
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	query := strings.TrimSpace(args.Query)
	if err := ValidateSearchQuery(query); err != nil {
		return SearchCompaniesResult{}, err
	}
	args.Query = query
	if args.Size > 0 {
		if err := ValidateSize(args.Size); err != nil {
			return SearchCompaniesResult{}, err
		}
	}

	opts := &SearchOptions{
		Page:                  args.Page,
		Size:                  args.Size,
		OrgForm:               args.OrgForm,
		Municipality:          args.Municipality,
		RegisteredInVAT:       args.RegisteredInVAT,
		Bankrupt:              args.Bankrupt,
		RegisteredInVoluntary: args.RegisteredInVoluntary,
	}

	resp, err := c.SearchCompanies(ctx, args.Query, opts)
	if err != nil {
		return SearchCompaniesResult{}, err
	}

	companies := make([]CompanySummary, len(resp.Embedded.Companies))
	for i, co := range resp.Embedded.Companies {
		companies[i] = buildCompanySummary(co)
	}

	return SearchCompaniesResult{
		Companies:    companies,
		TotalResults: resp.Page.TotalElements,
		Page:         resp.Page.Number,
		TotalPages:   resp.Page.TotalPages,
	}, nil
}

// buildCompanySummary flattens a Brønnøysund Company into the CompanySummary
// shape returned by the MCP search/batch endpoints. Each optional field group
// is guarded separately so callers don't need to repeat the nil checks.
func buildCompanySummary(co Company) CompanySummary {
	summary := CompanySummary{
		OrganizationNumber: co.OrganizationNumber,
		Name:               co.Name,
		Status:             getStatus(co.Bankrupt, co.UnderLiquidation),
	}
	if co.OrganizationForm != nil {
		summary.OrganizationForm = co.OrganizationForm.Code
	}
	if co.PostalAddress != nil {
		summary.PostalAddress = formatAddress(co.PostalAddress)
	}
	if co.BusinessAddress != nil {
		summary.BusinessAddress = formatAddress(co.BusinessAddress)
	}
	return summary
}

// buildSubUnitSummary flattens a Brønnøysund SubUnit into the SubUnitSummary
// shape returned by the MCP sub-unit endpoints.
func buildSubUnitSummary(su SubUnit) SubUnitSummary {
	summary := SubUnitSummary{
		OrganizationNumber: su.OrganizationNumber,
		Name:               su.Name,
		ParentOrgNumber:    su.ParentOrganizationNumber,
		EmployeeCount:      su.EmployeeCount,
	}
	if su.BusinessAddress != nil {
		summary.BusinessAddress = formatAddress(su.BusinessAddress)
	}
	return summary
}

// GetCompanyMCP is the MCP wrapper for GetCompany
func (c *Client) GetCompanyMCP(ctx context.Context, args GetCompanyArgs) (GetCompanyResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetCompanyResult{}, err
	}

	company, err := c.GetCompany(ctx, args.OrgNumber)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return GetCompanyResult{
				Found:   false,
				Message: fmt.Sprintf("No company found with organization number: %s", args.OrgNumber),
			}, nil
		}
		return GetCompanyResult{}, err
	}

	if args.Full {
		return GetCompanyResult{Found: true, Company: company}, nil
	}
	return GetCompanyResult{Found: true, Summary: buildCompanyDetailSummary(company)}, nil
}

// buildCompanyDetailSummary projects a full Brønnøysund Company response into
// the compact CompanyDetailSummary returned when the caller doesn't request
// `full` data.
func buildCompanyDetailSummary(company *Company) *CompanyDetailSummary {
	summary := &CompanyDetailSummary{
		OrganizationNumber:        company.OrganizationNumber,
		Name:                      company.Name,
		RegistrationDate:          company.RegistrationDate,
		EmployeeCount:             company.EmployeeCount,
		Website:                   company.Website,
		VATRegistered:             company.RegisteredInVAT,
		Status:                    getStatus(company.Bankrupt, company.UnderLiquidation),
		RegisteredInVoluntary:     company.RegisteredInVoluntary,
		VoluntaryRegistrationDate: company.VoluntaryRegistrationDate,
		Activity:                  company.Activity,
	}
	if company.OrganizationForm != nil {
		summary.OrganizationForm = company.OrganizationForm.Code + " - " + company.OrganizationForm.Description
	}
	if company.BusinessAddress != nil {
		summary.BusinessAddress = formatAddress(company.BusinessAddress)
	}
	if company.PostalAddress != nil {
		summary.PostalAddress = formatAddress(company.PostalAddress)
	}
	if company.IndustryCode1 != nil {
		summary.Industry = company.IndustryCode1.Code + " - " + company.IndustryCode1.Description
	}
	return summary
}

// GetRolesMCP is the MCP wrapper for GetRoles
func (c *Client) GetRolesMCP(ctx context.Context, args GetRolesArgs) (GetRolesResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetRolesResult{}, err
	}

	resp, err := c.GetRoles(ctx, args.OrgNumber)
	if err != nil {
		return GetRolesResult{}, err
	}

	groups := make([]RoleGroupSummary, 0, len(resp.RoleGroups))
	for _, rg := range resp.RoleGroups {
		groups = append(groups, buildRoleGroupSummary(rg))
	}

	return GetRolesResult{RoleGroups: groups}, nil
}

// buildRoleGroupSummary flattens a Brønnøysund RoleGroup into the
// RoleGroupSummary shape, delegating per-role projection to buildRoleSummary.
func buildRoleGroupSummary(rg RoleGroup) RoleGroupSummary {
	group := RoleGroupSummary{
		Type:         rg.Type.Code,
		Description:  rg.Type.Description,
		LastModified: rg.LastModified,
		Roles:        make([]RoleSummary, 0, len(rg.Roles)),
	}
	for _, r := range rg.Roles {
		group.Roles = append(group.Roles, buildRoleSummary(r))
	}
	return group
}

// buildRoleSummary flattens a Brønnøysund Role into the RoleSummary returned
// by the MCP roles endpoint. Person and Entity blocks are mutually exclusive
// per the upstream schema; the order below mirrors that contract.
func buildRoleSummary(r Role) RoleSummary {
	role := RoleSummary{
		Type:        r.Type.Code,
		Description: r.Type.Description,
		Resigned:    r.Resigned,
	}
	if r.Person != nil {
		role.Name = r.Person.Name.FullName()
		role.BirthDate = r.Person.BirthDate
	}
	if r.Entity != nil {
		role.EntityOrgNr = r.Entity.OrganizationNumber
		if len(r.Entity.Name) > 0 {
			role.Name = strings.Join(r.Entity.Name, " ")
		}
	}
	return role
}

// GetSubUnitsMCP is the MCP wrapper for GetSubUnits
func (c *Client) GetSubUnitsMCP(ctx context.Context, args GetSubUnitsArgs) (GetSubUnitsResult, error) {
	if err := ValidateOrgNumber(args.ParentOrgNumber); err != nil {
		return GetSubUnitsResult{}, err
	}

	resp, err := c.GetSubUnits(ctx, args.ParentOrgNumber)
	if err != nil {
		return GetSubUnitsResult{}, err
	}

	subunits := make([]SubUnitSummary, len(resp.Embedded.SubUnits))
	for i, su := range resp.Embedded.SubUnits {
		subunits[i] = buildSubUnitSummary(su)
	}

	return GetSubUnitsResult{
		SubUnits:     subunits,
		TotalResults: resp.Page.TotalElements,
	}, nil
}

// GetSubUnitMCP is the MCP wrapper for GetSubUnit
func (c *Client) GetSubUnitMCP(ctx context.Context, args GetSubUnitArgs) (GetSubUnitResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetSubUnitResult{}, err
	}

	subunit, err := c.GetSubUnit(ctx, args.OrgNumber)
	if err != nil {
		return GetSubUnitResult{}, err
	}
	return GetSubUnitResult{SubUnit: subunit}, nil
}

// GetUpdatesMCP is the MCP wrapper for GetUpdates
func (c *Client) GetUpdatesMCP(ctx context.Context, args GetUpdatesArgs) (GetUpdatesResult, error) {
	opts := &UpdatesOptions{Size: args.Size}
	resp, err := c.GetUpdates(ctx, args.Since, opts)
	if err != nil {
		return GetUpdatesResult{}, err
	}

	// Convert to summary format
	updates := make([]UpdateSummary, 0, len(resp.Embedded.Updates))
	for _, u := range resp.Embedded.Updates {
		updates = append(updates, UpdateSummary(u))
	}

	return GetUpdatesResult{Updates: updates}, nil
}

// SearchSubUnitsMCP is the MCP wrapper for SearchSubUnits
func (c *Client) SearchSubUnitsMCP(ctx context.Context, args SearchSubUnitsArgs) (SearchSubUnitsResult, error) {
	query := strings.TrimSpace(args.Query)
	if err := ValidateSearchQuery(query); err != nil {
		return SearchSubUnitsResult{}, err
	}
	args.Query = query
	if args.Size > 0 {
		if err := ValidateSize(args.Size); err != nil {
			return SearchSubUnitsResult{}, err
		}
	}

	opts := &SearchSubUnitsOptions{
		Page:         args.Page,
		Size:         args.Size,
		Municipality: args.Municipality,
	}

	resp, err := c.SearchSubUnits(ctx, args.Query, opts)
	if err != nil {
		return SearchSubUnitsResult{}, err
	}

	subunits := make([]SubUnitSummary, 0, len(resp.Embedded.SubUnits))
	for _, su := range resp.Embedded.SubUnits {
		subunits = append(subunits, buildSubUnitSummary(su))
	}

	return SearchSubUnitsResult{
		SubUnits:     subunits,
		TotalResults: resp.Page.TotalElements,
		Page:         resp.Page.Number,
		TotalPages:   resp.Page.TotalPages,
	}, nil
}

// ListMunicipalitiesMCP is the MCP wrapper for GetMunicipalities
func (c *Client) ListMunicipalitiesMCP(ctx context.Context, args ListMunicipalitiesArgs) (ListMunicipalitiesResult, error) {
	resp, err := c.GetMunicipalities(ctx)
	if err != nil {
		return ListMunicipalitiesResult{}, err
	}

	// Convert to summary format
	municipalities := make([]MunicipalitySummary, 0, len(resp.Embedded.Municipalities))
	for _, m := range resp.Embedded.Municipalities {
		municipalities = append(municipalities, MunicipalitySummary(m))
	}

	return ListMunicipalitiesResult{
		Municipalities: municipalities,
		Count:          len(municipalities),
	}, nil
}

// ListOrgFormsMCP is the MCP wrapper for GetOrgForms
func (c *Client) ListOrgFormsMCP(ctx context.Context, args ListOrgFormsArgs) (ListOrgFormsResult, error) {
	resp, err := c.GetOrgForms(ctx)
	if err != nil {
		return ListOrgFormsResult{}, err
	}

	// Convert to summary format
	orgForms := make([]OrgFormSummary, 0, len(resp.Embedded.OrgForms))
	for _, of := range resp.Embedded.OrgForms {
		orgForms = append(orgForms, OrgFormSummary(of))
	}

	return ListOrgFormsResult{
		OrgForms: orgForms,
		Count:    len(orgForms),
	}, nil
}

// GetSubUnitUpdatesMCP is the MCP wrapper for GetSubUnitUpdates
func (c *Client) GetSubUnitUpdatesMCP(ctx context.Context, args GetSubUnitUpdatesArgs) (GetSubUnitUpdatesResult, error) {
	opts := &UpdatesOptions{Size: args.Size}
	resp, err := c.GetSubUnitUpdates(ctx, args.Since, opts)
	if err != nil {
		return GetSubUnitUpdatesResult{}, err
	}

	// Convert to summary format
	updates := make([]SubUnitUpdateSummary, 0, len(resp.Embedded.Updates))
	for _, u := range resp.Embedded.Updates {
		updates = append(updates, SubUnitUpdateSummary(u))
	}

	return GetSubUnitUpdatesResult{Updates: updates}, nil
}

// BatchGetCompaniesMCP is the MCP wrapper for BatchGetCompanies
func (c *Client) BatchGetCompaniesMCP(ctx context.Context, args BatchGetCompaniesArgs) (BatchGetCompaniesResult, error) {
	if len(args.OrgNumbers) == 0 {
		return BatchGetCompaniesResult{}, nil
	}

	// Validate all org numbers
	for _, on := range args.OrgNumbers {
		if err := ValidateOrgNumber(on); err != nil {
			return BatchGetCompaniesResult{}, err
		}
	}

	resp, err := c.BatchGetCompanies(ctx, args.OrgNumbers)
	if err != nil {
		return BatchGetCompaniesResult{}, err
	}

	companies, found := buildBatchCompanies(resp.Embedded.Companies)
	notFound := diffMissingOrgNumbers(args.OrgNumbers, found)

	return BatchGetCompaniesResult{
		Companies:    companies,
		TotalResults: len(companies),
		NotFound:     notFound,
	}, nil
}

// buildBatchCompanies projects an upstream batch response into CompanySummary
// records and returns the set of upstream-known organization numbers (used by
// the caller to compute the not-found delta).
func buildBatchCompanies(in []Company) ([]CompanySummary, map[string]bool) {
	found := make(map[string]bool, len(in))
	companies := make([]CompanySummary, len(in))
	for i, co := range in {
		found[co.OrganizationNumber] = true
		companies[i] = buildCompanySummary(co)
	}
	return companies, found
}

// diffMissingOrgNumbers returns the input org numbers (preserving caller form,
// e.g. with separators) that are not present in the found set. Normalization
// is applied to the lookup, not the output, so error messages keep the
// caller's original spelling.
func diffMissingOrgNumbers(requested []string, found map[string]bool) []string {
	var notFound []string
	for _, on := range requested {
		if !found[NormalizeOrgNumber(on)] {
			notFound = append(notFound, on)
		}
	}
	return notFound
}

// getStatus derives company status from boolean flags
func getStatus(bankrupt, underLiquidation bool) string {
	if bankrupt {
		return "BANKRUPT"
	}
	if underLiquidation {
		return "LIQUIDATING"
	}
	return "ACTIVE"
}

// formatAddress formats an address as a single string
func formatAddress(addr *Address) string {
	if addr == nil {
		return ""
	}

	parts := make([]string, 0, 4)
	for _, line := range addr.AddressLines {
		if line != "" {
			parts = append(parts, line)
		}
	}
	if addr.PostalCode != "" || addr.PostalPlace != "" {
		parts = append(parts, strings.TrimSpace(addr.PostalCode+" "+addr.PostalPlace))
	}
	if addr.Country != "" && addr.CountryCode != "NO" {
		parts = append(parts, addr.Country)
	}

	return strings.Join(parts, ", ")
}
