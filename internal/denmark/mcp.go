package denmark

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
)

// MCP Tool wrapper methods
// These methods wrap the client methods with Args/Result types for MCP integration.

// SearchCompaniesMCP is the MCP wrapper for SearchCompany
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	query := strings.TrimSpace(args.Query)
	if err := ValidateSearchQuery(query); err != nil {
		return SearchCompaniesResult{}, err
	}

	company, err := c.SearchCompany(ctx, query)
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

// SearchByPhoneMCP is the MCP wrapper for SearchByPhone
func (c *Client) SearchByPhoneMCP(ctx context.Context, args SearchByPhoneArgs) (SearchByPhoneResult, error) {
	phone := strings.TrimSpace(args.Phone)
	if phone == "" {
		return SearchByPhoneResult{}, fmt.Errorf("phone number is required")
	}

	company, err := c.SearchByPhone(ctx, phone)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return SearchByPhoneResult{
				Found:   false,
				Message: "No company found with phone number: " + args.Phone,
			}, nil
		}
		return SearchByPhoneResult{}, err
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

	return SearchByPhoneResult{
		Company: summary,
		Found:   true,
	}, nil
}

// GetByPNumberMCP is the MCP wrapper for GetByPNumber
func (c *Client) GetByPNumberMCP(ctx context.Context, args GetByPNumberArgs) (GetByPNumberResult, error) {
	pnumber := strings.TrimSpace(args.PNumber)
	if pnumber == "" {
		return GetByPNumberResult{}, fmt.Errorf("P-number is required")
	}

	company, err := c.GetByPNumber(ctx, pnumber)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return GetByPNumberResult{
				Found:   false,
				Message: "No company found with P-number: " + args.PNumber,
			}, nil
		}
		return GetByPNumberResult{}, err
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

	return GetByPNumberResult{
		Company: summary,
		Found:   true,
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
