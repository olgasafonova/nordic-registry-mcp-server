package finland

import "context"

// SearchCompaniesMCP wraps SearchCompanies for MCP tool handlers
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	if err := ValidateSearchQuery(args.Query); err != nil {
		return SearchCompaniesResult{}, err
	}

	resp, err := c.SearchCompanies(ctx, args)
	if err != nil {
		return SearchCompaniesResult{}, err
	}

	result := SearchCompaniesResult{
		TotalResults: resp.TotalResults,
		Page:         args.Page,
	}

	for _, company := range resp.Companies {
		result.Companies = append(result.Companies, toCompanySummary(company))
	}

	return result, nil
}

// GetCompanyMCP wraps GetCompany for MCP tool handlers
func (c *Client) GetCompanyMCP(ctx context.Context, args GetCompanyArgs) (GetCompanyResult, error) {
	if err := ValidateBusinessID(args.BusinessID); err != nil {
		return GetCompanyResult{}, err
	}

	company, err := c.GetCompany(ctx, args.BusinessID)
	if err != nil {
		return GetCompanyResult{}, err
	}

	// Return full data if requested
	if args.Full {
		return GetCompanyResult{
			Company: toCompanyDetails(company),
		}, nil
	}

	// Default: return summary
	summary := toCompanyDetailSummary(company)
	return GetCompanyResult{Summary: summary}, nil
}

// toCompanyDetailSummary converts a Company to CompanyDetailSummary (compact format)
func toCompanyDetailSummary(c *Company) *CompanyDetailSummary {
	if c == nil {
		return nil
	}

	summary := &CompanyDetailSummary{
		BusinessID:       c.BusinessID.Value,
		RegistrationDate: c.RegistrationDate,
		Status:           statusToDesc(c.Status),
	}

	// Get current name (type=1, no end date)
	for _, name := range c.Names {
		if name.Type == "1" && name.EndDate == "" {
			summary.Name = name.Name
			break
		}
	}
	// Fallback to first name if no current name found
	if summary.Name == "" && len(c.Names) > 0 {
		summary.Name = c.Names[0].Name
	}

	// Get company form
	for _, form := range c.CompanyForms {
		if form.EndDate == "" {
			summary.CompanyForm = form.Type
			if desc := getEnglishDesc(form.Descriptions); desc != "" {
				summary.CompanyForm += " - " + desc
			}
			break
		}
	}

	// Get industry
	if c.MainBusinessLine != nil {
		summary.Industry = c.MainBusinessLine.Type
		if desc := getEnglishDesc(c.MainBusinessLine.Descriptions); desc != "" {
			summary.Industry += " - " + desc
		}
	}

	// Get website
	if c.Website != nil {
		summary.Website = c.Website.URL
	}

	// Get address (prefer street address, type=1)
	for _, addr := range c.Addresses {
		if addr.Type == 1 {
			summary.StreetAddress = formatAddress(addr)
			summary.City = getCity(addr)
			break
		}
	}

	return summary
}

// toCompanySummary converts a Company to CompanySummary
func toCompanySummary(c Company) CompanySummary {
	summary := CompanySummary{
		BusinessID:       c.BusinessID.Value,
		RegistrationDate: c.RegistrationDate,
		Status:           statusToDesc(c.Status),
	}

	// Get current name (type=1, no end date)
	for _, name := range c.Names {
		if name.Type == "1" && name.EndDate == "" {
			summary.Name = name.Name
			break
		}
	}
	// Fallback to first name if no current name found
	if summary.Name == "" && len(c.Names) > 0 {
		summary.Name = c.Names[0].Name
	}

	// Get company form
	for _, form := range c.CompanyForms {
		if form.EndDate == "" {
			summary.CompanyForm = form.Type
			summary.CompanyFormDesc = getEnglishDesc(form.Descriptions)
			break
		}
	}

	// Get industry
	if c.MainBusinessLine != nil {
		summary.IndustryCode = c.MainBusinessLine.Type
		summary.Industry = getEnglishDesc(c.MainBusinessLine.Descriptions)
	}

	// Get website
	if c.Website != nil {
		summary.Website = c.Website.URL
	}

	// Get address (prefer street address, type=1)
	for _, addr := range c.Addresses {
		if addr.Type == 1 {
			summary.StreetAddress = formatAddress(addr)
			summary.PostCode = addr.PostCode
			summary.City = getCity(addr)
			break
		}
	}

	return summary
}

// toCompanyDetails converts a Company to CompanyDetails
func toCompanyDetails(c *Company) *CompanyDetails {
	if c == nil {
		return nil
	}

	details := &CompanyDetails{
		BusinessID:       c.BusinessID.Value,
		BusinessIDDate:   c.BusinessID.RegistrationDate,
		RegistrationDate: c.RegistrationDate,
		Status:           c.Status,
		StatusDesc:       statusToDesc(c.Status),
		LastModified:     c.LastModified,
	}

	// EU ID
	if c.EUID != nil {
		details.EUID = c.EUID.Value
	}

	// Names
	for _, name := range c.Names {
		switch name.Type {
		case "1": // Current or previous name
			if name.EndDate == "" {
				details.Name = name.Name
			} else {
				details.PreviousNames = append(details.PreviousNames, name.Name)
			}
		case "3": // Auxiliary name
			details.AuxiliaryNames = append(details.AuxiliaryNames, name.Name)
		}
	}

	// Company form
	for _, form := range c.CompanyForms {
		if form.EndDate == "" {
			details.CompanyForm = form.Type
			details.CompanyFormDesc = getEnglishDesc(form.Descriptions)
			break
		}
	}

	// Industry
	if c.MainBusinessLine != nil {
		details.IndustryCode = c.MainBusinessLine.Type
		details.Industry = getEnglishDesc(c.MainBusinessLine.Descriptions)
	}

	// Website
	if c.Website != nil {
		details.Website = c.Website.URL
	}

	// Addresses
	for _, addr := range c.Addresses {
		addrSummary := &AddressSummary{
			Street:   formatAddress(addr),
			PostCode: addr.PostCode,
			City:     getCity(addr),
		}
		if addr.Type == 1 {
			details.StreetAddress = addrSummary
		} else if addr.Type == 2 {
			details.PostalAddress = addrSummary
		}
	}

	// Situations (liquidation, bankruptcy, etc.)
	for _, sit := range c.CompanySituations {
		details.Situations = append(details.Situations, SituationInfo{
			Type:      situationTypeToDesc(sit.Type),
			StartDate: sit.RegistrationDate,
			EndDate:   sit.EndDate,
		})
	}

	// Registrations
	for _, entry := range c.RegisteredEntries {
		details.Registrations = append(details.Registrations, RegistrationInfo{
			Register:  getEnglishDesc(entry.RegisterDescriptions),
			Status:    entry.RegistrationStatus,
			Authority: getEnglishDesc(entry.AuthorityDescriptions),
			Date:      entry.RegistrationDate,
		})
	}

	return details
}

// getEnglishDesc finds the English description (languageCode=3) or falls back to Finnish (1)
func getEnglishDesc(descs []Description) string {
	var finnish string
	for _, d := range descs {
		if d.LanguageCode == "3" {
			return d.Description
		}
		if d.LanguageCode == "1" {
			finnish = d.Description
		}
	}
	return finnish
}

// formatAddress creates a formatted street address
func formatAddress(addr Address) string {
	if addr.Street == "" {
		return addr.FreeAddressLine
	}
	result := addr.Street
	if addr.BuildingNumber != "" {
		result += " " + addr.BuildingNumber
	}
	if addr.Entrance != "" {
		result += " " + addr.Entrance
	}
	if addr.ApartmentNumber != "" {
		result += ", " + addr.ApartmentNumber
	}
	return result
}

// getCity extracts city from post offices (prefer Finnish)
func getCity(addr Address) string {
	for _, po := range addr.PostOffices {
		if po.LanguageCode == "1" { // Finnish
			return po.City
		}
	}
	if len(addr.PostOffices) > 0 {
		return addr.PostOffices[0].City
	}
	return ""
}

// statusToDesc converts status code to description
func statusToDesc(status string) string {
	switch status {
	case "1":
		return "Registered"
	case "2":
		return "Active"
	case "3":
		return "Dissolved"
	case "4":
		return "Liquidation"
	case "5":
		return "Bankruptcy"
	default:
		return status
	}
}

// situationTypeToDesc converts situation type to description
func situationTypeToDesc(t string) string {
	switch t {
	case "SANE":
		return "Reorganization"
	case "SELTILA":
		return "Liquidation"
	case "KONK":
		return "Bankruptcy"
	default:
		return t
	}
}
