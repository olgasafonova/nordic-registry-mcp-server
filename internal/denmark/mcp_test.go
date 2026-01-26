package denmark

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
)

// =============================================================================
// Client Option Tests
// =============================================================================

func TestWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(WithLogger(logger))
	defer client.Close()

	if client.Logger != logger {
		t.Error("custom logger was not set")
	}
}

func TestWithCache(t *testing.T) {
	cache := infra.NewCache(500)
	defer cache.Close()

	client := NewClient(WithCache(cache))
	defer client.Close()

	if client.Cache != cache {
		t.Error("custom cache was not set")
	}
}

// =============================================================================
// SearchCompaniesMCP Tests
// =============================================================================

func TestSearchCompaniesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "Novo Nordisk" {
			t.Errorf("unexpected search param: %s", r.URL.Query().Get("search"))
		}

		company := Company{
			CVR:          10150817,
			Name:         "NOVO NORDISK A/S",
			Address:      "Novo Allé 1",
			Zipcode:      "2880",
			City:         "Bagsværd",
			CompanyType:  "Aktieselskab",
			IndustryDesc: "Fremstilling af farmaceutiske præparater",
			Employees:    45000,
			StartDate:    "01/01 - 1925",
			Phone:        "44448888",
			Email:        "info@novonordisk.com",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: "Novo Nordisk"})
	if err != nil {
		t.Fatalf("SearchCompaniesMCP failed: %v", err)
	}

	if !result.Found {
		t.Error("Expected Found=true")
	}
	if result.Company == nil {
		t.Fatal("Company is nil")
	}
	if result.Company.CVR != "10150817" {
		t.Errorf("CVR = %q, want %q", result.Company.CVR, "10150817")
	}
	if result.Company.Name != "NOVO NORDISK A/S" {
		t.Errorf("Name = %q, want %q", result.Company.Name, "NOVO NORDISK A/S")
	}
	if result.Company.Employees != "45000" {
		t.Errorf("Employees = %q, want %q", result.Company.Employees, "45000")
	}
	if result.Company.Status != "ACTIVE" {
		t.Errorf("Status = %q, want %q", result.Company.Status, "ACTIVE")
	}
}

func TestSearchCompaniesMCP_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CVR API returns empty object when no results
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: "nonexistent xyz123"})
	if err != nil {
		t.Fatalf("SearchCompaniesMCP returned error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found=false for no results")
	}
	if result.Company != nil {
		t.Error("Company should be nil for no results")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestSearchCompaniesMCP_EmptyQuery(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: ""})
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchCompaniesMCP_ShortQuery(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: "A"})
	if err == nil {
		t.Error("Expected error for single character query")
	}
}

func TestSearchCompaniesMCP_WhitespaceQuery(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: "   "})
	if err == nil {
		t.Error("Expected error for whitespace-only query")
	}
}

func TestSearchCompaniesMCP_DissolvedCompany(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:     12345678,
			Name:    "DISSOLVED COMPANY",
			EndDate: "2023-12-31",
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: "Dissolved"})
	if err != nil {
		t.Fatalf("SearchCompaniesMCP failed: %v", err)
	}

	if result.Company.Status != "DISSOLVED" {
		t.Errorf("Status = %q, want %q", result.Company.Status, "DISSOLVED")
	}
}

func TestSearchCompaniesMCP_BankruptCompany(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:       12345678,
			Name:      "BANKRUPT COMPANY",
			CreditEnd: true,
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompaniesMCP(ctx(), SearchCompaniesArgs{Query: "Bankrupt"})
	if err != nil {
		t.Fatalf("SearchCompaniesMCP failed: %v", err)
	}

	if result.Company.Status != "BANKRUPT" {
		t.Errorf("Status = %q, want %q", result.Company.Status, "BANKRUPT")
	}
}

// =============================================================================
// GetCompanyMCP Tests
// =============================================================================

func TestGetCompanyMCP_Summary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:          10150817,
			Name:         "NOVO NORDISK A/S",
			Address:      "Novo Allé 1",
			Zipcode:      "2880",
			City:         "Bagsværd",
			CompanyType:  "Aktieselskab",
			IndustryDesc: "Fremstilling af farmaceutiske præparater",
			Employees:    45000,
			StartDate:    "01/01 - 1925",
			Phone:        "44448888",
			Email:        "info@novonordisk.com",
			ProductionUnits: []ProductionUnit{
				{PNumber: 1234567890, Main: true, Name: "Hovedkontor"},
				{PNumber: 1234567891, Main: false, Name: "Branch"},
			},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompanyMCP(ctx(), GetCompanyArgs{CVR: "10150817"})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Company != nil {
		t.Error("Company should be nil when Full=false")
	}
	if result.Summary == nil {
		t.Fatal("Summary is nil")
	}
	if result.Summary.CVR != "10150817" {
		t.Errorf("CVR = %q, want %q", result.Summary.CVR, "10150817")
	}
	if result.Summary.ProductionUnits != 2 {
		t.Errorf("ProductionUnits = %d, want %d", result.Summary.ProductionUnits, 2)
	}
	if result.Summary.Employees != 45000 {
		t.Errorf("Employees = %d, want %d", result.Summary.Employees, 45000)
	}
}

func TestGetCompanyMCP_Full(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:       10150817,
			Name:      "NOVO NORDISK A/S",
			Employees: 45000,
			ProductionUnits: []ProductionUnit{
				{PNumber: 1234567890, Main: true},
			},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompanyMCP(ctx(), GetCompanyArgs{CVR: "10150817", Full: true})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Summary != nil {
		t.Error("Summary should be nil when Full=true")
	}
	if result.Company == nil {
		t.Fatal("Company is nil")
	}
	if result.Company.CVR != 10150817 {
		t.Errorf("CVR = %d, want %d", result.Company.CVR, 10150817)
	}
}

func TestGetCompanyMCP_EmptyCVR(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetCompanyMCP(ctx(), GetCompanyArgs{CVR: ""})
	if err == nil {
		t.Error("Expected error for empty CVR")
	}
}

func TestGetCompanyMCP_InvalidCVR(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetCompanyMCP(ctx(), GetCompanyArgs{CVR: "123"})
	if err == nil {
		t.Error("Expected error for invalid CVR")
	}
}

func TestGetCompanyMCP_CVRWithDKPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that the CVR is normalized (DK prefix removed)
		if r.URL.Query().Get("vat") != "10150817" {
			t.Errorf("CVR not normalized: %s", r.URL.Query().Get("vat"))
		}
		company := Company{CVR: 10150817, Name: "Test"}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompanyMCP(ctx(), GetCompanyArgs{CVR: "DK10150817"})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Summary == nil {
		t.Error("Summary should not be nil")
	}
}

// =============================================================================
// GetProductionUnitsMCP Tests
// =============================================================================

func TestGetProductionUnitsMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:  10150817,
			Name: "NOVO NORDISK A/S",
			ProductionUnits: []ProductionUnit{
				{PNumber: 1111111111, Main: true, Name: "Hovedkontor", Address: "Street 1", City: "Copenhagen", Zipcode: "1000", Employees: 5000, IndustryDesc: "Software"},
				{PNumber: 2222222222, Main: false, Name: "Branch 1", Address: "Street 2", City: "Aarhus", Zipcode: "8000", Employees: 1000, IndustryDesc: "Software"},
				{PNumber: 3333333333, Main: false, Name: "Branch 2", Address: "Street 3", City: "Odense", Zipcode: "5000", Employees: 500, IndustryDesc: "Software"},
			},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817"})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.TotalResults != 3 {
		t.Errorf("TotalResults = %d, want %d", result.TotalResults, 3)
	}
	if len(result.ProductionUnits) != 3 {
		t.Errorf("ProductionUnits count = %d, want %d", len(result.ProductionUnits), 3)
	}
	if result.ProductionUnits[0].PNumber != "1111111111" {
		t.Errorf("PNumber = %q, want %q", result.ProductionUnits[0].PNumber, "1111111111")
	}
	if !result.ProductionUnits[0].IsMain {
		t.Error("First unit should be main")
	}
	if result.ProductionUnits[1].IsMain {
		t.Error("Second unit should not be main")
	}
}

func TestGetProductionUnitsMCP_Pagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create 25 production units
		units := make([]ProductionUnit, 25)
		for i := range 25 {
			units[i] = ProductionUnit{
				PNumber: int64(1000000000 + i),
				Main:    i == 0,
				Name:    "Unit " + string(rune('A'+i)),
			}
		}
		company := Company{CVR: 10150817, ProductionUnits: units}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// Test first page with default size (20)
	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817"})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.TotalResults != 25 {
		t.Errorf("TotalResults = %d, want %d", result.TotalResults, 25)
	}
	if len(result.ProductionUnits) != 20 {
		t.Errorf("ProductionUnits count = %d, want %d", len(result.ProductionUnits), 20)
	}
	if result.Page != 0 {
		t.Errorf("Page = %d, want %d", result.Page, 0)
	}
	if result.Size != 20 {
		t.Errorf("Size = %d, want %d", result.Size, 20)
	}
	if result.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want %d", result.TotalPages, 2)
	}
	if !result.HasMore {
		t.Error("HasMore should be true")
	}

	// Test second page
	result2, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Page: 1})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP page 1 failed: %v", err)
	}

	if len(result2.ProductionUnits) != 5 {
		t.Errorf("Page 1 ProductionUnits count = %d, want %d", len(result2.ProductionUnits), 5)
	}
	if result2.HasMore {
		t.Error("HasMore should be false on last page")
	}
}

func TestGetProductionUnitsMCP_CustomPageSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		units := make([]ProductionUnit, 10)
		for i := range 10 {
			units[i] = ProductionUnit{PNumber: int64(1000000000 + i)}
		}
		company := Company{CVR: 10150817, ProductionUnits: units}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Size: 5})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if len(result.ProductionUnits) != 5 {
		t.Errorf("ProductionUnits count = %d, want %d", len(result.ProductionUnits), 5)
	}
	if result.Size != 5 {
		t.Errorf("Size = %d, want %d", result.Size, 5)
	}
	if result.TotalPages != 2 {
		t.Errorf("TotalPages = %d, want %d", result.TotalPages, 2)
	}
}

func TestGetProductionUnitsMCP_MaxPageSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		units := make([]ProductionUnit, 150)
		for i := range 150 {
			units[i] = ProductionUnit{PNumber: int64(1000000000 + i)}
		}
		company := Company{CVR: 10150817, ProductionUnits: units}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// Request size of 200, should be capped to 100
	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Size: 200})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.Size != 100 {
		t.Errorf("Size should be capped to 100, got %d", result.Size)
	}
	if len(result.ProductionUnits) != 100 {
		t.Errorf("ProductionUnits count = %d, want %d", len(result.ProductionUnits), 100)
	}
}

func TestGetProductionUnitsMCP_NegativePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:             10150817,
			ProductionUnits: []ProductionUnit{{PNumber: 1111111111}},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// Negative page should be treated as 0
	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Page: -1})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.Page != 0 {
		t.Errorf("Page = %d, want %d", result.Page, 0)
	}
}

func TestGetProductionUnitsMCP_EmptyUnits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{CVR: 10150817, ProductionUnits: []ProductionUnit{}}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817"})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.TotalResults != 0 {
		t.Errorf("TotalResults = %d, want %d", result.TotalResults, 0)
	}
	if len(result.ProductionUnits) != 0 {
		t.Errorf("ProductionUnits count = %d, want %d", len(result.ProductionUnits), 0)
	}
	if result.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want %d (empty list should have 1 page)", result.TotalPages, 1)
	}
}

func TestGetProductionUnitsMCP_EmployeesVariants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR: 10150817,
			ProductionUnits: []ProductionUnit{
				{PNumber: 1111111111, Employees: 100},         // int
				{PNumber: 2222222222, Employees: "1-4"},       // string
				{PNumber: 3333333333, Employees: float64(50)}, // float64
				{PNumber: 4444444444, Employees: nil},         // nil
			},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817"})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	// Note: JSON unmarshals numbers as float64
	if result.ProductionUnits[0].Employees != "100" {
		t.Errorf("Employees[0] = %q, want %q", result.ProductionUnits[0].Employees, "100")
	}
	if result.ProductionUnits[1].Employees != "1-4" {
		t.Errorf("Employees[1] = %q, want %q", result.ProductionUnits[1].Employees, "1-4")
	}
	if result.ProductionUnits[2].Employees != "50" {
		t.Errorf("Employees[2] = %q, want %q", result.ProductionUnits[2].Employees, "50")
	}
	if result.ProductionUnits[3].Employees != "" {
		t.Errorf("Employees[3] = %q, want empty", result.ProductionUnits[3].Employees)
	}
}

func TestGetProductionUnitsMCP_InvalidCVR(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "invalid"})
	if err == nil {
		t.Error("Expected error for invalid CVR")
	}
}

// =============================================================================
// SearchByPhoneMCP Tests
// =============================================================================

func TestSearchByPhoneMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("phone") != "44448888" {
			t.Errorf("unexpected phone param: %s", r.URL.Query().Get("phone"))
		}

		company := Company{
			CVR:       10150817,
			Name:      "NOVO NORDISK A/S",
			Phone:     "44448888",
			Employees: 45000,
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchByPhoneMCP(ctx(), SearchByPhoneArgs{Phone: "44448888"})
	if err != nil {
		t.Fatalf("SearchByPhoneMCP failed: %v", err)
	}

	if !result.Found {
		t.Error("Expected Found=true")
	}
	if result.Company == nil {
		t.Fatal("Company is nil")
	}
	if result.Company.Phone != "44448888" {
		t.Errorf("Phone = %q, want %q", result.Company.Phone, "44448888")
	}
}

func TestSearchByPhoneMCP_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchByPhoneMCP(ctx(), SearchByPhoneArgs{Phone: "12345678"})
	if err != nil {
		t.Fatalf("SearchByPhoneMCP returned error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found=false")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestSearchByPhoneMCP_EmptyPhone(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchByPhoneMCP(ctx(), SearchByPhoneArgs{Phone: ""})
	if err == nil {
		t.Error("Expected error for empty phone")
	}
}

func TestSearchByPhoneMCP_WhitespacePhone(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchByPhoneMCP(ctx(), SearchByPhoneArgs{Phone: "   "})
	if err == nil {
		t.Error("Expected error for whitespace-only phone")
	}
}

func TestSearchByPhoneMCP_InvalidPhone(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchByPhoneMCP(ctx(), SearchByPhoneArgs{Phone: "12345"})
	if err == nil {
		t.Error("Expected error for too short phone number")
	}
}

func TestSearchByPhoneMCP_PhoneNormalization(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedNormal string
	}{
		{"with spaces", "44 44 88 88", "44448888"},
		{"with dashes", "44-44-88-88", "44448888"},
		{"with country code", "+4544448888", "44448888"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPhone string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPhone = r.URL.Query().Get("phone")
				company := Company{CVR: 12345678, Name: "Test"}
				_ = json.NewEncoder(w).Encode(company)
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL))
			defer client.Close()

			_, _ = client.SearchByPhoneMCP(ctx(), SearchByPhoneArgs{Phone: tt.input})
			if receivedPhone != tt.expectedNormal {
				t.Errorf("phone param = %q, want %q", receivedPhone, tt.expectedNormal)
			}
		})
	}
}

// =============================================================================
// GetByPNumberMCP Tests
// =============================================================================

func TestGetByPNumberMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("produ") != "1234567890" {
			t.Errorf("unexpected produ param: %s", r.URL.Query().Get("produ"))
		}

		company := Company{
			CVR:       10150817,
			Name:      "NOVO NORDISK A/S",
			Employees: 45000,
			ProductionUnits: []ProductionUnit{
				{PNumber: 1234567890, Main: true, Name: "Hovedkontor"},
			},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetByPNumberMCP(ctx(), GetByPNumberArgs{PNumber: "1234567890"})
	if err != nil {
		t.Fatalf("GetByPNumberMCP failed: %v", err)
	}

	if !result.Found {
		t.Error("Expected Found=true")
	}
	if result.Company == nil {
		t.Fatal("Company is nil")
	}
	if result.Company.CVR != "10150817" {
		t.Errorf("CVR = %q, want %q", result.Company.CVR, "10150817")
	}
}

func TestGetByPNumberMCP_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetByPNumberMCP(ctx(), GetByPNumberArgs{PNumber: "0000000000"})
	if err != nil {
		t.Fatalf("GetByPNumberMCP returned error: %v", err)
	}

	if result.Found {
		t.Error("Expected Found=false")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestGetByPNumberMCP_EmptyPNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetByPNumberMCP(ctx(), GetByPNumberArgs{PNumber: ""})
	if err == nil {
		t.Error("Expected error for empty P-number")
	}
}

func TestGetByPNumberMCP_WhitespacePNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetByPNumberMCP(ctx(), GetByPNumberArgs{PNumber: "   "})
	if err == nil {
		t.Error("Expected error for whitespace-only P-number")
	}
}

func TestGetByPNumberMCP_InvalidPNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetByPNumberMCP(ctx(), GetByPNumberArgs{PNumber: "12345"})
	if err == nil {
		t.Error("Expected error for invalid P-number")
	}
}

func TestGetByPNumberMCP_PNumberNormalization(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedNormal string
	}{
		{"with spaces", "12 3456 7890", "1234567890"},
		{"with dashes", "12-3456-7890", "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPNumber string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPNumber = r.URL.Query().Get("produ")
				company := Company{CVR: 12345678, Name: "Test"}
				_ = json.NewEncoder(w).Encode(company)
			}))
			defer server.Close()

			client := NewClient(WithBaseURL(server.URL))
			defer client.Close()

			_, _ = client.GetByPNumberMCP(ctx(), GetByPNumberArgs{PNumber: tt.input})
			if receivedPNumber != tt.expectedNormal {
				t.Errorf("produ param = %q, want %q", receivedPNumber, tt.expectedNormal)
			}
		})
	}
}

// =============================================================================
// Context and Error Handling Tests
// =============================================================================

func TestMCPMethods_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't send response - wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.SearchCompaniesMCP(cancelCtx, SearchCompaniesArgs{Query: "test"})
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

func TestMCPMethods_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompanyMCP(ctx(), GetCompanyArgs{CVR: "10150817"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

// Note: ctx() helper is defined in client_test.go

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestGetProductionUnitsMCP_PageBeyondTotal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:             10150817,
			ProductionUnits: []ProductionUnit{{PNumber: 1111111111}},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// Request page 10 when there's only 1 unit (page 0)
	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Page: 10})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if len(result.ProductionUnits) != 0 {
		t.Errorf("Expected 0 units for page beyond total, got %d", len(result.ProductionUnits))
	}
	if result.TotalResults != 1 {
		t.Errorf("TotalResults = %d, want %d", result.TotalResults, 1)
	}
}

func TestGetProductionUnitsMCP_ZeroSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:             10150817,
			ProductionUnits: []ProductionUnit{{PNumber: 1111111111}},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// Request size of 0 should use default (20)
	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Size: 0})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.Size != 20 {
		t.Errorf("Size = %d, want %d (default)", result.Size, 20)
	}
}

func TestGetProductionUnitsMCP_NegativeSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			CVR:             10150817,
			ProductionUnits: []ProductionUnit{{PNumber: 1111111111}},
		}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// Request negative size should use default (20)
	result, err := client.GetProductionUnitsMCP(ctx(), GetProductionUnitsArgs{CVR: "10150817", Size: -5})
	if err != nil {
		t.Fatalf("GetProductionUnitsMCP failed: %v", err)
	}

	if result.Size != 20 {
		t.Errorf("Size = %d, want %d (default)", result.Size, 20)
	}
}
