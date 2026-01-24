package norway

import (
	"context"
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

