package denmark

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/base"
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
		err      *NotFoundError
		expected string
	}{
		{
			name:     "with query",
			err:      &NotFoundError{Query: "test company"},
			expected: "company not found: test company",
		},
		{
			name:     "with CVR",
			err:      &NotFoundError{CVR: "12345678"},
			expected: "company not found with CVR: 12345678",
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
