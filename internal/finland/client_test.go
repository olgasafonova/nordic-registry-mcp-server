package finland

import (
	"encoding/json"
	"net/http"
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

func TestNormalizeBusinessID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid format", "0112038-9", "0112038-9", false},
		{"with spaces", " 0112038-9 ", "0112038-9", false},
		{"with FI prefix", "FI0112038-9", "0112038-9", false},
		{"with fi prefix lowercase", "fi0112038-9", "0112038-9", false},
		{"invalid format no hyphen", "01120389", "", true},
		{"invalid format too short", "112038-9", "", true},
		{"invalid format letters", "011203A-9", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBusinessID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeBusinessID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeBusinessID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClient_Initialization(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Verify all client components are properly initialized
	if client.Cache == nil {
		t.Error("cache should not be nil")
	}
	if client.CircuitBreaker == nil {
		t.Error("circuit breaker should not be nil")
	}
	if client.Dedup == nil {
		t.Error("dedup should not be nil")
	}
}

func TestStatusToDesc(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1", "Registered"},
		{"2", "Active"},
		{"3", "Dissolved"},
		{"4", "Liquidation"},
		{"5", "Bankruptcy"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := statusToDesc(tt.input)
			if result != tt.expected {
				t.Errorf("statusToDesc(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSituationTypeToDesc(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SANE", "Reorganization"},
		{"SELTILA", "Liquidation"},
		{"KONK", "Bankruptcy"},
		{"OTHER", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := situationTypeToDesc(tt.input)
			if result != tt.expected {
				t.Errorf("situationTypeToDesc(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetEnglishDesc(t *testing.T) {
	tests := []struct {
		name     string
		descs    []Description
		expected string
	}{
		{
			name:     "english available",
			descs:    []Description{{LanguageCode: "3", Description: "Limited company"}},
			expected: "Limited company",
		},
		{
			name:     "fallback to finnish",
			descs:    []Description{{LanguageCode: "1", Description: "Osakeyhtiö"}},
			expected: "Osakeyhtiö",
		},
		{
			name: "prefer english over finnish",
			descs: []Description{
				{LanguageCode: "1", Description: "Osakeyhtiö"},
				{LanguageCode: "3", Description: "Limited company"},
			},
			expected: "Limited company",
		},
		{
			name:     "empty list",
			descs:    []Description{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnglishDesc(tt.descs)
			if result != tt.expected {
				t.Errorf("getEnglishDesc() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     Address
		expected string
	}{
		{
			name:     "street only",
			addr:     Address{Street: "Main Street"},
			expected: "Main Street",
		},
		{
			name:     "street with building number",
			addr:     Address{Street: "Main Street", BuildingNumber: "10"},
			expected: "Main Street 10",
		},
		{
			name:     "full address",
			addr:     Address{Street: "Main Street", BuildingNumber: "10", Entrance: "A", ApartmentNumber: "5"},
			expected: "Main Street 10 A, 5",
		},
		{
			name:     "free form address",
			addr:     Address{FreeAddressLine: "PO Box 123"},
			expected: "PO Box 123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAddress(tt.addr)
			if result != tt.expected {
				t.Errorf("formatAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseCompanySearchResponse(t *testing.T) {
	apiResponse := `{
		"totalResults": 1,
		"companies": [
			{
				"businessId": {
					"value": "0112038-9",
					"registrationDate": "1978-01-01"
				},
				"euId": {
					"value": "FIOY0112038-9"
				},
				"names": [
					{"name": "Nokia Oyj", "type": "1", "registrationDate": "2017-04-28"},
					{"name": "Nokia Corporation", "type": "3"}
				],
				"mainBusinessLine": {
					"type": "26110",
					"descriptions": [
						{"languageCode": "1", "description": "Elektronisten komponenttien valmistus"},
						{"languageCode": "3", "description": "Manufacture of electronic components"}
					]
				},
				"website": {
					"url": "www.nokia.com"
				},
				"companyForms": [
					{
						"type": "16",
						"descriptions": [
							{"languageCode": "1", "description": "Julkinen osakeyhtiö"},
							{"languageCode": "3", "description": "Public limited company"}
						]
					}
				],
				"status": "2"
			}
		]
	}`

	var resp CompanySearchResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.TotalResults != 1 {
		t.Errorf("TotalResults = %d, want %d", resp.TotalResults, 1)
	}
	if len(resp.Companies) != 1 {
		t.Fatalf("Expected 1 company, got %d", len(resp.Companies))
	}

	company := resp.Companies[0]
	if company.BusinessID.Value != "0112038-9" {
		t.Errorf("BusinessID = %q, want %q", company.BusinessID.Value, "0112038-9")
	}
	if company.EUID == nil || company.EUID.Value != "FIOY0112038-9" {
		t.Error("EUID should be FIOY0112038-9")
	}
	if len(company.Names) != 2 {
		t.Fatalf("Expected 2 names, got %d", len(company.Names))
	}
	if company.Names[0].Name != "Nokia Oyj" {
		t.Errorf("Name = %q, want %q", company.Names[0].Name, "Nokia Oyj")
	}
	if company.MainBusinessLine == nil {
		t.Fatal("MainBusinessLine should not be nil")
	}
	if company.MainBusinessLine.Type != "26110" {
		t.Errorf("MainBusinessLine.Type = %q, want %q", company.MainBusinessLine.Type, "26110")
	}
	if company.Website == nil || company.Website.URL != "www.nokia.com" {
		t.Error("Website.URL should be www.nokia.com")
	}
	if len(company.CompanyForms) != 1 {
		t.Fatalf("Expected 1 company form, got %d", len(company.CompanyForms))
	}
	if company.CompanyForms[0].Type != "16" {
		t.Errorf("CompanyForm.Type = %q, want %q", company.CompanyForms[0].Type, "16")
	}
}

func TestParseCompanyWithSituations(t *testing.T) {
	apiResponse := `{
		"businessId": {"value": "1234567-8"},
		"companySituations": [
			{"type": "SANE", "registrationDate": "2023-01-15"},
			{"type": "KONK", "registrationDate": "2024-01-01"}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(company.CompanySituations) != 2 {
		t.Fatalf("Expected 2 situations, got %d", len(company.CompanySituations))
	}
	if company.CompanySituations[0].Type != "SANE" {
		t.Errorf("Situation[0].Type = %q, want %q", company.CompanySituations[0].Type, "SANE")
	}
	if company.CompanySituations[1].Type != "KONK" {
		t.Errorf("Situation[1].Type = %q, want %q", company.CompanySituations[1].Type, "KONK")
	}
}

func TestParseCompanyWithAddresses(t *testing.T) {
	apiResponse := `{
		"businessId": {"value": "1234567-8"},
		"addresses": [
			{
				"type": 1,
				"street": "Karakaari 7",
				"postCode": "02610",
				"buildingNumber": "7",
				"postOffices": [
					{"city": "Espoo", "languageCode": "1"},
					{"city": "Esbo", "languageCode": "2"}
				],
				"country": "FI"
			},
			{
				"type": 2,
				"postOfficeBox": "PL 226",
				"postCode": "00045",
				"postOffices": [
					{"city": "NOKIA GROUP", "languageCode": "1"}
				]
			}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(company.Addresses) != 2 {
		t.Fatalf("Expected 2 addresses, got %d", len(company.Addresses))
	}

	streetAddr := company.Addresses[0]
	if streetAddr.Type != 1 {
		t.Errorf("Address[0].Type = %d, want 1", streetAddr.Type)
	}
	if streetAddr.Street != "Karakaari 7" {
		t.Errorf("Street = %q, want %q", streetAddr.Street, "Karakaari 7")
	}
	if len(streetAddr.PostOffices) != 2 {
		t.Fatalf("Expected 2 post offices, got %d", len(streetAddr.PostOffices))
	}
	if streetAddr.PostOffices[0].City != "Espoo" {
		t.Errorf("PostOffice.City = %q, want %q", streetAddr.PostOffices[0].City, "Espoo")
	}

	postalAddr := company.Addresses[1]
	if postalAddr.Type != 2 {
		t.Errorf("Address[1].Type = %d, want 2", postalAddr.Type)
	}
	if postalAddr.PostOfficeBox != "PL 226" {
		t.Errorf("PostOfficeBox = %q, want %q", postalAddr.PostOfficeBox, "PL 226")
	}
}

func TestParseCompanyWithRegisteredEntries(t *testing.T) {
	apiResponse := `{
		"businessId": {"value": "1234567-8"},
		"registeredEntries": [
			{
				"registrationStatus": "Registered",
				"type": "1",
				"register": "1",
				"registerDescriptions": [
					{"languageCode": "1", "description": "Kaupparekisteri"},
					{"languageCode": "3", "description": "Trade Register"}
				],
				"registrationDate": "1990-01-01"
			}
		]
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(company.RegisteredEntries) != 1 {
		t.Fatalf("Expected 1 registered entry, got %d", len(company.RegisteredEntries))
	}

	entry := company.RegisteredEntries[0]
	if entry.RegistrationStatus != "Registered" {
		t.Errorf("RegistrationStatus = %q, want %q", entry.RegistrationStatus, "Registered")
	}
	if len(entry.RegisterDescriptions) != 2 {
		t.Fatalf("Expected 2 descriptions, got %d", len(entry.RegisterDescriptions))
	}
}

func TestGetEnglishDescFallback(t *testing.T) {
	// Test fallback order: en > fi (Swedish not supported, returns empty)
	tests := []struct {
		name     string
		descs    []Description
		expected string
	}{
		{
			name:     "only swedish returns empty",
			descs:    []Description{{LanguageCode: "2", Description: "Svenska"}},
			expected: "", // Function only handles English (3) and Finnish (1)
		},
		{
			name: "swedish and finnish, returns finnish",
			descs: []Description{
				{LanguageCode: "2", Description: "Svenska"},
				{LanguageCode: "1", Description: "Suomi"},
			},
			expected: "Suomi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnglishDesc(tt.descs)
			if result != tt.expected {
				t.Errorf("getEnglishDesc() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetCity(t *testing.T) {
	tests := []struct {
		name     string
		addr     Address
		expected string
	}{
		{
			name: "finnish city",
			addr: Address{PostOffices: []PostOffice{
				{City: "Espoo", LanguageCode: "1"},
				{City: "Esbo", LanguageCode: "2"},
			}},
			expected: "Espoo",
		},
		{
			name: "only swedish city",
			addr: Address{PostOffices: []PostOffice{
				{City: "Esbo", LanguageCode: "2"},
			}},
			expected: "Esbo",
		},
		{
			name:     "no post offices",
			addr:     Address{},
			expected: "",
		},
		{
			name: "prefer finnish over swedish",
			addr: Address{PostOffices: []PostOffice{
				{City: "Esbo", LanguageCode: "2"},
				{City: "Espoo", LanguageCode: "1"},
			}},
			expected: "Espoo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCity(tt.addr)
			if result != tt.expected {
				t.Errorf("getCity() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestToCompanySummary(t *testing.T) {
	company := Company{
		BusinessID: BusinessID{Value: "0112038-9", RegistrationDate: "1978-01-01"},
		Names: []CompanyName{
			{Name: "Nokia Oyj", Type: "1", EndDate: ""},
			{Name: "Old Name", Type: "1", EndDate: "2010-01-01"},
		},
		CompanyForms: []CompanyForm{
			{Type: "16", Descriptions: []Description{{LanguageCode: "3", Description: "Public limited company"}}, EndDate: ""},
		},
		MainBusinessLine: &BusinessLine{
			Type:         "26110",
			Descriptions: []Description{{LanguageCode: "3", Description: "Manufacture of electronic components"}},
		},
		Website: &Website{URL: "www.nokia.com"},
		Addresses: []Address{
			{
				Type:           1,
				Street:         "Karakaari",
				BuildingNumber: "7",
				PostCode:       "02610",
				PostOffices:    []PostOffice{{City: "Espoo", LanguageCode: "1"}},
			},
		},
		Status: "2",
	}

	summary := toCompanySummary(company)

	if summary.BusinessID != "0112038-9" {
		t.Errorf("BusinessID = %q, want %q", summary.BusinessID, "0112038-9")
	}
	if summary.Name != "Nokia Oyj" {
		t.Errorf("Name = %q, want %q", summary.Name, "Nokia Oyj")
	}
	if summary.CompanyForm != "16" {
		t.Errorf("CompanyForm = %q, want %q", summary.CompanyForm, "16")
	}
	if summary.CompanyFormDesc != "Public limited company" {
		t.Errorf("CompanyFormDesc = %q, want %q", summary.CompanyFormDesc, "Public limited company")
	}
	if summary.IndustryCode != "26110" {
		t.Errorf("IndustryCode = %q, want %q", summary.IndustryCode, "26110")
	}
	if summary.Website != "www.nokia.com" {
		t.Errorf("Website = %q, want %q", summary.Website, "www.nokia.com")
	}
	if summary.City != "Espoo" {
		t.Errorf("City = %q, want %q", summary.City, "Espoo")
	}
	if summary.Status != "Active" {
		t.Errorf("Status = %q, want %q", summary.Status, "Active")
	}
}

func TestToCompanySummaryFallbacks(t *testing.T) {
	// Test fallback to first name when no current name
	company := Company{
		BusinessID: BusinessID{Value: "1234567-8"},
		Names: []CompanyName{
			{Name: "Historical Name", Type: "1", EndDate: "2010-01-01"},
		},
		Status: "3",
	}

	summary := toCompanySummary(company)

	if summary.Name != "Historical Name" {
		t.Errorf("Name = %q, want %q (fallback to first name)", summary.Name, "Historical Name")
	}
	if summary.Status != "Dissolved" {
		t.Errorf("Status = %q, want %q", summary.Status, "Dissolved")
	}
}

func TestToCompanyDetailSummary(t *testing.T) {
	company := &Company{
		BusinessID:       BusinessID{Value: "0112038-9", RegistrationDate: "1978-01-01"},
		RegistrationDate: "1978-01-01",
		Names: []CompanyName{
			{Name: "Nokia Oyj", Type: "1", EndDate: ""},
		},
		CompanyForms: []CompanyForm{
			{Type: "16", Descriptions: []Description{{LanguageCode: "3", Description: "Public limited company"}}, EndDate: ""},
		},
		MainBusinessLine: &BusinessLine{
			Type:         "26110",
			Descriptions: []Description{{LanguageCode: "3", Description: "Manufacture of electronic components"}},
		},
		Website: &Website{URL: "www.nokia.com"},
		Addresses: []Address{
			{
				Type:           1,
				Street:         "Karakaari",
				BuildingNumber: "7",
				PostCode:       "02610",
				PostOffices:    []PostOffice{{City: "Espoo", LanguageCode: "1"}},
			},
		},
		Status: "2",
	}

	summary := toCompanyDetailSummary(company)

	if summary.BusinessID != "0112038-9" {
		t.Errorf("BusinessID = %q, want %q", summary.BusinessID, "0112038-9")
	}
	if summary.Name != "Nokia Oyj" {
		t.Errorf("Name = %q, want %q", summary.Name, "Nokia Oyj")
	}
	if summary.CompanyForm != "16 - Public limited company" {
		t.Errorf("CompanyForm = %q, want %q", summary.CompanyForm, "16 - Public limited company")
	}
	if summary.Industry != "26110 - Manufacture of electronic components" {
		t.Errorf("Industry = %q, want %q", summary.Industry, "26110 - Manufacture of electronic components")
	}
	if summary.Website != "www.nokia.com" {
		t.Errorf("Website = %q, want %q", summary.Website, "www.nokia.com")
	}
	if summary.StreetAddress != "Karakaari 7" {
		t.Errorf("StreetAddress = %q, want %q", summary.StreetAddress, "Karakaari 7")
	}
	if summary.City != "Espoo" {
		t.Errorf("City = %q, want %q", summary.City, "Espoo")
	}
	if summary.Status != "Active" {
		t.Errorf("Status = %q, want %q", summary.Status, "Active")
	}
}

func TestToCompanyDetailSummaryNil(t *testing.T) {
	summary := toCompanyDetailSummary(nil)
	if summary != nil {
		t.Error("toCompanyDetailSummary(nil) should return nil")
	}
}

func TestToCompanyDetails(t *testing.T) {
	company := &Company{
		BusinessID:       BusinessID{Value: "0112038-9", RegistrationDate: "1978-01-01"},
		EUID:             &EUID{Value: "FIOY0112038-9"},
		RegistrationDate: "1978-01-01",
		LastModified:     "2024-01-15T10:00:00Z",
		Names: []CompanyName{
			{Name: "Nokia Oyj", Type: "1", EndDate: ""},
			{Name: "Old Nokia", Type: "1", EndDate: "2010-01-01"},
			{Name: "Nokia Mobile", Type: "3", EndDate: ""},
		},
		CompanyForms: []CompanyForm{
			{Type: "16", Descriptions: []Description{{LanguageCode: "3", Description: "Public limited company"}}, EndDate: ""},
		},
		MainBusinessLine: &BusinessLine{
			Type:         "26110",
			Descriptions: []Description{{LanguageCode: "3", Description: "Manufacture of electronic components"}},
		},
		Website: &Website{URL: "www.nokia.com"},
		Addresses: []Address{
			{
				Type:           1,
				Street:         "Karakaari",
				BuildingNumber: "7",
				PostCode:       "02610",
				PostOffices:    []PostOffice{{City: "Espoo", LanguageCode: "1"}},
			},
			{
				Type:        2,
				PostCode:    "00045",
				PostOffices: []PostOffice{{City: "NOKIA GROUP", LanguageCode: "1"}},
			},
		},
		CompanySituations: []CompanySituation{
			{Type: "SANE", RegistrationDate: "2023-01-15", EndDate: "2023-06-15"},
		},
		RegisteredEntries: []RegisteredEntry{
			{
				RegistrationStatus:   "Registered",
				RegisterDescriptions: []Description{{LanguageCode: "3", Description: "Trade Register"}},
				RegistrationDate:     "1990-01-01",
			},
		},
		Status: "2",
	}

	details := toCompanyDetails(company)

	if details.BusinessID != "0112038-9" {
		t.Errorf("BusinessID = %q, want %q", details.BusinessID, "0112038-9")
	}
	if details.EUID != "FIOY0112038-9" {
		t.Errorf("EUID = %q, want %q", details.EUID, "FIOY0112038-9")
	}
	if details.Name != "Nokia Oyj" {
		t.Errorf("Name = %q, want %q", details.Name, "Nokia Oyj")
	}
	if len(details.PreviousNames) != 1 || details.PreviousNames[0] != "Old Nokia" {
		t.Errorf("PreviousNames = %v, want [Old Nokia]", details.PreviousNames)
	}
	if len(details.AuxiliaryNames) != 1 || details.AuxiliaryNames[0] != "Nokia Mobile" {
		t.Errorf("AuxiliaryNames = %v, want [Nokia Mobile]", details.AuxiliaryNames)
	}
	if details.CompanyForm != "16" {
		t.Errorf("CompanyForm = %q, want %q", details.CompanyForm, "16")
	}
	if details.CompanyFormDesc != "Public limited company" {
		t.Errorf("CompanyFormDesc = %q, want %q", details.CompanyFormDesc, "Public limited company")
	}
	if details.IndustryCode != "26110" {
		t.Errorf("IndustryCode = %q, want %q", details.IndustryCode, "26110")
	}
	if details.Website != "www.nokia.com" {
		t.Errorf("Website = %q, want %q", details.Website, "www.nokia.com")
	}
	if details.StreetAddress == nil || details.StreetAddress.City != "Espoo" {
		t.Error("StreetAddress not properly set")
	}
	if details.PostalAddress == nil || details.PostalAddress.City != "NOKIA GROUP" {
		t.Error("PostalAddress not properly set")
	}
	if len(details.Situations) != 1 || details.Situations[0].Type != "Reorganization" {
		t.Errorf("Situations = %v, want [Reorganization]", details.Situations)
	}
	if len(details.Registrations) != 1 || details.Registrations[0].Register != "Trade Register" {
		t.Errorf("Registrations = %v, want [Trade Register]", details.Registrations)
	}
	if details.StatusDesc != "Active" {
		t.Errorf("StatusDesc = %q, want %q", details.StatusDesc, "Active")
	}
}

func TestToCompanyDetailsNil(t *testing.T) {
	details := toCompanyDetails(nil)
	if details != nil {
		t.Error("toCompanyDetails(nil) should return nil")
	}
}
