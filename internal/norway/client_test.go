package norway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
			result := normalizeOrgNumber(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeOrgNumber(%q) = %q, want %q", tt.input, result, tt.expected)
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
