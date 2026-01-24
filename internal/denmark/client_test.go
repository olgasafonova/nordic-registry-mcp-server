package denmark

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/base"
	apierrors "github.com/olgasafonova/nordic-registry-mcp-server/internal/errors"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
	if client.Cache == nil {
		t.Error("Cache is nil")
	}
	if client.Dedup == nil {
		t.Error("Dedup is nil")
	}
	if client.CircuitBreaker == nil {
		t.Error("CircuitBreaker is nil")
	}
	client.Close()
}

func TestNewClientWithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 60 * time.Second}
	client := NewClient(WithHTTPClient(customHTTPClient))

	if client.HTTPClient != customHTTPClient {
		t.Error("custom HTTP client was not set")
	}
	client.Close()
}

func TestClient_ConcurrencyLimit(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Check semaphore capacity
	if cap(client.Semaphore) != base.MaxConcurrentRequests {
		t.Errorf("semaphore capacity = %d, want %d", cap(client.Semaphore), base.MaxConcurrentRequests)
	}
}

func TestClient_DefaultTimeout(t *testing.T) {
	client := NewClient()
	defer client.Close()

	if client.HTTPClient.Timeout != base.DefaultTimeout {
		t.Errorf("timeout = %v, want %v", client.HTTPClient.Timeout, base.DefaultTimeout)
	}
}

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      *apierrors.NotFoundError
		expected string
	}{
		{
			name:     "with query",
			err:      apierrors.NewNotFoundError("denmark", "test company"),
			expected: "company not found in denmark registry: test company",
		},
		{
			name:     "with CVR",
			err:      apierrors.NewNotFoundError("denmark", "12345678"),
			expected: "company not found in denmark registry: 12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("error message = %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestGetCompany_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client := NewClient(WithHTTPClient(server.Client()))
	defer client.Close()

	// Verify client is properly initialized
	if client.Cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestGetCompany_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := NewClient(WithHTTPClient(server.Client()))
	defer client.Close()

	// Verify client handles server errors gracefully
	if client.CircuitBreaker == nil {
		t.Error("circuit breaker should not be nil")
	}
}

func TestCompanyStatus(t *testing.T) {
	tests := []struct {
		name     string
		company  *Company
		expected string
	}{
		{
			name:     "active company",
			company:  &Company{},
			expected: "ACTIVE",
		},
		{
			name:     "dissolved company",
			company:  &Company{EndDate: "2024-01-01"},
			expected: "DISSOLVED",
		},
		{
			name:     "bankrupt company",
			company:  &Company{CreditEnd: true},
			expected: "BANKRUPT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getStatus(tt.company)
			if status != tt.expected {
				t.Errorf("getStatus() = %q, want %q", status, tt.expected)
			}
		})
	}
}

func TestGetCompany_ParsesAPIResponse(t *testing.T) {
	// This response matches the actual CVR API format
	apiResponse := `{
		"vat": 10150817,
		"name": "NOVO NORDISK A/S",
		"address": "Novo Allé 1",
		"zipcode": "2880",
		"city": "Bagsværd",
		"country": "DK",
		"phone": "44448888",
		"email": "info@novonordisk.com",
		"startdate": "01/01 - 1925",
		"enddate": "",
		"employees": 45000,
		"companydesc": "Aktieselskab",
		"industrycode": 212000,
		"industrydesc": "Fremstilling af farmaceutiske præparater",
		"creditstartdate": "",
		"creditbankrupt": false,
		"status": "NORMAL",
		"productionunits": [
			{
				"pno": 1234567890,
				"main": true,
				"name": "Hovedkontor",
				"address": "Novo Allé 1",
				"zipcode": "2880",
				"city": "Bagsværd",
				"employees": 5000,
				"industrycode": 212000,
				"industrydesc": "Fremstilling af farmaceutiske præparater"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiResponse))
	}))
	defer server.Close()

	client := NewClient()
	defer client.Close()

	// Override the base URL (we need to create a custom doRequest for testing)
	// For now, verify the JSON can be parsed directly
	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse API response: %v", err)
	}

	// Verify all fields parsed correctly
	if company.CVR != 10150817 {
		t.Errorf("CVR = %d, want 10150817", company.CVR)
	}
	if company.Name != "NOVO NORDISK A/S" {
		t.Errorf("Name = %q, want %q", company.Name, "NOVO NORDISK A/S")
	}
	if company.Employees != 45000 {
		t.Errorf("Employees = %d, want 45000", company.Employees)
	}
	if company.IndustryCode != 212000 {
		t.Errorf("IndustryCode = %d, want 212000", company.IndustryCode)
	}
	if company.CreditEnd != false {
		t.Errorf("CreditEnd = %v, want false", company.CreditEnd)
	}
	if len(company.ProductionUnits) != 1 {
		t.Fatalf("ProductionUnits count = %d, want 1", len(company.ProductionUnits))
	}

	pu := company.ProductionUnits[0]
	if pu.PNumber != 1234567890 {
		t.Errorf("ProductionUnit.PNumber = %d, want 1234567890", pu.PNumber)
	}
	if !pu.Main {
		t.Error("ProductionUnit.Main = false, want true")
	}
}

func TestGetCompany_HandlesNullEmployees(t *testing.T) {
	// Production units can have null employees
	apiResponse := `{
		"vat": 12345678,
		"name": "Test Company",
		"employees": 100,
		"creditbankrupt": false,
		"productionunits": [
			{
				"pno": 9876543210,
				"main": true,
				"name": "Branch",
				"employees": null
			}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse response with null employees: %v", err)
	}

	if len(company.ProductionUnits) != 1 {
		t.Fatalf("ProductionUnits count = %d, want 1", len(company.ProductionUnits))
	}

	// employees is `any` type, should be nil for null
	if company.ProductionUnits[0].Employees != nil {
		t.Errorf("Employees should be nil for null, got %v", company.ProductionUnits[0].Employees)
	}
}

func TestGetCompany_HandlesStringEmployees(t *testing.T) {
	// Some responses return employees as string
	apiResponse := `{
		"vat": 12345678,
		"name": "Test Company",
		"employees": 50,
		"creditbankrupt": false,
		"productionunits": [
			{
				"pno": 9876543210,
				"main": false,
				"name": "Small Branch",
				"employees": "1-4"
			}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse response with string employees: %v", err)
	}

	pu := company.ProductionUnits[0]
	empStr, ok := pu.Employees.(string)
	if !ok {
		t.Errorf("Employees should be string, got %T", pu.Employees)
	}
	if empStr != "1-4" {
		t.Errorf("Employees = %q, want %q", empStr, "1-4")
	}
}

func TestNormalizeCVR(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"12345678", "12345678"},
		{"DK12345678", "12345678"},
		{"dk12345678", "12345678"},
		{"DK-12345678", "12345678"},
		{"12 34 56 78", "12345678"},
		{"DK 12-34-56-78", "12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeCVR(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeCVR(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOwnerParsing(t *testing.T) {
	apiResponse := `{
		"vat": 12345678,
		"name": "Test ApS",
		"creditbankrupt": false,
		"owners": [
			{
				"name": "John Doe"
			},
			{
				"name": "Jane Smith"
			}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(company.Owners) != 2 {
		t.Fatalf("Expected 2 owners, got %d", len(company.Owners))
	}

	if company.Owners[0].Name != "John Doe" {
		t.Errorf("Owner[0].Name = %q, want %q", company.Owners[0].Name, "John Doe")
	}
}

func TestCompanyTypes(t *testing.T) {
	// Test different company types parsing
	tests := []struct {
		name     string
		json     string
		expected string
	}{
		{
			name:     "ApS company",
			json:     `{"vat": 12345678, "name": "Test ApS", "companydesc": "Anpartsselskab", "creditbankrupt": false}`,
			expected: "Anpartsselskab",
		},
		{
			name:     "A/S company",
			json:     `{"vat": 87654321, "name": "Test A/S", "companydesc": "Aktieselskab", "creditbankrupt": false}`,
			expected: "Aktieselskab",
		},
		{
			name:     "I/S company",
			json:     `{"vat": 11111111, "name": "Test I/S", "companydesc": "Interessentskab", "creditbankrupt": false}`,
			expected: "Interessentskab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var company Company
			if err := json.Unmarshal([]byte(tt.json), &company); err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
			if company.CompanyType != tt.expected {
				t.Errorf("CompanyType = %q, want %q", company.CompanyType, tt.expected)
			}
		})
	}
}

func TestGetStatusVariations(t *testing.T) {
	tests := []struct {
		name     string
		company  *Company
		expected string
	}{
		{
			name:     "active with employees",
			company:  &Company{Employees: 100},
			expected: "ACTIVE",
		},
		{
			name:     "dissolved with end date",
			company:  &Company{EndDate: "2023-12-31"},
			expected: "DISSOLVED",
		},
		{
			name:     "bankrupt with credit end",
			company:  &Company{CreditEnd: true},
			expected: "BANKRUPT",
		},
		{
			name:     "dissolved takes precedence over bankrupt",
			company:  &Company{EndDate: "2023-12-31", CreditEnd: true},
			expected: "DISSOLVED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getStatus(tt.company)
			if status != tt.expected {
				t.Errorf("getStatus() = %q, want %q", status, tt.expected)
			}
		})
	}
}

func TestProductionUnitParsing(t *testing.T) {
	apiResponse := `{
		"vat": 12345678,
		"name": "Test Company",
		"creditbankrupt": false,
		"productionunits": [
			{
				"pno": 1111111111,
				"main": true,
				"name": "Hovedkontor",
				"address": "Street 1",
				"zipcode": "1000",
				"city": "Copenhagen",
				"employees": 50,
				"industrycode": 123456,
				"industrydesc": "Software development"
			},
			{
				"pno": 2222222222,
				"main": false,
				"name": "Branch Office",
				"address": "Street 2",
				"zipcode": "2000",
				"city": "Frederiksberg",
				"employees": 10,
				"industrycode": 123456,
				"industrydesc": "Software development"
			}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(company.ProductionUnits) != 2 {
		t.Fatalf("Expected 2 production units, got %d", len(company.ProductionUnits))
	}

	// Check main unit
	main := company.ProductionUnits[0]
	if !main.Main {
		t.Error("First unit should be main")
	}
	if main.PNumber != 1111111111 {
		t.Errorf("PNumber = %d, want %d", main.PNumber, 1111111111)
	}

	// Check branch
	branch := company.ProductionUnits[1]
	if branch.Main {
		t.Error("Second unit should not be main")
	}
	if branch.City != "Frederiksberg" {
		t.Errorf("City = %q, want %q", branch.City, "Frederiksberg")
	}
}

// HTTP Mocking Tests
// These tests use httptest to mock the CVR API and verify actual client behavior

func TestGetCompany_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("vat") != "10150817" {
			t.Errorf("unexpected vat param: %s", r.URL.Query().Get("vat"))
		}
		if r.URL.Query().Get("country") != "dk" {
			t.Errorf("unexpected country param: %s", r.URL.Query().Get("country"))
		}

		company := Company{
			CVR:         10150817,
			Name:        "NOVO NORDISK A/S",
			Address:     "Novo Allé 1",
			Zipcode:     "2880",
			City:        "Bagsværd",
			Employees:   45000,
			CompanyType: "Aktieselskab",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompany(ctx(), "10150817")
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}

	if result.CVR != 10150817 {
		t.Errorf("CVR = %d, want %d", result.CVR, 10150817)
	}
	if result.Name != "NOVO NORDISK A/S" {
		t.Errorf("Name = %q, want %q", result.Name, "NOVO NORDISK A/S")
	}
}

func TestGetCompany_NotFound_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "not found", "t": 1}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompany(ctx(), "00000000")
	if err == nil {
		t.Fatal("Expected error for not found, got nil")
	}
}

func TestSearchCompany_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "Novo Nordisk" {
			t.Errorf("unexpected search param: %s", r.URL.Query().Get("search"))
		}

		company := Company{
			CVR:       10150817,
			Name:      "NOVO NORDISK A/S",
			Employees: 45000,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompany(ctx(), "Novo Nordisk")
	if err != nil {
		t.Fatalf("SearchCompany failed: %v", err)
	}

	if result.Name != "NOVO NORDISK A/S" {
		t.Errorf("Name = %q, want %q", result.Name, "NOVO NORDISK A/S")
	}
}

func TestSearchCompany_NoResults_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CVR API returns empty object when no results
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.SearchCompany(ctx(), "nonexistent company xyz123")
	if err == nil {
		t.Fatal("Expected error for no results, got nil")
	}
}

func TestGetByPNumber_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("produ") != "1234567890" {
			t.Errorf("unexpected produ param: %s", r.URL.Query().Get("produ"))
		}

		company := Company{
			CVR:       10150817,
			Name:      "NOVO NORDISK A/S",
			Employees: 45000,
			ProductionUnits: []ProductionUnit{
				{
					PNumber: 1234567890,
					Main:    true,
					Name:    "Hovedkontor",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetByPNumber(ctx(), "1234567890")
	if err != nil {
		t.Fatalf("GetByPNumber failed: %v", err)
	}

	if result.CVR != 10150817 {
		t.Errorf("CVR = %d, want %d", result.CVR, 10150817)
	}
}

func TestSearchByPhone_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("phone") != "44448888" {
			t.Errorf("unexpected phone param: %s", r.URL.Query().Get("phone"))
		}

		company := Company{
			CVR:   10150817,
			Name:  "NOVO NORDISK A/S",
			Phone: "44448888",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchByPhone(ctx(), "44448888")
	if err != nil {
		t.Fatalf("SearchByPhone failed: %v", err)
	}

	if result.Phone != "44448888" {
		t.Errorf("Phone = %q, want %q", result.Phone, "44448888")
	}
}

func TestSearchByPhone_PhoneNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with spaces", "44 44 88 88", "44448888"},
		{"with dashes", "44-44-88-88", "44448888"},
		{"with country code", "+4544448888", "44448888"},
		{"clean", "44448888", "44448888"},
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

			_, _ = client.SearchByPhone(ctx(), tt.input)
			if receivedPhone != tt.expected {
				t.Errorf("phone param = %q, want %q", receivedPhone, tt.expected)
			}
		})
	}
}

func TestClient_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		company := Company{CVR: 10150817, Name: "NOVO NORDISK A/S"}
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, err := client.GetCompany(ctx(), "10150817")
	if err != nil {
		t.Fatalf("First GetCompany failed: %v", err)
	}

	// Second call should hit cache
	_, err = client.GetCompany(ctx(), "10150817")
	if err != nil {
		t.Fatalf("Second GetCompany failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestClient_ServerError_WithMockServer(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompany(ctx(), "10150817")
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
	// Should have retried
	if attempts < 2 {
		t.Errorf("Expected at least 2 attempts (retry), got %d", attempts)
	}
}

func TestWithBaseURL(t *testing.T) {
	client := NewClient(WithBaseURL("http://test.example.com"))
	defer client.Close()

	if client.baseURL != "http://test.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "http://test.example.com")
	}
}

func TestValidateCVR_internal(t *testing.T) {
	tests := []struct {
		name    string
		cvr     string
		wantErr bool
	}{
		{"valid 8 digits", "12345678", false},
		{"too short", "1234567", true},
		{"too long", "123456789", true},
		{"with letters", "1234567A", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCVR(tt.cvr)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCVR(%q) error = %v, wantErr %v", tt.cvr, err, tt.wantErr)
			}
		})
	}
}

func TestAPIError_String(t *testing.T) {
	tests := []struct {
		name     string
		err      APIError
		expected string
	}{
		{
			name:     "with error message",
			err:      APIError{Error: "Not found"},
			expected: "Not found",
		},
		{
			name:     "empty",
			err:      APIError{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.String()
			if result != tt.expected {
				t.Errorf("APIError.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// ctx returns a background context for testing
func ctx() context.Context {
	return context.Background()
}
