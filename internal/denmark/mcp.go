package denmark

import (
	"context"
	"strconv"

	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
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
		if apierrors.IsNotFound(err) {
			return SearchCompaniesResult{
				Found:   false,
				Message: "No company found matching: " + args.Query,
			}, nil
		}
		return SearchCompaniesResult{}, err
	}

	summary := &CompanySummary{
		CVR:          strconv.Itoa(company.CVR),
		Name:         company.Name,
		Address:      company.Address,
		City:         company.City,
		Zipcode:      company.Zipcode,
		CompanyType:  company.CompanyType,
		IndustryDesc: company.IndustryDesc,
		Employees:    strconv.Itoa(company.Employees),
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

	// Return full data if requested
	if args.Full {
		return GetCompanyResult{Company: company}, nil
	}

	// Default: return summary
	summary := &CompanyDetailSummary{
		CVR:             strconv.Itoa(company.CVR),
		Name:            company.Name,
		Address:         company.Address,
		City:            company.City,
		Zipcode:         company.Zipcode,
		CompanyType:     company.CompanyType,
		Industry:        company.IndustryDesc,
		Employees:       company.Employees,
		StartDate:       company.StartDate,
		Status:          getStatus(company),
		Phone:           company.Phone,
		Email:           company.Email,
		ProductionUnits: len(company.ProductionUnits),
	}

	return GetCompanyResult{Summary: summary}, nil
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
		employees := ""
		switch v := pu.Employees.(type) {
		case float64:
			employees = strconv.Itoa(int(v))
		case int:
			employees = strconv.Itoa(v)
		case string:
			employees = v
		}
		units = append(units, ProductionUnitSummary{
			PNumber:      strconv.FormatInt(pu.PNumber, 10),
			Name:         pu.Name,
			Address:      pu.Address,
			City:         pu.City,
			Zipcode:      pu.Zipcode,
			IsMain:       pu.Main,
			Employees:    employees,
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
	if c.CreditEnd {
		return "BANKRUPT"
	}
	return "ACTIVE"
}
