package norway

import (
	"context"
	"strings"
)

// MCP Tool wrapper methods
// These methods wrap the client methods with Args/Result types for MCP integration.

// SearchCompaniesMCP is the MCP wrapper for SearchCompanies
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	opts := &SearchOptions{
		Page:            args.Page,
		Size:            args.Size,
		OrgForm:         args.OrgForm,
		Municipality:    args.Municipality,
		RegisteredInVAT: args.RegisteredInVAT,
		Bankrupt:        args.Bankrupt,
	}

	resp, err := c.SearchCompanies(ctx, args.Query, opts)
	if err != nil {
		return SearchCompaniesResult{}, err
	}

	// Convert to summary format
	companies := make([]CompanySummary, 0, len(resp.Embedded.Companies))
	for _, co := range resp.Embedded.Companies {
		summary := CompanySummary{
			OrganizationNumber: co.OrganizationNumber,
			Name:               co.Name,
			Bankrupt:           co.Bankrupt,
			UnderLiquidation:   co.UnderLiquidation,
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
		companies = append(companies, summary)
	}

	return SearchCompaniesResult{
		Companies:    companies,
		TotalResults: resp.Page.TotalElements,
		Page:         resp.Page.Number,
		TotalPages:   resp.Page.TotalPages,
	}, nil
}

// GetCompanyMCP is the MCP wrapper for GetCompany
func (c *Client) GetCompanyMCP(ctx context.Context, args GetCompanyArgs) (GetCompanyResult, error) {
	company, err := c.GetCompany(ctx, args.OrgNumber)
	if err != nil {
		return GetCompanyResult{}, err
	}
	return GetCompanyResult{Company: company}, nil
}

// GetRolesMCP is the MCP wrapper for GetRoles
func (c *Client) GetRolesMCP(ctx context.Context, args GetRolesArgs) (GetRolesResult, error) {
	resp, err := c.GetRoles(ctx, args.OrgNumber)
	if err != nil {
		return GetRolesResult{}, err
	}

	// Convert to summary format
	groups := make([]RoleGroupSummary, 0, len(resp.RoleGroups))
	for _, rg := range resp.RoleGroups {
		group := RoleGroupSummary{
			Type:         rg.Type.Code,
			Description:  rg.Type.Description,
			LastModified: rg.LastModified,
			Roles:        make([]RoleSummary, 0, len(rg.Roles)),
		}

		for _, r := range rg.Roles {
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
			group.Roles = append(group.Roles, role)
		}
		groups = append(groups, group)
	}

	return GetRolesResult{RoleGroups: groups}, nil
}

// GetSubUnitsMCP is the MCP wrapper for GetSubUnits
func (c *Client) GetSubUnitsMCP(ctx context.Context, args GetSubUnitsArgs) (GetSubUnitsResult, error) {
	resp, err := c.GetSubUnits(ctx, args.ParentOrgNumber)
	if err != nil {
		return GetSubUnitsResult{}, err
	}

	// Convert to summary format
	subunits := make([]SubUnitSummary, 0, len(resp.Embedded.SubUnits))
	for _, su := range resp.Embedded.SubUnits {
		summary := SubUnitSummary{
			OrganizationNumber: su.OrganizationNumber,
			Name:               su.Name,
			ParentOrgNumber:    su.ParentOrganizationNumber,
			EmployeeCount:      su.EmployeeCount,
		}
		if su.BusinessAddress != nil {
			summary.BusinessAddress = formatAddress(su.BusinessAddress)
		}
		subunits = append(subunits, summary)
	}

	return GetSubUnitsResult{
		SubUnits:     subunits,
		TotalResults: resp.Page.TotalElements,
	}, nil
}

// GetSubUnitMCP is the MCP wrapper for GetSubUnit
func (c *Client) GetSubUnitMCP(ctx context.Context, args GetSubUnitArgs) (GetSubUnitResult, error) {
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
