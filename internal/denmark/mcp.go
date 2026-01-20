package denmark

import (
	"context"
	"errors"
)

// MCP Tool wrapper methods
// These methods wrap the client methods with Args/Result types for MCP integration.

// SearchCompaniesMCP is the MCP wrapper for SearchCompany
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	if err := ValidateSearchQuery(args.Query); err != nil {
		return SearchCompaniesResult{}, err
	}

	company, err := c.SearchCompany(ctx, args.Query)
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			return SearchCompaniesResult{
				Found:   false,
				Message: "No company found matching: " + args.Query,
			}, nil
		}
		return SearchCompaniesResult{}, err
	}

	summary := &CompanySummary{
		CVR:          company.CVR,
		Name:         company.Name,
		Address:      company.Address,
		City:         company.City,
		Zipcode:      company.Zipcode,
		CompanyType:  company.CompanyType,
		IndustryDesc: company.IndustryDesc,
		Employees:    company.Employees,
		StartDate:    company.StartDate,
		Status:       getStatus(company),
		Phone:        company.Phone,
		Email:        company.Email,
	}

	return SearchCompaniesResult{
		Company: summary,
		Found:   true,
	}, nil
}

// GetCompanyMCP is the MCP wrapper for GetCompany
func (c *Client) GetCompanyMCP(ctx context.Context, args GetCompanyArgs) (GetCompanyResult, error) {
	if err := ValidateCVR(args.CVR); err != nil {
		return GetCompanyResult{}, err
	}

	company, err := c.GetCompany(ctx, args.CVR)
	if err != nil {
		return GetCompanyResult{}, err
	}
	return GetCompanyResult{Company: company}, nil
}

// GetProductionUnitsMCP gets production units for a company
func (c *Client) GetProductionUnitsMCP(ctx context.Context, args GetProductionUnitsArgs) (GetProductionUnitsResult, error) {
	if err := ValidateCVR(args.CVR); err != nil {
		return GetProductionUnitsResult{}, err
	}

	company, err := c.GetCompany(ctx, args.CVR)
	if err != nil {
		return GetProductionUnitsResult{}, err
	}

	units := make([]ProductionUnitSummary, 0, len(company.ProductionUnits))
	for _, pu := range company.ProductionUnits {
		units = append(units, ProductionUnitSummary{
			PNumber:      pu.PNumber,
			Name:         pu.Name,
			Address:      pu.Address,
			City:         pu.City,
			Zipcode:      pu.Zipcode,
			IsMain:       pu.Main,
			Employees:    pu.Employees,
			IndustryDesc: pu.IndustryDesc,
		})
	}

	return GetProductionUnitsResult{
		ProductionUnits: units,
		TotalResults:    len(units),
	}, nil
}

// getStatus derives company status from available fields
func getStatus(c *Company) string {
	if c.EndDate != "" {
		return "DISSOLVED"
	}
	if c.CreditEnd != "" {
		return "BANKRUPT"
	}
	return "ACTIVE"
}
