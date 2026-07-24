package finland

import (
	"context"
	"strings"
)

// Default and max page sizes for Finland search
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// SearchCompaniesMCP wraps SearchCompanies for MCP tool handlers
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	query := strings.TrimSpace(args.Query)
	if err := ValidateSearchQuery(query); err != nil {
		return SearchCompaniesResult{}, err
	}
	args.Query = query

	// Apply size defaults and limits
	size := args.Size
	if size <= 0 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}

	resp, err := c.SearchCompanies(ctx, args)
	if err != nil {
		return SearchCompaniesResult{}, err
	}

	result := SearchCompaniesResult{
		TotalResults: resp.TotalResults,
		Page:         args.Page,
		Size:         size,
	}

	// Limit results to requested size
	companies := resp.Companies
	if len(companies) > size {
		companies = companies[:size]
		result.HasMore = true
	} else {
		// Check if there are more pages
		result.HasMore = (args.Page+1)*size < resp.TotalResults
	}

	for _, company := range companies {
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

// currentCompanyName returns the active name (type=1, no end date) for a
// PRH Company, falling back to the first name if no active name is recorded.
func currentCompanyName(c *Company) string {
	for _, name := range c.Names {
		if name.Type == "1" && name.EndDate == "" {
			return name.Name
		}
	}
	if len(c.Names) > 0 {
		return c.Names[0].Name
	}
	return ""
}

// currentCompanyForm returns the active company form (no end date). The two
// returned strings are the type code and its English description; either may
// be empty if not found.
func currentCompanyForm(c *Company) (string, string) {
	for _, form := range c.CompanyForms {
		if form.EndDate == "" {
			return form.Type, getEnglishDesc(form.Descriptions)
		}
	}
	return "", ""
}

// primaryStreetAddress returns the first street address (type=1). The boolean
// reports whether a match was found.
func primaryStreetAddress(c *Company) (Address, bool) {
	for _, addr := range c.Addresses {
		if addr.Type == 1 {
			return addr, true
		}
	}
	return Address{}, false
}

// toCompanyDetailSummary converts a Company to CompanyDetailSummary (compact
// format). It projects the full CompanySummary down to the detail shape,
// collapsing the code/description pairs into single "code - desc" strings.
func toCompanyDetailSummary(c *Company) *CompanyDetailSummary {
	if c == nil {
		return nil
	}

	s := toCompanySummary(*c)
	return &CompanyDetailSummary{
		BusinessID:       s.BusinessID,
		RegistrationDate: s.RegistrationDate,
		Status:           s.Status,
		Name:             s.Name,
		CompanyForm:      formatCodeAndDesc(s.CompanyForm, s.CompanyFormDesc),
		Industry:         formatCodeAndDesc(s.IndustryCode, s.Industry),
		Website:          s.Website,
		StreetAddress:    s.StreetAddress,
		City:             s.City,
	}
}

// formatCodeAndDesc returns "code - desc" when desc is non-empty, otherwise the
// bare code. Centralizes the PRH code+English-description rendering pattern.
func formatCodeAndDesc(code, desc string) string {
	if desc != "" {
		return code + " - " + desc
	}
	return code
}

// toCompanySummary converts a Company to CompanySummary
func toCompanySummary(c Company) CompanySummary {
	summary := CompanySummary{
		BusinessID:       c.BusinessID.Value,
		RegistrationDate: c.RegistrationDate,
		Status:           statusToDesc(c.Status),
		Name:             currentCompanyName(&c),
	}

	summary.CompanyForm, summary.CompanyFormDesc = currentCompanyForm(&c)

	if c.MainBusinessLine != nil {
		summary.IndustryCode = c.MainBusinessLine.Type
		summary.Industry = getEnglishDesc(c.MainBusinessLine.Descriptions)
	}

	if c.Website != nil {
		summary.Website = c.Website.URL
	}

	if addr, ok := primaryStreetAddress(&c); ok {
		summary.StreetAddress = formatAddress(addr)
		summary.PostCode = addr.PostCode
		summary.City = getCity(addr)
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

	if c.EUID != nil {
		details.EUID = c.EUID.Value
	}

	applyDetailNames(details, c.Names)

	if code, desc := currentCompanyForm(c); code != "" {
		details.CompanyForm = code
		details.CompanyFormDesc = desc
	}

	if c.MainBusinessLine != nil {
		details.IndustryCode = c.MainBusinessLine.Type
		details.Industry = getEnglishDesc(c.MainBusinessLine.Descriptions)
	}

	if c.Website != nil {
		details.Website = c.Website.URL
	}

	applyDetailAddresses(details, c.Addresses)
	details.Situations = collectSituations(c.CompanySituations)
	details.Registrations = collectRegistrations(c.RegisteredEntries)

	return details
}

// applyDetailNames classifies PRH name entries into the current name, previous
// names (ended type=1 entries), and auxiliary names (type=3). Other types are
// silently dropped, matching the original behavior.
func applyDetailNames(details *CompanyDetails, names []CompanyName) {
	for _, name := range names {
		switch name.Type {
		case "1":
			if name.EndDate == "" {
				details.Name = name.Name
			} else {
				details.PreviousNames = append(details.PreviousNames, name.Name)
			}
		case "3":
			details.AuxiliaryNames = append(details.AuxiliaryNames, name.Name)
		}
	}
}

// applyDetailAddresses routes type=1 (street) and type=2 (postal) PRH
// addresses to the corresponding CompanyDetails slots.
func applyDetailAddresses(details *CompanyDetails, addresses []Address) {
	for _, addr := range addresses {
		addrSummary := &AddressSummary{
			Street:   formatAddress(addr),
			PostCode: addr.PostCode,
			City:     getCity(addr),
		}
		switch addr.Type {
		case 1:
			details.StreetAddress = addrSummary
		case 2:
			details.PostalAddress = addrSummary
		}
	}
}

// collectSituations flattens PRH liquidation / bankruptcy / restructuring
// records into SituationInfo entries.
func collectSituations(situations []CompanySituation) []SituationInfo {
	var out []SituationInfo
	for _, sit := range situations {
		out = append(out, SituationInfo{
			Type:      situationTypeToDesc(sit.Type),
			StartDate: sit.RegistrationDate,
			EndDate:   sit.EndDate,
		})
	}
	return out
}

// collectRegistrations flattens PRH register memberships (trade register, VAT
// register, etc.) into RegistrationInfo entries.
func collectRegistrations(entries []RegisteredEntry) []RegistrationInfo {
	var out []RegistrationInfo
	for _, entry := range entries {
		out = append(out, RegistrationInfo{
			Register:  getEnglishDesc(entry.RegisterDescriptions),
			Status:    entry.RegistrationStatus,
			Authority: getEnglishDesc(entry.AuthorityDescriptions),
			Date:      entry.RegistrationDate,
		})
	}
	return out
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
