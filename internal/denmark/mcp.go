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

// toCompanySummary maps a CVR company record to the shared summary shape.
func toCompanySummary(company *Company) *CompanySummary {
	return &CompanySummary{
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
}

// companyLookup describes the outcome of a single-company lookup shared by
// the search-style MCP wrappers: either a summary, a polite not-found
// message, or an error.
type companyLookup struct {
	Company *CompanySummary
	Found   bool
	Message string
}

// lookupCompany runs lookup and maps the result into the shared
// found/summary/message shape, treating not-found as a non-error outcome.
func lookupCompany(lookup func() (*Company, error), notFoundMsg string) (companyLookup, error) {
	company, err := lookup()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return companyLookup{Found: false, Message: notFoundMsg}, nil
		}
		return companyLookup{}, err
	}
	return companyLookup{Company: toCompanySummary(company), Found: true}, nil
}

// SearchCompaniesMCP is the MCP wrapper for SearchCompany
func (c *Client) SearchCompaniesMCP(ctx context.Context, args SearchCompaniesArgs) (SearchCompaniesResult, error) {
	query := strings.TrimSpace(args.Query)
	if err := ValidateSearchQuery(query); err != nil {
		return SearchCompaniesResult{}, err
	}

	outcome, err := lookupCompany(func() (*Company, error) {
		return c.SearchCompany(ctx, query)
	}, "No company found matching: "+args.Query)
	if err != nil {
		return SearchCompaniesResult{}, err
	}

	return SearchCompaniesResult(outcome), nil
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

// Default and max page sizes for Denmark production units
const (
	DefaultProductionUnitsPageSize = 20
	MaxProductionUnitsPageSize     = 100
)

// pageWindow describes one page of a paginated listing: the normalized
// page/size, the total page count, and the slice bounds into the full list.
type pageWindow struct {
	page       int
	size       int
	totalPages int
	start      int
	end        int
}

// productionUnitsWindow normalizes page/size against the configured defaults
// and limits, then computes the slice bounds over total items.
func productionUnitsWindow(page, size, total int) pageWindow {
	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = DefaultProductionUnitsPageSize
	}
	if size > MaxProductionUnitsPageSize {
		size = MaxProductionUnitsPageSize
	}

	totalPages := (total + size - 1) / size // ceiling division
	if totalPages == 0 {
		totalPages = 1
	}

	w := pageWindow{page: page, size: size, totalPages: totalPages}
	w.start = min(page*size, total)
	w.end = min(w.start+size, total)
	return w
}

// employeesString renders the CVR API's polymorphic employees field (number
// or string, depending on the record) as a string.
func employeesString(v any) string {
	switch v := v.(type) {
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	}
	return ""
}

// toProductionUnitSummary converts a ProductionUnit to its MCP summary shape.
func toProductionUnitSummary(pu ProductionUnit) ProductionUnitSummary {
	return ProductionUnitSummary{
		PNumber:      strconv.FormatInt(pu.PNumber, 10),
		Name:         pu.Name,
		Address:      pu.Address,
		City:         pu.City,
		Zipcode:      pu.Zipcode,
		IsMain:       pu.Main,
		Employees:    employeesString(pu.Employees),
		IndustryDesc: pu.IndustryDesc,
	}
}

// GetProductionUnitsMCP gets production units for a company with pagination
func (c *Client) GetProductionUnitsMCP(ctx context.Context, args GetProductionUnitsArgs) (GetProductionUnitsResult, error) {
	if err := ValidateCVR(args.CVR); err != nil {
		return GetProductionUnitsResult{}, err
	}

	company, err := c.GetCompany(ctx, args.CVR)
	if err != nil {
		return GetProductionUnitsResult{}, err
	}

	totalUnits := len(company.ProductionUnits)
	w := productionUnitsWindow(args.Page, args.Size, totalUnits)

	units := make([]ProductionUnitSummary, 0, w.end-w.start)
	for _, pu := range company.ProductionUnits[w.start:w.end] {
		units = append(units, toProductionUnitSummary(pu))
	}

	return GetProductionUnitsResult{
		ProductionUnits: units,
		TotalResults:    totalUnits,
		Page:            w.page,
		Size:            w.size,
		TotalPages:      w.totalPages,
		HasMore:         w.page < w.totalPages-1,
	}, nil
}

// lookupByIdentifier validates a required identifier and runs the shared
// single-company lookup flow. The not-found message echoes the raw
// (untrimmed) identifier the caller supplied.
func lookupByIdentifier(raw, fieldName, notFoundPrefix string, lookup func(string) (*Company, error)) (companyLookup, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return companyLookup{}, fmt.Errorf("%s is required", fieldName)
	}
	return lookupCompany(func() (*Company, error) { return lookup(value) }, notFoundPrefix+raw)
}

// SearchByPhoneMCP is the MCP wrapper for SearchByPhone
func (c *Client) SearchByPhoneMCP(ctx context.Context, args SearchByPhoneArgs) (SearchByPhoneResult, error) {
	outcome, err := lookupByIdentifier(args.Phone, "phone number", "No company found with phone number: ",
		func(v string) (*Company, error) { return c.SearchByPhone(ctx, v) })
	return SearchByPhoneResult(outcome), err
}

// GetByPNumberMCP is the MCP wrapper for GetByPNumber
func (c *Client) GetByPNumberMCP(ctx context.Context, args GetByPNumberArgs) (GetByPNumberResult, error) {
	outcome, err := lookupByIdentifier(args.PNumber, "P-number", "No company found with P-number: ",
		func(v string) (*Company, error) { return c.GetByPNumber(ctx, v) })
	return GetByPNumberResult(outcome), err
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
