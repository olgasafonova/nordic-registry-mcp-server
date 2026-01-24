package norway

import (
	"context"
	"strings"
)

// MCP Tool wrapper methods
// These methods wrap the client methods with Args/Result types for MCP integration.

// SearchCompaniesMCP is the MCP wrapper for SearchCompanies
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	if err := ValidateSearchQuery(args.Query); err != nil {
		return SearchCompaniesResult{}, err
	}
	if args.Size > 0 {
		if err := ValidateSize(args.Size); err != nil {
			return SearchCompaniesResult{}, err
		}
	}

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
			Status:             getStatus(co.Bankrupt, co.UnderLiquidation),
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
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetCompanyResult{}, err
	}

	company, err := c.GetCompany(ctx, args.OrgNumber)
	if err != nil {
		return GetCompanyResult{}, err
	}

	// Return full data if requested
	if args.Full {
		return GetCompanyResult{Company: company}, nil
	}

	// Default: return summary
	summary := &CompanyDetailSummary{
		OrganizationNumber: company.OrganizationNumber,
		Name:               company.Name,
		RegistrationDate:   company.RegistrationDate,
		EmployeeCount:      company.EmployeeCount,
		Website:            company.Website,
		VATRegistered:      company.RegisteredInVAT,
		Status:             getStatus(company.Bankrupt, company.UnderLiquidation),
		Bankrupt:           company.Bankrupt,
		UnderLiquidation:   company.UnderLiquidation,
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

	return GetCompanyResult{Summary: summary}, nil
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
	if err := ValidateOrgNumber(args.ParentOrgNumber); err != nil {
		return GetSubUnitsResult{}, err
	}

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
	if err := ValidateSearchQuery(args.Query); err != nil {
		return SearchSubUnitsResult{}, err
	}
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

// GetSignatureRightsMCP is the MCP wrapper for getting signature rights
func (c *Client) GetSignatureRightsMCP(ctx context.Context, args GetSignatureRightsArgs) (GetSignatureRightsResult, error) {
	if err := ValidateOrgNumber(args.OrgNumber); err != nil {
		return GetSignatureRightsResult{}, err
	}

	// Get roles data - signature rights are part of roles
	resp, err := c.GetRoles(ctx, args.OrgNumber)
	if err != nil {
		return GetSignatureRightsResult{}, err
	}

	result := GetSignatureRightsResult{
		OrganizationNumber: args.OrgNumber,
		SignatureRights:    []SignatureRight{},
		Prokura:            []SignatureRight{},
	}

	// Extract signature-related roles (SIGN = signaturrett, PROK = prokura)
	for _, rg := range resp.RoleGroups {
		for _, r := range rg.Roles {
			if r.Resigned {
				continue
			}

			sr := SignatureRight{
				Type:        r.Type.Code,
				Description: r.Type.Description,
			}

			if r.Person != nil {
				sr.Name = r.Person.Name.FullName()
				sr.BirthDate = r.Person.BirthDate
			}
			if r.Entity != nil {
				sr.EntityOrgNr = r.Entity.OrganizationNumber
				if len(r.Entity.Name) > 0 {
					sr.Name = strings.Join(r.Entity.Name, " ")
				}
			}

			switch r.Type.Code {
			case "SIGN":
				result.SignatureRights = append(result.SignatureRights, sr)
			case "PROK":
				result.Prokura = append(result.Prokura, sr)
			}
		}
	}

	// Build summary
	var summary strings.Builder
	if len(result.SignatureRights) > 0 {
		summary.WriteString("Signature rights: ")
		names := make([]string, len(result.SignatureRights))
		for i, sr := range result.SignatureRights {
			names[i] = sr.Name
		}
		summary.WriteString(strings.Join(names, ", "))
	}
	if len(result.Prokura) > 0 {
		if summary.Len() > 0 {
			summary.WriteString(". ")
		}
		summary.WriteString("Prokura: ")
		names := make([]string, len(result.Prokura))
		for i, sr := range result.Prokura {
			names[i] = sr.Name
		}
		summary.WriteString(strings.Join(names, ", "))
	}
	if summary.Len() == 0 {
		summary.WriteString("No signature rights or prokura found")
	}
	result.Summary = summary.String()

	return result, nil
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
