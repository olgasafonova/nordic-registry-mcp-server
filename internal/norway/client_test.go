package norway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/olgasafonova/nordic-registry-mcp-server/internal/base"
	"github.com/olgasafonova/nordic-registry-mcp-server/internal/infra"
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

func TestSearchCompanies(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/enhetsregisteret/api/enheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("navn") != "Equinor" {
			t.Errorf("unexpected query param navn: %s", r.URL.Query().Get("navn"))
		}

		// Return mock response
		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{
				Companies: []Company{
					{
						OrganizationNumber: "923609016",
						Name:               "EQUINOR ASA",
					},
				},
			},
			Page: PageInfo{
				TotalElements: 1,
				TotalPages:    1,
				Number:        0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with custom HTTP client pointing to mock server
	client := NewClient(WithHTTPClient(server.Client()))
	// Override the base URL by replacing the URL in requests
	// We need to patch the client to use our test server

	// For now, test that the client can be created and closed
	client.Close()
}

func TestGetCompany_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"feilmelding": "Ingen enhet med organisasjonsnummer 000000000 ble funnet"}`))
	}))
	defer server.Close()

	client := NewClient(WithHTTPClient(server.Client()))
	defer client.Close()

	// The client uses a hardcoded BaseURL, so this test validates the client construction
	// A more complete test would require dependency injection for the base URL
}

func TestGetCompany_ServerError(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
	}))
	defer server.Close()

	client := NewClient(WithHTTPClient(server.Client()))
	defer client.Close()

	// Verify client components are initialized for error handling
	if client.CircuitBreaker == nil {
		t.Error("circuit breaker should not be nil")
	}
	if client.Cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestClient_ConcurrencyLimit(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Check semaphore capacity
	if cap(client.Semaphore) != base.MaxConcurrentRequests {
		t.Errorf("semaphore capacity = %d, want %d", cap(client.Semaphore), base.MaxConcurrentRequests)
	}
}

func TestSearchOptions(t *testing.T) {
	tests := []struct {
		name string
		opts *SearchOptions
	}{
		{"nil options", nil},
		{"empty options", &SearchOptions{}},
		{"with page", &SearchOptions{Page: 1}},
		{"with size", &SearchOptions{Size: 50}},
		{"with org form", &SearchOptions{OrgForm: "AS"}},
		{"with all options", &SearchOptions{
			Page:    2,
			Size:    25,
			OrgForm: "ENK",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify options can be created
			_ = tt.opts
		})
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(WithHTTPClient(server.Client()))
	defer client.Close()

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The SearchCompanies call should respect context cancellation
	// (Note: this is a partial test since we can't easily override BaseURL)
	_ = ctx
}

// API Response Parsing Tests
// These tests verify our types match the actual API response shapes

func TestParseRolesResponse_WithElectedBy(t *testing.T) {
	// This response matches the actual Brønnøysund API format for roles
	// The valgtAv field is an object, not a string
	apiResponse := `{
		"rollegrupper": [
			{
				"type": {
					"kode": "STYR",
					"beskrivelse": "Styre"
				},
				"sistEndret": "2024-01-15",
				"roller": [
					{
						"type": {
							"kode": "STYR",
							"beskrivelse": "Styremedlem"
						},
						"person": {
							"navn": {
								"fornavn": "Ola",
								"etternavn": "Nordmann"
							},
							"fodselsdato": "1970-01-01",
							"erDod": false
						},
						"fratraadt": false,
						"valgtAv": {
							"kode": "AREP",
							"beskrivelse": "Representant for de ansatte",
							"_links": {
								"self": {
									"href": "https://data.brreg.no/enhetsregisteret/api/valgtAv/AREP"
								}
							}
						}
					}
				]
			}
		]
	}`

	var rolesResp RolesResponse
	if err := json.Unmarshal([]byte(apiResponse), &rolesResp); err != nil {
		t.Fatalf("Failed to parse roles response: %v", err)
	}

	if len(rolesResp.RoleGroups) != 1 {
		t.Fatalf("Expected 1 role group, got %d", len(rolesResp.RoleGroups))
	}

	roleGroup := rolesResp.RoleGroups[0]
	if roleGroup.Type.Code != "STYR" {
		t.Errorf("RoleGroup.Type.Code = %q, want %q", roleGroup.Type.Code, "STYR")
	}

	if len(roleGroup.Roles) != 1 {
		t.Fatalf("Expected 1 role, got %d", len(roleGroup.Roles))
	}

	role := roleGroup.Roles[0]
	if role.ElectedBy == nil {
		t.Fatal("ElectedBy should not be nil")
	}
	if role.ElectedBy.Code != "AREP" {
		t.Errorf("ElectedBy.Code = %q, want %q", role.ElectedBy.Code, "AREP")
	}
	if role.ElectedBy.Description != "Representant for de ansatte" {
		t.Errorf("ElectedBy.Description = %q, want %q", role.ElectedBy.Description, "Representant for de ansatte")
	}
}

func TestParseRolesResponse_WithoutElectedBy(t *testing.T) {
	// Some roles don't have valgtAv
	apiResponse := `{
		"rollegrupper": [
			{
				"type": {
					"kode": "DAGL",
					"beskrivelse": "Daglig leder"
				},
				"roller": [
					{
						"type": {
							"kode": "DAGL",
							"beskrivelse": "Daglig leder"
						},
						"person": {
							"navn": {
								"fornavn": "Kari",
								"etternavn": "Hansen"
							},
							"erDod": false
						},
						"fratraadt": false
					}
				]
			}
		]
	}`

	var rolesResp RolesResponse
	if err := json.Unmarshal([]byte(apiResponse), &rolesResp); err != nil {
		t.Fatalf("Failed to parse roles response: %v", err)
	}

	role := rolesResp.RoleGroups[0].Roles[0]
	if role.ElectedBy != nil {
		t.Error("ElectedBy should be nil when not present")
	}
}

func TestParseCompanyResponse(t *testing.T) {
	// Full company response from Brønnøysund API
	apiResponse := `{
		"organisasjonsnummer": "923609016",
		"navn": "EQUINOR ASA",
		"organisasjonsform": {
			"kode": "ASA",
			"beskrivelse": "Allmennaksjeselskap"
		},
		"registreringsdatoEnhetsregisteret": "1972-06-16",
		"registrertIMvaregisteret": true,
		"registrertIForetaksregisteret": true,
		"konkurs": false,
		"underAvvikling": false,
		"antallAnsatte": 21000,
		"forretningsadresse": {
			"land": "Norge",
			"landkode": "NO",
			"postnummer": "4035",
			"poststed": "STAVANGER",
			"kommunenummer": "1103",
			"kommune": "STAVANGER",
			"adresse": ["Forusbeen 50"]
		},
		"naeringskode1": {
			"kode": "06.100",
			"beskrivelse": "Utvinning av råolje"
		}
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse company response: %v", err)
	}

	if company.OrganizationNumber != "923609016" {
		t.Errorf("OrganizationNumber = %q, want %q", company.OrganizationNumber, "923609016")
	}
	if company.Name != "EQUINOR ASA" {
		t.Errorf("Name = %q, want %q", company.Name, "EQUINOR ASA")
	}
	if company.EmployeeCount != 21000 {
		t.Errorf("EmployeeCount = %d, want 21000", company.EmployeeCount)
	}
	if company.OrganizationForm == nil || company.OrganizationForm.Code != "ASA" {
		t.Error("OrganizationForm not parsed correctly")
	}
	if company.BusinessAddress == nil || company.BusinessAddress.PostalCode != "4035" {
		t.Error("BusinessAddress not parsed correctly")
	}
}

func TestParseSubUnitResponse(t *testing.T) {
	apiResponse := `{
		"organisasjonsnummer": "876543219",
		"navn": "EQUINOR ASA AVD OSLO",
		"overordnetEnhet": "923609016",
		"antallAnsatte": 500,
		"forretningsadresse": {
			"land": "Norge",
			"landkode": "NO",
			"postnummer": "0283",
			"poststed": "OSLO",
			"adresse": ["Martin Linges vei 33"]
		},
		"naeringskode1": {
			"kode": "06.100",
			"beskrivelse": "Utvinning av råolje"
		}
	}`

	var subunit SubUnit
	if err := json.Unmarshal([]byte(apiResponse), &subunit); err != nil {
		t.Fatalf("Failed to parse subunit response: %v", err)
	}

	if subunit.OrganizationNumber != "876543219" {
		t.Errorf("OrganizationNumber = %q, want %q", subunit.OrganizationNumber, "876543219")
	}
	if subunit.ParentOrganizationNumber != "923609016" {
		t.Errorf("ParentOrganizationNumber = %q, want %q", subunit.ParentOrganizationNumber, "923609016")
	}
	if subunit.EmployeeCount != 500 {
		t.Errorf("EmployeeCount = %d, want 500", subunit.EmployeeCount)
	}
}

func TestNormalizeOrgNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"923609016", "923609016"},
		{"923 609 016", "923609016"},
		{"923-609-016", "923609016"},
		{"923 609-016", "923609016"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeOrgNumber(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeOrgNumber(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateOrgNumberInternal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 9 digits", "923609016", false},
		{"valid with spaces normalized", "923609016", false},
		{"too short", "12345678", true},
		{"too long", "1234567890", true},
		{"letters in number", "92360901A", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrgNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOrgNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestPersonName_FullName(t *testing.T) {
	tests := []struct {
		name     string
		person   PersonName
		expected string
	}{
		{
			name:     "first and last",
			person:   PersonName{FirstName: "Ola", LastName: "Nordmann"},
			expected: "Ola Nordmann",
		},
		{
			name:     "with middle name",
			person:   PersonName{FirstName: "Ola", MiddleName: "Johan", LastName: "Nordmann"},
			expected: "Ola Johan Nordmann",
		},
		{
			name:     "last name only",
			person:   PersonName{LastName: "Nordmann"},
			expected: "Nordmann",
		},
		{
			name:     "first name only",
			person:   PersonName{FirstName: "Ola"},
			expected: "Ola",
		},
		{
			name:     "empty",
			person:   PersonName{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.person.FullName()
			if result != tt.expected {
				t.Errorf("PersonName.FullName() = %q, want %q", result, tt.expected)
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
			name:     "with message",
			err:      APIError{Message: "Not found"},
			expected: "Not found",
		},
		{
			name:     "with error only",
			err:      APIError{Error: "Server error"},
			expected: "Server error",
		},
		{
			name:     "message takes precedence",
			err:      APIError{Error: "Generic error", Message: "Specific message"},
			expected: "Specific message",
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

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     *Address
		expected string
	}{
		{
			name:     "nil address",
			addr:     nil,
			expected: "",
		},
		{
			name: "full address",
			addr: &Address{
				AddressLines: []string{"Forusbeen 50"},
				PostalCode:   "4035",
				PostalPlace:  "STAVANGER",
			},
			expected: "Forusbeen 50, 4035 STAVANGER",
		},
		{
			name: "multiple address lines",
			addr: &Address{
				AddressLines: []string{"Building A", "Floor 2"},
				PostalCode:   "0283",
				PostalPlace:  "OSLO",
			},
			expected: "Building A, Floor 2, 0283 OSLO",
		},
		{
			name: "postal code and city only",
			addr: &Address{
				PostalCode:  "1234",
				PostalPlace: "SOMEWHERE",
			},
			expected: "1234 SOMEWHERE",
		},
		{
			name:     "empty address",
			addr:     &Address{},
			expected: "",
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

func TestParseUpdatesResponse(t *testing.T) {
	apiResponse := `{
		"_embedded": {
			"oppdaterteEnheter": [
				{
					"organisasjonsnummer": "923609016",
					"dato": "2024-01-15T10:30:00Z",
					"endringstype": "Endring"
				}
			]
		}
	}`

	var resp UpdatesResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse updates response: %v", err)
	}

	if len(resp.Embedded.Updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(resp.Embedded.Updates))
	}

	update := resp.Embedded.Updates[0]
	if update.OrganizationNumber != "923609016" {
		t.Errorf("OrganizationNumber = %q, want %q", update.OrganizationNumber, "923609016")
	}
	if update.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if update.ChangeType != "Endring" {
		t.Errorf("ChangeType = %q, want %q", update.ChangeType, "Endring")
	}
}

func TestParseRolesResponse(t *testing.T) {
	apiResponse := `{
		"rollegrupper": [
			{
				"type": {"kode": "STYR", "beskrivelse": "Styre"},
				"sistEndret": "2023-12-01",
				"roller": [
					{
						"type": {"kode": "LEDE", "beskrivelse": "Styreleder"},
						"person": {
							"navn": {"fornavn": "Ola", "mellomnavn": "Johan", "etternavn": "Nordmann"},
							"fodselsdato": "1970-05-15",
							"erDod": false
						},
						"fratraadt": false
					},
					{
						"type": {"kode": "MEDL", "beskrivelse": "Styremedlem"},
						"enhet": {
							"organisasjonsnummer": "912345678",
							"navn": ["HOLDING AS"]
						},
						"fratraadt": false
					}
				]
			}
		]
	}`

	var resp RolesResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse roles response: %v", err)
	}

	if len(resp.RoleGroups) != 1 {
		t.Fatalf("Expected 1 role group, got %d", len(resp.RoleGroups))
	}

	group := resp.RoleGroups[0]
	if group.Type.Code != "STYR" {
		t.Errorf("Type.Code = %q, want %q", group.Type.Code, "STYR")
	}
	if len(group.Roles) != 2 {
		t.Fatalf("Expected 2 roles, got %d", len(group.Roles))
	}

	// Check person role
	personRole := group.Roles[0]
	if personRole.Person == nil {
		t.Fatal("Expected person role to have person")
	}
	if personRole.Person.Name.FullName() != "Ola Johan Nordmann" {
		t.Errorf("Person.Name.FullName() = %q, want %q", personRole.Person.Name.FullName(), "Ola Johan Nordmann")
	}

	// Check entity role
	entityRole := group.Roles[1]
	if entityRole.Entity == nil {
		t.Fatal("Expected entity role to have entity")
	}
	if entityRole.Entity.OrganizationNumber != "912345678" {
		t.Errorf("Entity.OrganizationNumber = %q, want %q", entityRole.Entity.OrganizationNumber, "912345678")
	}
}

func TestParseCompanyFullResponse(t *testing.T) {
	apiResponse := `{
		"organisasjonsnummer": "923609016",
		"navn": "EQUINOR ASA",
		"organisasjonsform": {"kode": "ASA", "beskrivelse": "Allmennaksjeselskap"},
		"registreringsdatoEnhetsregisteret": "2018-02-01",
		"stiftelsedato": "1972-09-18",
		"postadresse": {
			"land": "Norge",
			"landkode": "NO",
			"postnummer": "4035",
			"poststed": "STAVANGER",
			"adresse": ["Forusbeen 50"]
		},
		"forretningsadresse": {
			"land": "Norge",
			"landkode": "NO",
			"postnummer": "4035",
			"poststed": "STAVANGER",
			"kommunenummer": "1103",
			"kommune": "STAVANGER",
			"adresse": ["Forusbeen 50"]
		},
		"registrertIMvaregisteret": true,
		"registrertIForetaksregisteret": true,
		"konkurs": false,
		"underAvvikling": false,
		"antallAnsatte": 21000,
		"harRegistrertAntallAnsatte": true,
		"hjemmeside": "www.equinor.com",
		"naeringskode1": {"kode": "06.100", "beskrivelse": "Utvinning av råolje"},
		"kapital": {"belop": 8089312534, "valuta": "NOK"}
	}`

	var company Company
	if err := json.Unmarshal([]byte(apiResponse), &company); err != nil {
		t.Fatalf("Failed to parse company response: %v", err)
	}

	if company.OrganizationNumber != "923609016" {
		t.Errorf("OrganizationNumber = %q, want %q", company.OrganizationNumber, "923609016")
	}
	if company.Name != "EQUINOR ASA" {
		t.Errorf("Name = %q, want %q", company.Name, "EQUINOR ASA")
	}
	if company.OrganizationForm == nil || company.OrganizationForm.Code != "ASA" {
		t.Error("OrganizationForm should be ASA")
	}
	if company.EmployeeCount != 21000 {
		t.Errorf("EmployeeCount = %d, want %d", company.EmployeeCount, 21000)
	}
	if company.Website != "www.equinor.com" {
		t.Errorf("Website = %q, want %q", company.Website, "www.equinor.com")
	}
	if !company.RegisteredInVAT {
		t.Error("RegisteredInVAT should be true")
	}
	if company.Bankrupt {
		t.Error("Bankrupt should be false")
	}
	if company.Capital == nil || company.Capital.Amount != 8089312534 {
		t.Error("Capital should have correct amount")
	}
	if company.IndustryCode1 == nil || company.IndustryCode1.Code != "06.100" {
		t.Error("IndustryCode1 should be 06.100")
	}
}

func TestParseSubUnitSearchResponse(t *testing.T) {
	apiResponse := `{
		"_embedded": {
			"underenheter": [
				{
					"organisasjonsnummer": "912345678",
					"navn": "EQUINOR ASA AVD OSLO",
					"overordnetEnhet": "923609016",
					"beliggenhetsadresse": {
						"postnummer": "0283",
						"poststed": "OSLO",
						"adresse": ["Drammensveien 264"]
					},
					"antallAnsatte": 500
				}
			]
		},
		"page": {"size": 20, "totalElements": 1, "totalPages": 1, "number": 0}
	}`

	var resp SubUnitSearchResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse subunit search response: %v", err)
	}

	if len(resp.Embedded.SubUnits) != 1 {
		t.Fatalf("Expected 1 subunit, got %d", len(resp.Embedded.SubUnits))
	}

	su := resp.Embedded.SubUnits[0]
	if su.OrganizationNumber != "912345678" {
		t.Errorf("OrganizationNumber = %q, want %q", su.OrganizationNumber, "912345678")
	}
	if su.ParentOrganizationNumber != "923609016" {
		t.Errorf("ParentOrganizationNumber = %q, want %q", su.ParentOrganizationNumber, "923609016")
	}
	if su.EmployeeCount != 500 {
		t.Errorf("EmployeeCount = %d, want %d", su.EmployeeCount, 500)
	}
	if resp.Page.TotalElements != 1 {
		t.Errorf("Page.TotalElements = %d, want %d", resp.Page.TotalElements, 1)
	}
}

func TestFormatAddressWithCountry(t *testing.T) {
	// Test address with non-Norwegian country
	addr := &Address{
		AddressLines: []string{"123 Main Street"},
		PostalCode:   "12345",
		PostalPlace:  "NEW YORK",
		Country:      "United States",
		CountryCode:  "US",
	}
	result := formatAddress(addr)
	expected := "123 Main Street, 12345 NEW YORK, United States"
	if result != expected {
		t.Errorf("formatAddress() = %q, want %q", result, expected)
	}
}

func TestFormatAddressNorway(t *testing.T) {
	// Test Norwegian address (should not include country)
	addr := &Address{
		AddressLines: []string{"Forusbeen 50"},
		PostalCode:   "4035",
		PostalPlace:  "STAVANGER",
		Country:      "Norge",
		CountryCode:  "NO",
	}
	result := formatAddress(addr)
	expected := "Forusbeen 50, 4035 STAVANGER"
	if result != expected {
		t.Errorf("formatAddress() = %q, want %q", result, expected)
	}
}

func TestSearchOptionsBoolPointers(t *testing.T) {
	// Test with bool pointers
	vatTrue := true
	bankruptFalse := false
	opts := &SearchOptions{
		Page:            1,
		Size:            50,
		OrgForm:         "AS",
		Municipality:    "0301",
		RegisteredInVAT: &vatTrue,
		Bankrupt:        &bankruptFalse,
	}
	if opts.Page != 1 {
		t.Errorf("Page = %d, want %d", opts.Page, 1)
	}
	if opts.RegisteredInVAT == nil || *opts.RegisteredInVAT != true {
		t.Error("RegisteredInVAT should be true")
	}
	if opts.Bankrupt == nil || *opts.Bankrupt != false {
		t.Error("Bankrupt should be false")
	}
}

func TestNewClientWithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := NewClient(WithLogger(logger))
	defer client.Close()

	if client.Logger != logger {
		t.Error("custom logger was not set")
	}
}

func TestNewClientWithCache(t *testing.T) {
	cache := infra.NewCache(500)
	defer cache.Close()

	client := NewClient(WithCache(cache))
	defer client.Close()

	if client.Cache != cache {
		t.Error("custom cache was not set")
	}
}

func TestParseMunicipalitiesResponse(t *testing.T) {
	apiResponse := `{
		"_embedded": {
			"kommuner": [
				{"nummer": "0301", "navn": "Oslo"},
				{"nummer": "4601", "navn": "Bergen"},
				{"nummer": "5001", "navn": "Trondheim"}
			]
		}
	}`

	var resp MunicipalitiesResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(resp.Embedded.Municipalities) != 3 {
		t.Fatalf("Expected 3 municipalities, got %d", len(resp.Embedded.Municipalities))
	}

	if resp.Embedded.Municipalities[0].Number != "0301" {
		t.Errorf("Number = %q, want %q", resp.Embedded.Municipalities[0].Number, "0301")
	}
	if resp.Embedded.Municipalities[0].Name != "Oslo" {
		t.Errorf("Name = %q, want %q", resp.Embedded.Municipalities[0].Name, "Oslo")
	}
}

func TestParseOrgFormsResponse(t *testing.T) {
	apiResponse := `{
		"_embedded": {
			"organisasjonsformer": [
				{"kode": "AS", "beskrivelse": "Aksjeselskap"},
				{"kode": "ENK", "beskrivelse": "Enkeltpersonforetak"},
				{"kode": "NUF", "beskrivelse": "Norskregistrert utenlandsk foretak"}
			]
		}
	}`

	var resp OrgFormsResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(resp.Embedded.OrgForms) != 3 {
		t.Fatalf("Expected 3 org forms, got %d", len(resp.Embedded.OrgForms))
	}

	if resp.Embedded.OrgForms[0].Code != "AS" {
		t.Errorf("Code = %q, want %q", resp.Embedded.OrgForms[0].Code, "AS")
	}
	if resp.Embedded.OrgForms[0].Description != "Aksjeselskap" {
		t.Errorf("Description = %q, want %q", resp.Embedded.OrgForms[0].Description, "Aksjeselskap")
	}
}

func TestParseSubUnitUpdatesResponse(t *testing.T) {
	apiResponse := `{
		"_embedded": {
			"oppdaterteUnderenheter": [
				{
					"oppdateringsid": 100,
					"organisasjonsnummer": "912345678",
					"dato": "2024-01-15T10:00:00.000Z",
					"endringstype": "Ny"
				},
				{
					"oppdateringsid": 101,
					"organisasjonsnummer": "912345679",
					"dato": "2024-01-15T11:00:00.000Z",
					"endringstype": "Endring"
				}
			]
		}
	}`

	var resp SubUnitUpdatesResponse
	if err := json.Unmarshal([]byte(apiResponse), &resp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(resp.Embedded.Updates) != 2 {
		t.Fatalf("Expected 2 updates, got %d", len(resp.Embedded.Updates))
	}

	if resp.Embedded.Updates[0].UpdateID != 100 {
		t.Errorf("UpdateID = %d, want %d", resp.Embedded.Updates[0].UpdateID, 100)
	}
	if resp.Embedded.Updates[0].OrganizationNumber != "912345678" {
		t.Errorf("OrganizationNumber = %q, want %q", resp.Embedded.Updates[0].OrganizationNumber, "912345678")
	}
	if resp.Embedded.Updates[0].ChangeType != "Ny" {
		t.Errorf("ChangeType = %q, want %q", resp.Embedded.Updates[0].ChangeType, "Ny")
	}
	if resp.Embedded.Updates[1].ChangeType != "Endring" {
		t.Errorf("ChangeType = %q, want %q", resp.Embedded.Updates[1].ChangeType, "Endring")
	}
}

func TestFormatAddressNil(t *testing.T) {
	result := formatAddress(nil)
	if result != "" {
		t.Errorf("formatAddress(nil) = %q, want %q", result, "")
	}
}

func TestFormatAddressEmptyLines(t *testing.T) {
	// Test address with empty lines
	addr := &Address{
		AddressLines: []string{"", "123 Main St", ""},
		PostalCode:   "1234",
		PostalPlace:  "OSLO",
		CountryCode:  "NO",
	}
	result := formatAddress(addr)
	expected := "123 Main St, 1234 OSLO"
	if result != expected {
		t.Errorf("formatAddress() = %q, want %q", result, expected)
	}
}

func TestFormatAddressMultipleLines(t *testing.T) {
	// Test address with multiple address lines
	addr := &Address{
		AddressLines: []string{"Apt 4B", "123 Main St", "Building A"},
		PostalCode:   "1234",
		PostalPlace:  "OSLO",
		CountryCode:  "NO",
	}
	result := formatAddress(addr)
	expected := "Apt 4B, 123 Main St, Building A, 1234 OSLO"
	if result != expected {
		t.Errorf("formatAddress() = %q, want %q", result, expected)
	}
}

func TestFormatAddressOnlyPostal(t *testing.T) {
	// Test address with only postal info
	addr := &Address{
		PostalCode:  "0001",
		PostalPlace: "OSLO",
		CountryCode: "NO",
	}
	result := formatAddress(addr)
	expected := "0001 OSLO"
	if result != expected {
		t.Errorf("formatAddress() = %q, want %q", result, expected)
	}
}

func TestPersonNameFullNameVariations(t *testing.T) {
	tests := []struct {
		name     string
		person   PersonName
		expected string
	}{
		{
			name:     "first and last only",
			person:   PersonName{FirstName: "John", LastName: "Doe"},
			expected: "John Doe",
		},
		{
			name:     "with middle name",
			person:   PersonName{FirstName: "John", MiddleName: "William", LastName: "Doe"},
			expected: "John William Doe",
		},
		{
			name:     "first only",
			person:   PersonName{FirstName: "John"},
			expected: "John",
		},
		{
			name:     "last only",
			person:   PersonName{LastName: "Doe"},
			expected: "Doe",
		},
		{
			name:     "empty",
			person:   PersonName{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.person.FullName()
			if result != tt.expected {
				t.Errorf("FullName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSearchSubUnitsOptions(t *testing.T) {
	opts := &SearchSubUnitsOptions{
		Page:         2,
		Size:         50,
		Municipality: "0301",
	}

	if opts.Page != 2 {
		t.Errorf("Page = %d, want %d", opts.Page, 2)
	}
	if opts.Size != 50 {
		t.Errorf("Size = %d, want %d", opts.Size, 50)
	}
	if opts.Municipality != "0301" {
		t.Errorf("Municipality = %q, want %q", opts.Municipality, "0301")
	}
}

func TestUpdatesOptions(t *testing.T) {
	opts := &UpdatesOptions{
		Size: 100,
	}

	if opts.Size != 100 {
		t.Errorf("Size = %d, want %d", opts.Size, 100)
	}
}

// HTTP Mocking Tests
// These tests use httptest to mock the API and verify actual client behavior

func TestSearchCompanies_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/enheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("navn") != "Equinor" {
			t.Errorf("unexpected query param navn: %s", r.URL.Query().Get("navn"))
		}

		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{
				Companies: []Company{
					{
						OrganizationNumber: "923609016",
						Name:               "EQUINOR ASA",
					},
				},
			},
			Page: PageInfo{
				TotalElements: 1,
				TotalPages:    1,
				Number:        0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompanies(context.Background(), "Equinor", nil)
	if err != nil {
		t.Fatalf("SearchCompanies failed: %v", err)
	}

	if len(result.Embedded.Companies) != 1 {
		t.Fatalf("Expected 1 company, got %d", len(result.Embedded.Companies))
	}
	if result.Embedded.Companies[0].Name != "EQUINOR ASA" {
		t.Errorf("Company name = %q, want %q", result.Embedded.Companies[0].Name, "EQUINOR ASA")
	}
}

func TestGetCompany_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/enheter/923609016" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		company := Company{
			OrganizationNumber: "923609016",
			Name:               "EQUINOR ASA",
			EmployeeCount:      21000,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompany(context.Background(), "923609016")
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}

	if result.Name != "EQUINOR ASA" {
		t.Errorf("Company name = %q, want %q", result.Name, "EQUINOR ASA")
	}
	if result.EmployeeCount != 21000 {
		t.Errorf("EmployeeCount = %d, want %d", result.EmployeeCount, 21000)
	}
}

func TestGetCompany_NotFound_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"feilmelding": "Ingen enhet funnet"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompany(context.Background(), "000000000")
	if err == nil {
		t.Fatal("Expected error for not found, got nil")
	}
}

func TestGetRoles_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/enheter/923609016/roller" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := RolesResponse{
			RoleGroups: []RoleGroup{
				{
					Type: RoleType{Code: "STYR", Description: "Styre"},
					Roles: []Role{
						{
							Type: RoleType{Code: "LEDE", Description: "Styreleder"},
							Person: &Person{
								Name: PersonName{FirstName: "Ola", LastName: "Nordmann"},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetRoles(context.Background(), "923609016")
	if err != nil {
		t.Fatalf("GetRoles failed: %v", err)
	}

	if len(result.RoleGroups) != 1 {
		t.Fatalf("Expected 1 role group, got %d", len(result.RoleGroups))
	}
	if result.RoleGroups[0].Type.Code != "STYR" {
		t.Errorf("RoleGroup type = %q, want %q", result.RoleGroups[0].Type.Code, "STYR")
	}
}

func TestGetSubUnits_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/underenheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("overordnetEnhet") != "923609016" {
			t.Errorf("unexpected query param: %s", r.URL.Query().Get("overordnetEnhet"))
		}

		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{
				SubUnits: []SubUnit{
					{
						OrganizationNumber:       "912345678",
						Name:                     "EQUINOR AVD OSLO",
						ParentOrganizationNumber: "923609016",
					},
				},
			},
			Page: PageInfo{TotalElements: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSubUnits(context.Background(), "923609016")
	if err != nil {
		t.Fatalf("GetSubUnits failed: %v", err)
	}

	if len(result.Embedded.SubUnits) != 1 {
		t.Fatalf("Expected 1 subunit, got %d", len(result.Embedded.SubUnits))
	}
}

func TestGetSubUnit_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/underenheter/912345678" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		su := SubUnit{
			OrganizationNumber:       "912345678",
			Name:                     "EQUINOR AVD OSLO",
			ParentOrganizationNumber: "923609016",
			EmployeeCount:            500,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(su)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSubUnit(context.Background(), "912345678")
	if err != nil {
		t.Fatalf("GetSubUnit failed: %v", err)
	}

	if result.Name != "EQUINOR AVD OSLO" {
		t.Errorf("SubUnit name = %q, want %q", result.Name, "EQUINOR AVD OSLO")
	}
}

func TestGetUpdates_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oppdateringer/enheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := UpdatesResponse{
			Embedded: struct {
				Updates []UpdateEntry `json:"oppdaterteEnheter"`
			}{
				Updates: []UpdateEntry{
					{
						UpdateID:           123,
						OrganizationNumber: "923609016",
						UpdatedAt:          time.Now(),
						ChangeType:         "Endring",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetUpdates(context.Background(), time.Now().Add(-24*time.Hour), nil)
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}

	if len(result.Embedded.Updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(result.Embedded.Updates))
	}
}

func TestSearchSubUnits_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/underenheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("navn") != "Oslo" {
			t.Errorf("unexpected query param: %s", r.URL.Query().Get("navn"))
		}

		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{
				SubUnits: []SubUnit{
					{
						OrganizationNumber: "912345678",
						Name:               "COMPANY AVD OSLO",
					},
				},
			},
			Page: PageInfo{TotalElements: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchSubUnits(context.Background(), "Oslo", nil)
	if err != nil {
		t.Fatalf("SearchSubUnits failed: %v", err)
	}

	if len(result.Embedded.SubUnits) != 1 {
		t.Fatalf("Expected 1 subunit, got %d", len(result.Embedded.SubUnits))
	}
}

func TestGetMunicipalities_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kommuner" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		// Verify size parameter is sent to fetch all municipalities
		if size := r.URL.Query().Get("size"); size != "500" {
			t.Errorf("expected size=500, got size=%s", size)
		}

		resp := MunicipalitiesResponse{
			Embedded: struct {
				Municipalities []Municipality `json:"kommuner"`
			}{
				Municipalities: []Municipality{
					{Number: "0301", Name: "Oslo"},
					{Number: "4601", Name: "Bergen"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetMunicipalities(context.Background())
	if err != nil {
		t.Fatalf("GetMunicipalities failed: %v", err)
	}

	if len(result.Embedded.Municipalities) != 2 {
		t.Fatalf("Expected 2 municipalities, got %d", len(result.Embedded.Municipalities))
	}
}

func TestGetOrgForms_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organisasjonsformer" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := OrgFormsResponse{
			Embedded: struct {
				OrgForms []OrgForm `json:"organisasjonsformer"`
			}{
				OrgForms: []OrgForm{
					{Code: "AS", Description: "Aksjeselskap"},
					{Code: "ENK", Description: "Enkeltpersonforetak"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetOrgForms(context.Background())
	if err != nil {
		t.Fatalf("GetOrgForms failed: %v", err)
	}

	if len(result.Embedded.OrgForms) != 2 {
		t.Fatalf("Expected 2 org forms, got %d", len(result.Embedded.OrgForms))
	}
}

func TestGetSubUnitUpdates_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oppdateringer/underenheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := SubUnitUpdatesResponse{
			Embedded: struct {
				Updates []SubUnitUpdateEntry `json:"oppdaterteUnderenheter"`
			}{
				Updates: []SubUnitUpdateEntry{
					{
						UpdateID:           456,
						OrganizationNumber: "912345678",
						UpdatedAt:          time.Now(),
						ChangeType:         "Ny",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSubUnitUpdates(context.Background(), time.Now().Add(-24*time.Hour), nil)
	if err != nil {
		t.Fatalf("GetSubUnitUpdates failed: %v", err)
	}

	if len(result.Embedded.Updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(result.Embedded.Updates))
	}
}

func TestSearchCompanies_WithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all options are passed
		query := r.URL.Query()
		if query.Get("page") != "2" {
			t.Errorf("page = %q, want %q", query.Get("page"), "2")
		}
		if query.Get("size") != "50" {
			t.Errorf("size = %q, want %q", query.Get("size"), "50")
		}
		if query.Get("organisasjonsform") != "AS" {
			t.Errorf("organisasjonsform = %q, want %q", query.Get("organisasjonsform"), "AS")
		}
		if query.Get("kommunenummer") != "0301" {
			t.Errorf("kommunenummer = %q, want %q", query.Get("kommunenummer"), "0301")
		}

		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{},
			Page: PageInfo{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	vatTrue := true
	_, err := client.SearchCompanies(context.Background(), "Test", &SearchOptions{
		Page:            2,
		Size:            50,
		OrgForm:         "AS",
		Municipality:    "0301",
		RegisteredInVAT: &vatTrue,
	})
	if err != nil {
		t.Fatalf("SearchCompanies failed: %v", err)
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

	_, err := client.GetCompany(context.Background(), "923609016")
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
	// Should have retried
	if attempts < 2 {
		t.Errorf("Expected at least 2 attempts (retry), got %d", attempts)
	}
}

func TestClient_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		company := Company{
			OrganizationNumber: "923609016",
			Name:               "EQUINOR ASA",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, err := client.GetCompany(context.Background(), "923609016")
	if err != nil {
		t.Fatalf("First GetCompany failed: %v", err)
	}

	// Second call should hit cache
	_, err = client.GetCompany(context.Background(), "923609016")
	if err != nil {
		t.Fatalf("Second GetCompany failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestWithBaseURL(t *testing.T) {
	client := NewClient(WithBaseURL("http://test.example.com"))
	defer client.Close()

	if client.baseURL != "http://test.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "http://test.example.com")
	}
}

func TestGetStatus(t *testing.T) {
	tests := []struct {
		name             string
		bankrupt         bool
		underLiquidation bool
		expected         string
	}{
		{
			name:             "active company",
			bankrupt:         false,
			underLiquidation: false,
			expected:         "ACTIVE",
		},
		{
			name:             "bankrupt company",
			bankrupt:         true,
			underLiquidation: false,
			expected:         "BANKRUPT",
		},
		{
			name:             "company under liquidation",
			bankrupt:         false,
			underLiquidation: true,
			expected:         "LIQUIDATING",
		},
		{
			name:             "bankrupt takes priority over liquidation",
			bankrupt:         true,
			underLiquidation: true,
			expected:         "BANKRUPT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getStatus(tt.bankrupt, tt.underLiquidation)
			if status != tt.expected {
				t.Errorf("getStatus(%v, %v) = %q, want %q", tt.bankrupt, tt.underLiquidation, status, tt.expected)
			}
		})
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestValidateAndNormalizeOrgNumber(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantResult string
		wantErr    bool
	}{
		{"valid 9 digits", "923609016", "923609016", false},
		{"valid with spaces", "923 609 016", "923609016", false},
		{"valid with dashes", "923-609-016", "923609016", false},
		{"empty string", "", "", true},
		{"too short", "12345678", "", true},
		{"too long", "1234567890", "", true},
		{"with letters", "92360901A", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndNormalizeOrgNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndNormalizeOrgNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.wantResult {
				t.Errorf("ValidateAndNormalizeOrgNumber(%q) = %q, want %q", tt.input, result, tt.wantResult)
			}
		})
	}
}

func TestValidateSearchQuery_MaxLength(t *testing.T) {
	// Test query exceeding max length
	longQuery := make([]byte, MaxQueryLength+1)
	for i := range longQuery {
		longQuery[i] = 'a'
	}
	err := ValidateSearchQuery(string(longQuery))
	if err == nil {
		t.Error("Expected error for query exceeding max length")
	}
}

// =============================================================================
// BatchGetCompanies Tests
// =============================================================================

func TestBatchGetCompanies_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/enheter" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify org numbers are passed
		orgNumbers := r.URL.Query().Get("organisasjonsnummer")
		if !strings.Contains(orgNumbers, "923609016") {
			t.Errorf("expected org number in query, got: %s", orgNumbers)
		}

		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{
				Companies: []Company{
					{
						OrganizationNumber: "923609016",
						Name:               "EQUINOR ASA",
					},
					{
						OrganizationNumber: "914778271",
						Name:               "TELENOR ASA",
					},
				},
			},
			Page: PageInfo{TotalElements: 2},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.BatchGetCompanies(context.Background(), []string{"923609016", "914778271"})
	if err != nil {
		t.Fatalf("BatchGetCompanies failed: %v", err)
	}

	if len(result.Embedded.Companies) != 2 {
		t.Fatalf("Expected 2 companies, got %d", len(result.Embedded.Companies))
	}
}

func TestBatchGetCompanies_EmptyInput(t *testing.T) {
	client := NewClient()
	defer client.Close()

	result, err := client.BatchGetCompanies(context.Background(), []string{})
	if err != nil {
		t.Fatalf("BatchGetCompanies failed: %v", err)
	}
	if len(result.Embedded.Companies) != 0 {
		t.Errorf("Expected empty result, got %d companies", len(result.Embedded.Companies))
	}
}

func TestBatchGetCompanies_TooMany(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Create slice with more than 2000 org numbers
	orgNumbers := make([]string, 2001)
	for i := range orgNumbers {
		orgNumbers[i] = "923609016"
	}

	_, err := client.BatchGetCompanies(context.Background(), orgNumbers)
	if err == nil {
		t.Error("Expected error for too many org numbers")
	}
}

func TestBatchGetCompanies_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.BatchGetCompanies(context.Background(), []string{"invalid"})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

func TestBatchGetCompanies_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{
				Companies: []Company{{OrganizationNumber: "923609016", Name: "TEST"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.BatchGetCompanies(context.Background(), []string{"923609016"})
	// Second call should hit cache
	_, _ = client.BatchGetCompanies(context.Background(), []string{"923609016"})

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

// =============================================================================
// MCP Wrapper Tests
// =============================================================================

func TestSearchCompaniesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{
				Companies: []Company{
					{
						OrganizationNumber: "923609016",
						Name:               "EQUINOR ASA",
						OrganizationForm:   &OrganizationForm{Code: "ASA", Description: "Allmennaksjeselskap"},
						BusinessAddress: &Address{
							AddressLines: []string{"Forusbeen 50"},
							PostalCode:   "4035",
							PostalPlace:  "STAVANGER",
						},
						PostalAddress: &Address{
							AddressLines: []string{"Postboks 8500"},
							PostalCode:   "4035",
							PostalPlace:  "STAVANGER",
						},
					},
				},
			},
			Page: PageInfo{TotalElements: 1, TotalPages: 1, Number: 0},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchCompaniesMCP(context.Background(), SearchCompaniesArgs{
		Query: "Equinor",
	})
	if err != nil {
		t.Fatalf("SearchCompaniesMCP failed: %v", err)
	}

	if len(result.Companies) != 1 {
		t.Fatalf("Expected 1 company, got %d", len(result.Companies))
	}
	if result.Companies[0].Name != "EQUINOR ASA" {
		t.Errorf("Name = %q, want %q", result.Companies[0].Name, "EQUINOR ASA")
	}
	if result.Companies[0].OrganizationForm != "ASA" {
		t.Errorf("OrganizationForm = %q, want %q", result.Companies[0].OrganizationForm, "ASA")
	}
	if result.TotalResults != 1 {
		t.Errorf("TotalResults = %d, want 1", result.TotalResults)
	}
}

func TestSearchCompaniesMCP_EmptyQuery(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchCompaniesMCP(context.Background(), SearchCompaniesArgs{Query: ""})
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchCompaniesMCP_ShortQuery(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchCompaniesMCP(context.Background(), SearchCompaniesArgs{Query: "a"})
	if err == nil {
		t.Error("Expected error for query too short")
	}
}

func TestSearchCompaniesMCP_InvalidSize(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchCompaniesMCP(context.Background(), SearchCompaniesArgs{
		Query: "Equinor",
		Size:  150, // exceeds max 100
	})
	if err == nil {
		t.Error("Expected error for size exceeding 100")
	}
}

func TestSearchCompaniesMCP_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "1" {
			t.Errorf("page = %q, want %q", query.Get("page"), "1")
		}
		if query.Get("size") != "50" {
			t.Errorf("size = %q, want %q", query.Get("size"), "50")
		}
		if query.Get("organisasjonsform") != "AS" {
			t.Errorf("organisasjonsform = %q, want %q", query.Get("organisasjonsform"), "AS")
		}
		if query.Get("kommunenummer") != "0301" {
			t.Errorf("kommunenummer = %q, want %q", query.Get("kommunenummer"), "0301")
		}

		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{Companies: []Company{}},
			Page: PageInfo{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	vatTrue := true
	bankruptFalse := false
	voluntaryTrue := true
	_, err := client.SearchCompaniesMCP(context.Background(), SearchCompaniesArgs{
		Query:                 "Test",
		Page:                  1,
		Size:                  50,
		OrgForm:               "AS",
		Municipality:          "0301",
		RegisteredInVAT:       &vatTrue,
		Bankrupt:              &bankruptFalse,
		RegisteredInVoluntary: &voluntaryTrue,
	})
	if err != nil {
		t.Fatalf("SearchCompaniesMCP failed: %v", err)
	}
}

func TestGetCompanyMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			OrganizationNumber:    "923609016",
			Name:                  "EQUINOR ASA",
			OrganizationForm:      &OrganizationForm{Code: "ASA", Description: "Allmennaksjeselskap"},
			RegistrationDate:      "1972-06-16",
			EmployeeCount:         21000,
			Website:               "www.equinor.com",
			RegisteredInVAT:       true,
			RegisteredInVoluntary: false,
			IndustryCode1:         &IndustryCode{Code: "06.100", Description: "Utvinning av råolje"},
			BusinessAddress: &Address{
				AddressLines: []string{"Forusbeen 50"},
				PostalCode:   "4035",
				PostalPlace:  "STAVANGER",
			},
			PostalAddress: &Address{
				AddressLines: []string{"Postboks 8500"},
				PostalCode:   "4035",
				PostalPlace:  "STAVANGER",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: "923609016"})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Summary == nil {
		t.Fatal("Expected summary, got nil")
	}
	if result.Summary.Name != "EQUINOR ASA" {
		t.Errorf("Name = %q, want %q", result.Summary.Name, "EQUINOR ASA")
	}
	if result.Summary.EmployeeCount != 21000 {
		t.Errorf("EmployeeCount = %d, want 21000", result.Summary.EmployeeCount)
	}
	if result.Summary.Status != "ACTIVE" {
		t.Errorf("Status = %q, want ACTIVE", result.Summary.Status)
	}
}

func TestGetCompanyMCP_FullDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			OrganizationNumber: "923609016",
			Name:               "EQUINOR ASA",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(company)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{
		OrgNumber: "923609016",
		Full:      true,
	})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Company == nil {
		t.Fatal("Expected full company, got nil")
	}
	if result.Summary != nil {
		t.Error("Expected nil summary when full=true")
	}
}

func TestGetCompanyMCP_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: "invalid"})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

func TestGetRolesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RolesResponse{
			RoleGroups: []RoleGroup{
				{
					Type:         RoleType{Code: "STYR", Description: "Styre"},
					LastModified: "2024-01-15",
					Roles: []Role{
						{
							Type:   RoleType{Code: "LEDE", Description: "Styreleder"},
							Person: &Person{Name: PersonName{FirstName: "Ola", LastName: "Nordmann"}, BirthDate: "1970-01-01"},
						},
						{
							Type:   RoleType{Code: "MEDL", Description: "Styremedlem"},
							Entity: &RoleEntity{OrganizationNumber: "912345678", Name: []string{"HOLDING", "AS"}},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetRolesMCP(context.Background(), GetRolesArgs{OrgNumber: "923609016"})
	if err != nil {
		t.Fatalf("GetRolesMCP failed: %v", err)
	}

	if len(result.RoleGroups) != 1 {
		t.Fatalf("Expected 1 role group, got %d", len(result.RoleGroups))
	}
	if len(result.RoleGroups[0].Roles) != 2 {
		t.Fatalf("Expected 2 roles, got %d", len(result.RoleGroups[0].Roles))
	}
	// Check person role
	if result.RoleGroups[0].Roles[0].Name != "Ola Nordmann" {
		t.Errorf("Role name = %q, want %q", result.RoleGroups[0].Roles[0].Name, "Ola Nordmann")
	}
	// Check entity role
	if result.RoleGroups[0].Roles[1].EntityOrgNr != "912345678" {
		t.Errorf("EntityOrgNr = %q, want %q", result.RoleGroups[0].Roles[1].EntityOrgNr, "912345678")
	}
	if result.RoleGroups[0].Roles[1].Name != "HOLDING AS" {
		t.Errorf("Entity name = %q, want %q", result.RoleGroups[0].Roles[1].Name, "HOLDING AS")
	}
}

func TestGetRolesMCP_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetRolesMCP(context.Background(), GetRolesArgs{OrgNumber: "invalid"})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

func TestGetSubUnitsMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{
				SubUnits: []SubUnit{
					{
						OrganizationNumber:       "912345678",
						Name:                     "EQUINOR AVD OSLO",
						ParentOrganizationNumber: "923609016",
						EmployeeCount:            500,
						BusinessAddress: &Address{
							AddressLines: []string{"Drammensveien 264"},
							PostalCode:   "0283",
							PostalPlace:  "OSLO",
						},
					},
				},
			},
			Page: PageInfo{TotalElements: 1},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSubUnitsMCP(context.Background(), GetSubUnitsArgs{ParentOrgNumber: "923609016"})
	if err != nil {
		t.Fatalf("GetSubUnitsMCP failed: %v", err)
	}

	if len(result.SubUnits) != 1 {
		t.Fatalf("Expected 1 subunit, got %d", len(result.SubUnits))
	}
	if result.SubUnits[0].Name != "EQUINOR AVD OSLO" {
		t.Errorf("Name = %q, want %q", result.SubUnits[0].Name, "EQUINOR AVD OSLO")
	}
	if result.TotalResults != 1 {
		t.Errorf("TotalResults = %d, want 1", result.TotalResults)
	}
}

func TestGetSubUnitsMCP_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetSubUnitsMCP(context.Background(), GetSubUnitsArgs{ParentOrgNumber: "invalid"})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

func TestGetSubUnitMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		su := SubUnit{
			OrganizationNumber:       "912345678",
			Name:                     "EQUINOR AVD OSLO",
			ParentOrganizationNumber: "923609016",
			EmployeeCount:            500,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(su)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSubUnitMCP(context.Background(), GetSubUnitArgs{OrgNumber: "912345678"})
	if err != nil {
		t.Fatalf("GetSubUnitMCP failed: %v", err)
	}

	if result.SubUnit == nil {
		t.Fatal("SubUnit is nil")
	}
	if result.SubUnit.Name != "EQUINOR AVD OSLO" {
		t.Errorf("Name = %q, want %q", result.SubUnit.Name, "EQUINOR AVD OSLO")
	}
}

func TestGetSubUnitMCP_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetSubUnitMCP(context.Background(), GetSubUnitArgs{OrgNumber: "invalid"})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

func TestGetUpdatesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := UpdatesResponse{
			Embedded: struct {
				Updates []UpdateEntry `json:"oppdaterteEnheter"`
			}{
				Updates: []UpdateEntry{
					{
						UpdateID:           123,
						OrganizationNumber: "923609016",
						UpdatedAt:          time.Now(),
						ChangeType:         "Endring",
					},
					{
						UpdateID:           124,
						OrganizationNumber: "914778271",
						UpdatedAt:          time.Now(),
						ChangeType:         "Ny",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetUpdatesMCP(context.Background(), GetUpdatesArgs{
		Since: time.Now().Add(-24 * time.Hour),
		Size:  100,
	})
	if err != nil {
		t.Fatalf("GetUpdatesMCP failed: %v", err)
	}

	if len(result.Updates) != 2 {
		t.Fatalf("Expected 2 updates, got %d", len(result.Updates))
	}
	if result.Updates[0].ChangeType != "Endring" {
		t.Errorf("ChangeType = %q, want %q", result.Updates[0].ChangeType, "Endring")
	}
}

func TestSearchSubUnitsMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{
				SubUnits: []SubUnit{
					{
						OrganizationNumber:       "912345678",
						Name:                     "COMPANY AVD OSLO",
						ParentOrganizationNumber: "923609016",
						EmployeeCount:            100,
						BusinessAddress: &Address{
							AddressLines: []string{"Test 1"},
							PostalCode:   "0001",
							PostalPlace:  "OSLO",
						},
					},
				},
			},
			Page: PageInfo{TotalElements: 1, TotalPages: 1, Number: 0},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.SearchSubUnitsMCP(context.Background(), SearchSubUnitsArgs{
		Query:        "Oslo",
		Page:         0,
		Size:         20,
		Municipality: "0301",
	})
	if err != nil {
		t.Fatalf("SearchSubUnitsMCP failed: %v", err)
	}

	if len(result.SubUnits) != 1 {
		t.Fatalf("Expected 1 subunit, got %d", len(result.SubUnits))
	}
	if result.TotalResults != 1 {
		t.Errorf("TotalResults = %d, want 1", result.TotalResults)
	}
}

func TestSearchSubUnitsMCP_EmptyQuery(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchSubUnitsMCP(context.Background(), SearchSubUnitsArgs{Query: ""})
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestSearchSubUnitsMCP_InvalidSize(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.SearchSubUnitsMCP(context.Background(), SearchSubUnitsArgs{
		Query: "Oslo",
		Size:  150,
	})
	if err == nil {
		t.Error("Expected error for invalid size")
	}
}

func TestListMunicipalitiesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MunicipalitiesResponse{
			Embedded: struct {
				Municipalities []Municipality `json:"kommuner"`
			}{
				Municipalities: []Municipality{
					{Number: "0301", Name: "Oslo"},
					{Number: "4601", Name: "Bergen"},
					{Number: "5001", Name: "Trondheim"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.ListMunicipalitiesMCP(context.Background(), ListMunicipalitiesArgs{})
	if err != nil {
		t.Fatalf("ListMunicipalitiesMCP failed: %v", err)
	}

	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}
	if len(result.Municipalities) != 3 {
		t.Fatalf("Expected 3 municipalities, got %d", len(result.Municipalities))
	}
	if result.Municipalities[0].Number != "0301" {
		t.Errorf("Number = %q, want %q", result.Municipalities[0].Number, "0301")
	}
}

func TestListOrgFormsMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OrgFormsResponse{
			Embedded: struct {
				OrgForms []OrgForm `json:"organisasjonsformer"`
			}{
				OrgForms: []OrgForm{
					{Code: "AS", Description: "Aksjeselskap"},
					{Code: "ENK", Description: "Enkeltpersonforetak"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.ListOrgFormsMCP(context.Background(), ListOrgFormsArgs{})
	if err != nil {
		t.Fatalf("ListOrgFormsMCP failed: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
	if len(result.OrgForms) != 2 {
		t.Fatalf("Expected 2 org forms, got %d", len(result.OrgForms))
	}
	if result.OrgForms[0].Code != "AS" {
		t.Errorf("Code = %q, want %q", result.OrgForms[0].Code, "AS")
	}
}

func TestGetSubUnitUpdatesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SubUnitUpdatesResponse{
			Embedded: struct {
				Updates []SubUnitUpdateEntry `json:"oppdaterteUnderenheter"`
			}{
				Updates: []SubUnitUpdateEntry{
					{
						UpdateID:           456,
						OrganizationNumber: "912345678",
						UpdatedAt:          time.Now(),
						ChangeType:         "Ny",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSubUnitUpdatesMCP(context.Background(), GetSubUnitUpdatesArgs{
		Since: time.Now().Add(-24 * time.Hour),
		Size:  50,
	})
	if err != nil {
		t.Fatalf("GetSubUnitUpdatesMCP failed: %v", err)
	}

	if len(result.Updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(result.Updates))
	}
	if result.Updates[0].ChangeType != "Ny" {
		t.Errorf("ChangeType = %q, want %q", result.Updates[0].ChangeType, "Ny")
	}
}

func TestGetSignatureRightsMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RolesResponse{
			RoleGroups: []RoleGroup{
				{
					Type: RoleType{Code: "SIGN", Description: "Signatur"},
					Roles: []Role{
						{
							Type:   RoleType{Code: "SIGN", Description: "Signaturrett"},
							Person: &Person{Name: PersonName{FirstName: "Ola", LastName: "Nordmann"}, BirthDate: "1970-01-01"},
						},
					},
				},
				{
					Type: RoleType{Code: "PROK", Description: "Prokura"},
					Roles: []Role{
						{
							Type:   RoleType{Code: "PROK", Description: "Prokura"},
							Person: &Person{Name: PersonName{FirstName: "Kari", LastName: "Hansen"}, BirthDate: "1980-05-15"},
						},
					},
				},
				{
					Type: RoleType{Code: "STYR", Description: "Styre"},
					Roles: []Role{
						{
							Type:     RoleType{Code: "LEDE", Description: "Styreleder"},
							Person:   &Person{Name: PersonName{FirstName: "Per", LastName: "Olsen"}},
							Resigned: true, // Should be excluded
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSignatureRightsMCP(context.Background(), GetSignatureRightsArgs{OrgNumber: "923609016"})
	if err != nil {
		t.Fatalf("GetSignatureRightsMCP failed: %v", err)
	}

	if result.OrganizationNumber != "923609016" {
		t.Errorf("OrganizationNumber = %q, want %q", result.OrganizationNumber, "923609016")
	}
	if len(result.SignatureRights) != 1 {
		t.Fatalf("Expected 1 signature right, got %d", len(result.SignatureRights))
	}
	if result.SignatureRights[0].Name != "Ola Nordmann" {
		t.Errorf("SignatureRight name = %q, want %q", result.SignatureRights[0].Name, "Ola Nordmann")
	}
	if len(result.Prokura) != 1 {
		t.Fatalf("Expected 1 prokura, got %d", len(result.Prokura))
	}
	if result.Prokura[0].Name != "Kari Hansen" {
		t.Errorf("Prokura name = %q, want %q", result.Prokura[0].Name, "Kari Hansen")
	}
	if !strings.Contains(result.Summary, "Signature rights") {
		t.Errorf("Summary should contain 'Signature rights', got: %s", result.Summary)
	}
	if !strings.Contains(result.Summary, "Prokura") {
		t.Errorf("Summary should contain 'Prokura', got: %s", result.Summary)
	}
}

func TestGetSignatureRightsMCP_NoRights(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RolesResponse{
			RoleGroups: []RoleGroup{
				{
					Type: RoleType{Code: "STYR", Description: "Styre"},
					Roles: []Role{
						{
							Type:   RoleType{Code: "LEDE", Description: "Styreleder"},
							Person: &Person{Name: PersonName{FirstName: "Per", LastName: "Olsen"}},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSignatureRightsMCP(context.Background(), GetSignatureRightsArgs{OrgNumber: "923609016"})
	if err != nil {
		t.Fatalf("GetSignatureRightsMCP failed: %v", err)
	}

	if len(result.SignatureRights) != 0 {
		t.Errorf("Expected 0 signature rights, got %d", len(result.SignatureRights))
	}
	if len(result.Prokura) != 0 {
		t.Errorf("Expected 0 prokura, got %d", len(result.Prokura))
	}
	if result.Summary != "No signature rights or prokura found" {
		t.Errorf("Summary = %q, want %q", result.Summary, "No signature rights or prokura found")
	}
}

func TestGetSignatureRightsMCP_WithEntity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RolesResponse{
			RoleGroups: []RoleGroup{
				{
					Type: RoleType{Code: "SIGN", Description: "Signatur"},
					Roles: []Role{
						{
							Type:   RoleType{Code: "SIGN", Description: "Signaturrett"},
							Entity: &RoleEntity{OrganizationNumber: "912345678", Name: []string{"HOLDING AS"}},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.GetSignatureRightsMCP(context.Background(), GetSignatureRightsArgs{OrgNumber: "923609016"})
	if err != nil {
		t.Fatalf("GetSignatureRightsMCP failed: %v", err)
	}

	if len(result.SignatureRights) != 1 {
		t.Fatalf("Expected 1 signature right, got %d", len(result.SignatureRights))
	}
	if result.SignatureRights[0].EntityOrgNr != "912345678" {
		t.Errorf("EntityOrgNr = %q, want %q", result.SignatureRights[0].EntityOrgNr, "912345678")
	}
	if result.SignatureRights[0].Name != "HOLDING AS" {
		t.Errorf("Name = %q, want %q", result.SignatureRights[0].Name, "HOLDING AS")
	}
}

func TestGetSignatureRightsMCP_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.GetSignatureRightsMCP(context.Background(), GetSignatureRightsArgs{OrgNumber: "invalid"})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

func TestBatchGetCompaniesMCP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{
				Companies: []Company{
					{
						OrganizationNumber: "923609016",
						Name:               "EQUINOR ASA",
						OrganizationForm:   &OrganizationForm{Code: "ASA"},
						BusinessAddress:    &Address{AddressLines: []string{"Forusbeen 50"}, PostalCode: "4035", PostalPlace: "STAVANGER"},
						PostalAddress:      &Address{AddressLines: []string{"Postboks 8500"}, PostalCode: "4035", PostalPlace: "STAVANGER"},
					},
					{
						OrganizationNumber: "914778271",
						Name:               "TELENOR ASA",
						Bankrupt:           true,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	result, err := client.BatchGetCompaniesMCP(context.Background(), BatchGetCompaniesArgs{
		OrgNumbers: []string{"923609016", "914778271", "000000000"},
	})
	if err != nil {
		t.Fatalf("BatchGetCompaniesMCP failed: %v", err)
	}

	if len(result.Companies) != 2 {
		t.Fatalf("Expected 2 companies, got %d", len(result.Companies))
	}
	if result.TotalResults != 2 {
		t.Errorf("TotalResults = %d, want 2", result.TotalResults)
	}
	if len(result.NotFound) != 1 {
		t.Errorf("Expected 1 not found, got %d", len(result.NotFound))
	}
	// Check that bankrupt company has correct status
	for _, c := range result.Companies {
		if c.OrganizationNumber == "914778271" && c.Status != "BANKRUPT" {
			t.Errorf("Expected BANKRUPT status for 914778271, got %q", c.Status)
		}
	}
}

func TestBatchGetCompaniesMCP_EmptyInput(t *testing.T) {
	client := NewClient()
	defer client.Close()

	result, err := client.BatchGetCompaniesMCP(context.Background(), BatchGetCompaniesArgs{OrgNumbers: []string{}})
	if err != nil {
		t.Fatalf("BatchGetCompaniesMCP failed: %v", err)
	}
	if len(result.Companies) != 0 {
		t.Errorf("Expected empty result, got %d companies", len(result.Companies))
	}
}

func TestBatchGetCompaniesMCP_InvalidOrgNumber(t *testing.T) {
	client := NewClient()
	defer client.Close()

	_, err := client.BatchGetCompaniesMCP(context.Background(), BatchGetCompaniesArgs{
		OrgNumbers: []string{"923609016", "invalid"},
	})
	if err == nil {
		t.Error("Expected error for invalid org number")
	}
}

// =============================================================================
// doRequest Error Handling Tests
// =============================================================================

func TestDoRequest_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		apiErr := APIError{Message: "Invalid request parameter"}
		_ = json.NewEncoder(w).Encode(apiErr)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompany(context.Background(), "923609016")
	if err == nil {
		t.Fatal("Expected error")
	}
	if !strings.Contains(err.Error(), "Invalid request parameter") {
		t.Errorf("Expected API error message in error, got: %v", err)
	}
}

func TestDoRequest_NonJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Bad Gateway"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompany(context.Background(), "923609016")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestDoRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompany(context.Background(), "923609016")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// =============================================================================
// Additional Edge Cases
// =============================================================================

func TestGetRoles_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"feilmelding": "Ingen enhet funnet"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetRoles(context.Background(), "000000000")
	if err == nil {
		t.Fatal("Expected error for not found")
	}
}

func TestGetSubUnit_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSubUnit(context.Background(), "000000000")
	if err == nil {
		t.Fatal("Expected error for not found")
	}
}

func TestGetUpdates_WithSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("size") != "50" {
			t.Errorf("size = %q, want %q", r.URL.Query().Get("size"), "50")
		}
		resp := UpdatesResponse{
			Embedded: struct {
				Updates []UpdateEntry `json:"oppdaterteEnheter"`
			}{Updates: []UpdateEntry{}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetUpdates(context.Background(), time.Now().Add(-24*time.Hour), &UpdatesOptions{Size: 50})
	if err != nil {
		t.Fatalf("GetUpdates failed: %v", err)
	}
}

func TestSearchSubUnits_WithAllOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "2" {
			t.Errorf("page = %q, want %q", query.Get("page"), "2")
		}
		if query.Get("size") != "25" {
			t.Errorf("size = %q, want %q", query.Get("size"), "25")
		}
		if query.Get("kommunenummer") != "0301" {
			t.Errorf("kommunenummer = %q, want %q", query.Get("kommunenummer"), "0301")
		}

		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{SubUnits: []SubUnit{}},
			Page: PageInfo{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.SearchSubUnits(context.Background(), "Test", &SearchSubUnitsOptions{
		Page:         2,
		Size:         25,
		Municipality: "0301",
	})
	if err != nil {
		t.Fatalf("SearchSubUnits failed: %v", err)
	}
}

func TestGetSubUnitUpdates_WithSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("size") != "100" {
			t.Errorf("size = %q, want %q", r.URL.Query().Get("size"), "100")
		}
		resp := SubUnitUpdatesResponse{
			Embedded: struct {
				Updates []SubUnitUpdateEntry `json:"oppdaterteUnderenheter"`
			}{Updates: []SubUnitUpdateEntry{}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSubUnitUpdates(context.Background(), time.Now().Add(-24*time.Hour), &UpdatesOptions{Size: 100})
	if err != nil {
		t.Fatalf("GetSubUnitUpdates failed: %v", err)
	}
}

func TestClient_RolesCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := RolesResponse{RoleGroups: []RoleGroup{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.GetRoles(context.Background(), "923609016")
	// Second call should hit cache
	_, _ = client.GetRoles(context.Background(), "923609016")

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestClient_SubUnitsCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{SubUnits: []SubUnit{}},
			Page: PageInfo{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.GetSubUnits(context.Background(), "923609016")
	// Second call should hit cache
	_, _ = client.GetSubUnits(context.Background(), "923609016")

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestClient_SubUnitCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		su := SubUnit{OrganizationNumber: "912345678", Name: "TEST"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(su)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.GetSubUnit(context.Background(), "912345678")
	// Second call should hit cache
	_, _ = client.GetSubUnit(context.Background(), "912345678")

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestClient_SearchSubUnitsCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := SubUnitSearchResponse{
			Embedded: struct {
				SubUnits []SubUnit `json:"underenheter"`
			}{SubUnits: []SubUnit{}},
			Page: PageInfo{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.SearchSubUnits(context.Background(), "Oslo", nil)
	// Second call should hit cache
	_, _ = client.SearchSubUnits(context.Background(), "Oslo", nil)

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

// =============================================================================
// Additional Coverage Tests
// =============================================================================

func TestClient_MunicipalitiesCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := MunicipalitiesResponse{
			Embedded: struct {
				Municipalities []Municipality `json:"kommuner"`
			}{Municipalities: []Municipality{{Number: "0301", Name: "Oslo"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.GetMunicipalities(context.Background())
	// Second call should hit cache
	_, _ = client.GetMunicipalities(context.Background())

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestClient_OrgFormsCaching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := OrgFormsResponse{
			Embedded: struct {
				OrgForms []OrgForm `json:"organisasjonsformer"`
			}{OrgForms: []OrgForm{{Code: "AS", Description: "Aksjeselskap"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	// First call
	_, _ = client.GetOrgForms(context.Background())
	// Second call should hit cache
	_, _ = client.GetOrgForms(context.Background())

	if callCount != 1 {
		t.Errorf("Expected 1 API call (cached), got %d", callCount)
	}
}

func TestGetMunicipalities_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetMunicipalities(context.Background())
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetOrgForms_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetOrgForms(context.Background())
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestListMunicipalitiesMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.ListMunicipalitiesMCP(context.Background(), ListMunicipalitiesArgs{})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestListOrgFormsMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.ListOrgFormsMCP(context.Background(), ListOrgFormsArgs{})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetSubUnitMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSubUnitMCP(context.Background(), GetSubUnitArgs{OrgNumber: "923609016"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetUpdatesMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetUpdatesMCP(context.Background(), GetUpdatesArgs{Since: time.Now()})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetSubUnitUpdatesMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSubUnitUpdatesMCP(context.Background(), GetSubUnitUpdatesArgs{Since: time.Now()})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestSearchCompaniesMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.SearchCompaniesMCP(context.Background(), SearchCompaniesArgs{Query: "Test"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestSearchSubUnitsMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.SearchSubUnitsMCP(context.Background(), SearchSubUnitsArgs{Query: "Oslo"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetCompanyMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: "923609016"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetRolesMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetRolesMCP(context.Background(), GetRolesArgs{OrgNumber: "923609016"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetSubUnitsMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSubUnitsMCP(context.Background(), GetSubUnitsArgs{ParentOrgNumber: "923609016"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestGetSignatureRightsMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSignatureRightsMCP(context.Background(), GetSignatureRightsArgs{OrgNumber: "923609016"})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestBatchGetCompaniesMCP_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.BatchGetCompaniesMCP(context.Background(), BatchGetCompaniesArgs{OrgNumbers: []string{"923609016"}})
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestClient_SubUnits_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	_, err := client.GetSubUnits(context.Background(), "923609016")
	if err == nil {
		t.Error("Expected error for not found")
	}
}

func TestSearchCompanies_WithVATAndBankruptFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("registrertIMvaregisteret") != "true" {
			t.Errorf("registrertIMvaregisteret = %q, want %q", query.Get("registrertIMvaregisteret"), "true")
		}
		if query.Get("konkurs") != "false" {
			t.Errorf("konkurs = %q, want %q", query.Get("konkurs"), "false")
		}
		if query.Get("registrertIFrivillighetsregisteret") != "true" {
			t.Errorf("registrertIFrivillighetsregisteret = %q, want %q", query.Get("registrertIFrivillighetsregisteret"), "true")
		}

		resp := SearchResponse{
			Embedded: struct {
				Companies []Company `json:"enheter"`
			}{Companies: []Company{}},
			Page: PageInfo{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	defer client.Close()

	vatTrue := true
	bankruptFalse := false
	voluntaryTrue := true
	_, err := client.SearchCompanies(context.Background(), "Test", &SearchOptions{
		RegisteredInVAT:       &vatTrue,
		Bankrupt:              &bankruptFalse,
		RegisteredInVoluntary: &voluntaryTrue,
	})
	if err != nil {
		t.Fatalf("SearchCompanies failed: %v", err)
	}
}
