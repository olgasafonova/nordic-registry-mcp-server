//nolint:misspell // Swedish API uses "Organisation" spelling throughout
package sweden

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Client Creation Tests
// =============================================================================

func TestNewClient_MissingCredentials(t *testing.T) {
	// Without credentials, NewClient should fail
	_, err := NewClient()
	if err == nil {
		t.Error("Expected error when credentials are missing")
	}
	if !strings.Contains(err.Error(), "missing OAuth2 credentials") {
		t.Errorf("Expected 'missing OAuth2 credentials' error, got: %v", err)
	}
}

func TestNewClient_WithCredentials(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.clientID != "test-id" {
		t.Errorf("clientID = %q, want %q", client.clientID, "test-id")
	}
	if client.clientSecret != "test-secret" {
		t.Errorf("clientSecret = %q, want %q", client.clientSecret, "test-secret")
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 60 * time.Second}
	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithHTTPClient(customHTTPClient),
		WithBaseURL("http://custom-base.example.com"),
		WithTokenURL("http://custom-token.example.com"),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client.httpClient != customHTTPClient {
		t.Error("custom HTTP client was not set")
	}
	if client.baseURL != "http://custom-base.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "http://custom-base.example.com")
	}
	if client.tokenURL != "http://custom-token.example.com" {
		t.Errorf("tokenURL = %q, want %q", client.tokenURL, "http://custom-token.example.com")
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestNormalizeOrgNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5560125790", "5560125790"},
		{"556012-5790", "5560125790"},
		{"556 012 5790", "5560125790"},
		{"556-012-5790", "5560125790"},
		{"  5560125790  ", "5560125790"},
		{"556012 5790", "5560125790"},
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

func TestValidateOrgNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid 10 digits", "5560125790", false},
		{"valid with dash", "556012-5790", false},
		{"valid 12 digits (personnummer)", "199001011234", false},
		{"too short", "123456789", true},
		{"too long", "1234567890123", true},
		{"letters", "556012579A", true},
		{"empty", "", true},
		{"only spaces", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrgNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrgNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// Type Helper Method Tests
// =============================================================================

func TestOrganisation_GetName(t *testing.T) {
	tests := []struct {
		name     string
		org      Organisation
		expected string
	}{
		{
			name: "with name",
			org: Organisation{
				Organisationsnamn: &Organisationsnamn{
					OrganisationsnamnLista: []OrganisationsnamnObjekt{
						{Namn: "TEST AB"},
					},
				},
			},
			expected: "TEST AB",
		},
		{
			name: "multiple names returns first",
			org: Organisation{
				Organisationsnamn: &Organisationsnamn{
					OrganisationsnamnLista: []OrganisationsnamnObjekt{
						{Namn: "MAIN NAME AB"},
						{Namn: "SECONDARY NAME AB"},
					},
				},
			},
			expected: "MAIN NAME AB",
		},
		{
			name:     "nil organisationsnamn",
			org:      Organisation{},
			expected: "",
		},
		{
			name: "empty list",
			org: Organisation{
				Organisationsnamn: &Organisationsnamn{
					OrganisationsnamnLista: []OrganisationsnamnObjekt{},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.org.GetName()
			if result != tt.expected {
				t.Errorf("GetName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOrganisation_GetOrgNumber(t *testing.T) {
	tests := []struct {
		name     string
		org      Organisation
		expected string
	}{
		{
			name: "with org number",
			org: Organisation{
				Organisationsidentitet: &Identitetsbeteckning{
					Identitetsbeteckning: "5560125790",
				},
			},
			expected: "5560125790",
		},
		{
			name:     "nil identity",
			org:      Organisation{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.org.GetOrgNumber()
			if result != tt.expected {
				t.Errorf("GetOrgNumber() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOrganisation_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		org      Organisation
		expected bool
	}{
		{
			name: "active - VerksamOrganisation JA",
			org: Organisation{
				VerksamOrganisation: &VerksamOrganisation{Kod: JaNejJA},
			},
			expected: true,
		},
		{
			name: "VerksamOrganisation NEJ defaults to active",
			org: Organisation{
				VerksamOrganisation: &VerksamOrganisation{Kod: JaNejNEJ},
			},
			// Code defaults to active unless explicitly deregistered
			expected: true,
		},
		{
			name: "inactive - deregistered",
			org: Organisation{
				AvregistreradOrganisation: &AvregistreradOrganisation{
					Avregistreringsdatum: "2024-01-15",
				},
			},
			expected: false,
		},
		{
			name:     "default active when no status",
			org:      Organisation{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.org.IsActive()
			if result != tt.expected {
				t.Errorf("IsActive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOrganisation_GetFormCode(t *testing.T) {
	tests := []struct {
		name     string
		org      Organisation
		expected string
	}{
		{
			name: "with form code",
			org: Organisation{
				Organisationsform: &Organisationsform{Kod: "AB", Klartext: "Aktiebolag"},
			},
			expected: "AB",
		},
		{
			name:     "nil form",
			org:      Organisation{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.org.GetFormCode()
			if result != tt.expected {
				t.Errorf("GetFormCode() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOrganisation_GetFormDescription(t *testing.T) {
	org := Organisation{
		Organisationsform: &Organisationsform{Kod: "AB", Klartext: "Aktiebolag"},
	}
	if result := org.GetFormDescription(); result != "Aktiebolag" {
		t.Errorf("GetFormDescription() = %q, want %q", result, "Aktiebolag")
	}
}

func TestOrganisation_GetRegistrationDate(t *testing.T) {
	org := Organisation{
		Organisationsdatum: &Organisationsdatum{Registreringsdatum: "1990-05-15"},
	}
	if result := org.GetRegistrationDate(); result != "1990-05-15" {
		t.Errorf("GetRegistrationDate() = %q, want %q", result, "1990-05-15")
	}
}

func TestOrganisation_GetAddress(t *testing.T) {
	tests := []struct {
		name     string
		org      Organisation
		expected string
	}{
		{
			name: "full address",
			org: Organisation{
				PostadressOrganisation: &PostadressOrganisation{
					Postadress: &Postadress{
						Utdelningsadress: "Testgatan 1",
						Postnummer:       "12345",
						Postort:          "STOCKHOLM",
					},
				},
			},
			expected: "Testgatan 1, 12345 STOCKHOLM",
		},
		{
			name: "with c/o address",
			org: Organisation{
				PostadressOrganisation: &PostadressOrganisation{
					Postadress: &Postadress{
						CoAdress:         "c/o Someone",
						Utdelningsadress: "Testgatan 1",
						Postnummer:       "12345",
						Postort:          "STOCKHOLM",
					},
				},
			},
			expected: "c/o Someone, Testgatan 1, 12345 STOCKHOLM",
		},
		{
			name: "with foreign country",
			org: Organisation{
				PostadressOrganisation: &PostadressOrganisation{
					Postadress: &Postadress{
						Utdelningsadress: "123 Main St",
						Postnummer:       "10001",
						Postort:          "NEW YORK",
						Land:             "USA",
					},
				},
			},
			expected: "123 Main St, 10001 NEW YORK, USA",
		},
		{
			name: "Sweden excluded from address",
			org: Organisation{
				PostadressOrganisation: &PostadressOrganisation{
					Postadress: &Postadress{
						Utdelningsadress: "Testgatan 1",
						Postnummer:       "12345",
						Postort:          "STOCKHOLM",
						Land:             "Sverige",
					},
				},
			},
			expected: "Testgatan 1, 12345 STOCKHOLM",
		},
		{
			name:     "nil address",
			org:      Organisation{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.org.GetAddress()
			if result != tt.expected {
				t.Errorf("GetAddress() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestOrganisation_GetBusinessDescription(t *testing.T) {
	org := Organisation{
		Verksamhetsbeskrivning: &Verksamhetsbeskrivning{Beskrivning: "Software development"},
	}
	if result := org.GetBusinessDescription(); result != "Software development" {
		t.Errorf("GetBusinessDescription() = %q, want %q", result, "Software development")
	}
}

func TestOrganisation_GetSNICodes(t *testing.T) {
	org := Organisation{
		NaringsgrenOrganisation: &NaringsgrenOrganisation{
			SNI: []KodKlartext{
				{Kod: "62010", Klartext: "Dataprogrammering"},
				{Kod: "62020", Klartext: "Datakonsultverksamhet"},
			},
		},
	}
	codes := org.GetSNICodes()
	if len(codes) != 2 {
		t.Fatalf("GetSNICodes() returned %d codes, want 2", len(codes))
	}
	if codes[0].Kod != "62010" {
		t.Errorf("codes[0].Kod = %q, want %q", codes[0].Kod, "62010")
	}
}

// =============================================================================
// OAuth2 Token Tests
// =============================================================================

func TestClient_TokenRefresh(t *testing.T) {
	tokenCalls := 0
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++

		// Verify Basic auth
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Expected Basic auth, got: %s", auth)
		}

		// Decode and verify credentials
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			t.Errorf("Failed to decode auth: %v", err)
		}
		if string(decoded) != "test-id:test-secret" {
			t.Errorf("Credentials = %q, want %q", string(decoded), "test-id:test-secret")
		}

		// Return token response
		resp := TokenResponse{
			AccessToken: "test-access-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithTokenURL(tokenServer.URL),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Get token twice - should only call token endpoint once due to caching
	token1, err := client.getToken(context.Background())
	if err != nil {
		t.Fatalf("First getToken failed: %v", err)
	}
	if token1 != "test-access-token" {
		t.Errorf("token = %q, want %q", token1, "test-access-token")
	}

	token2, err := client.getToken(context.Background())
	if err != nil {
		t.Fatalf("Second getToken failed: %v", err)
	}
	if token2 != token1 {
		t.Errorf("Second token should be cached, got different value")
	}

	if tokenCalls != 1 {
		t.Errorf("Token endpoint called %d times, want 1", tokenCalls)
	}
}

func TestClient_TokenError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid_client"}`))
	}))
	defer tokenServer.Close()

	client, err := NewClient(
		WithCredentials("bad-id", "bad-secret"),
		WithTokenURL(tokenServer.URL),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.getToken(context.Background())
	if err == nil {
		t.Error("Expected error for invalid credentials")
	}
}

// =============================================================================
// API Call Tests with Mock Server
// =============================================================================

func createTestClient(t *testing.T, tokenHandler, apiHandler http.HandlerFunc) *Client {
	t.Helper()

	// Token server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tokenHandler != nil {
			tokenHandler(w, r)
			return
		}
		resp := TokenResponse{AccessToken: "test-token", ExpiresIn: 3600}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(tokenServer.Close)

	// API server
	apiServer := httptest.NewServer(apiHandler)
	t.Cleanup(apiServer.Close)

	client, err := NewClient(
		WithCredentials("test-id", "test-secret"),
		WithTokenURL(tokenServer.URL),
		WithBaseURL(apiServer.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}

func TestClient_GetCompany_Success(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/organisationer" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		// Verify Bearer token
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Expected 'Bearer test-token', got %q", auth)
		}

		resp := OrganisationerSvar{
			Organisationer: []Organisation{
				{
					Organisationsidentitet: &Identitetsbeteckning{
						Identitetsbeteckning: "5560125790",
					},
					Organisationsnamn: &Organisationsnamn{
						OrganisationsnamnLista: []OrganisationsnamnObjekt{
							{Namn: "VOLVO AB"},
						},
					},
					VerksamOrganisation: &VerksamOrganisation{Kod: JaNejJA},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	result, err := client.GetCompany(context.Background(), "5560125790")
	if err != nil {
		t.Fatalf("GetCompany failed: %v", err)
	}

	if len(result.Organisationer) != 1 {
		t.Fatalf("Expected 1 organisation, got %d", len(result.Organisationer))
	}
	if result.Organisationer[0].GetName() != "VOLVO AB" {
		t.Errorf("Name = %q, want %q", result.Organisationer[0].GetName(), "VOLVO AB")
	}
}

func TestClient_GetCompany_NotFound(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		resp := APIError{
			Status: 404,
			Title:  "Not Found",
			Detail: "Organisation finns ej",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	_, err := client.GetCompany(context.Background(), "0000000000")
	if err == nil {
		t.Error("Expected error for not found")
	}
}

func TestClient_GetCompany_EmptyOrgNumber(t *testing.T) {
	client, _ := NewClient(WithCredentials("test-id", "test-secret"))

	_, err := client.GetCompany(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty org number")
	}
}

func TestClient_GetCompany_Caching(t *testing.T) {
	apiCalls := 0
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		resp := OrganisationerSvar{
			Organisationer: []Organisation{
				{
					Organisationsidentitet: &Identitetsbeteckning{
						Identitetsbeteckning: "5560125790",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// First call
	_, err := client.GetCompany(context.Background(), "5560125790")
	if err != nil {
		t.Fatalf("First call failed: %v", err)
	}

	// Second call should be cached
	_, err = client.GetCompany(context.Background(), "5560125790")
	if err != nil {
		t.Fatalf("Second call failed: %v", err)
	}

	if apiCalls != 1 {
		t.Errorf("API called %d times, want 1 (cached)", apiCalls)
	}
}

func TestClient_GetDocumentList_Success(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dokumentlista" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := DokumentlistaSvar{
			Dokument: []Dokument{
				{
					DokumentID:             "doc-123",
					Filformat:              "iXBRL",
					RapporteringsperiodTom: "2023-12-31",
					Registreringstidpunkt:  "2024-03-15T10:00:00Z",
				},
				{
					DokumentID:             "doc-456",
					Filformat:              "XBRL",
					RapporteringsperiodTom: "2022-12-31",
					Registreringstidpunkt:  "2023-03-20T09:30:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	result, err := client.GetDocumentList(context.Background(), "5560125790")
	if err != nil {
		t.Fatalf("GetDocumentList failed: %v", err)
	}

	if len(result.Dokument) != 2 {
		t.Fatalf("Expected 2 documents, got %d", len(result.Dokument))
	}
	if result.Dokument[0].DokumentID != "doc-123" {
		t.Errorf("DocumentID = %q, want %q", result.Dokument[0].DokumentID, "doc-123")
	}
}

func TestClient_DownloadDocument_Success(t *testing.T) {
	expectedContent := []byte("fake zip content")
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/dokument/") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(expectedContent)
	})

	data, err := client.DownloadDocument(context.Background(), "doc-123")
	if err != nil {
		t.Fatalf("DownloadDocument failed: %v", err)
	}

	if string(data) != string(expectedContent) {
		t.Errorf("Content mismatch")
	}
}

func TestClient_DownloadDocument_EmptyID(t *testing.T) {
	client, _ := NewClient(WithCredentials("test-id", "test-secret"))

	_, err := client.DownloadDocument(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty document ID")
	}
}

func TestClient_IsAlive_Success(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/isalive" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("OK"))
	})

	alive, err := client.IsAlive(context.Background())
	if err != nil {
		t.Fatalf("IsAlive failed: %v", err)
	}
	if !alive {
		t.Error("Expected alive=true")
	}
}

func TestClient_IsAlive_NotOK(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("NOT OK"))
	})

	alive, err := client.IsAlive(context.Background())
	if err != nil {
		t.Fatalf("IsAlive failed: %v", err)
	}
	if alive {
		t.Error("Expected alive=false for non-OK response")
	}
}

// =============================================================================
// Circuit Breaker Tests
// =============================================================================

func TestClient_CircuitBreaker(t *testing.T) {
	failures := 0
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		failures++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "server error"}`))
	})

	// Make requests until circuit opens
	for range 10 {
		_, _ = client.GetCompany(context.Background(), "5560125790")
	}

	// Circuit should be open now
	status := client.CircuitBreakerStatus()
	if status != "open" {
		t.Errorf("CircuitBreakerStatus = %q, want %q", status, "open")
	}
}

func TestClient_CircuitBreakerStats(t *testing.T) {
	client, _ := NewClient(WithCredentials("test-id", "test-secret"))

	stats := client.CircuitBreakerStats()
	if stats.State != "closed" {
		t.Errorf("Initial state = %q, want %q", stats.State, "closed")
	}
	if stats.ConsecutiveFails != 0 {
		t.Errorf("Initial ConsecutiveFails = %d, want 0", stats.ConsecutiveFails)
	}
}

func TestClient_CacheSize(t *testing.T) {
	client, _ := NewClient(WithCredentials("test-id", "test-secret"))

	size := client.CacheSize()
	if size != 0 {
		t.Errorf("Initial cache size = %d, want 0", size)
	}
}

// =============================================================================
// MCP Wrapper Tests
// =============================================================================

func TestGetCompanyMCP_Success(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		resp := OrganisationerSvar{
			Organisationer: []Organisation{
				{
					Organisationsidentitet: &Identitetsbeteckning{
						Identitetsbeteckning: "5560125790",
					},
					Organisationsnamn: &Organisationsnamn{
						OrganisationsnamnLista: []OrganisationsnamnObjekt{
							{Namn: "TEST AB"},
						},
					},
					Organisationsform: &Organisationsform{
						Kod:      "AB",
						Klartext: "Aktiebolag",
					},
					JuridiskForm: &JuridiskForm{
						Kod:      "49",
						Klartext: "Övriga aktiebolag",
					},
					VerksamOrganisation: &VerksamOrganisation{Kod: JaNejJA},
					Organisationsdatum: &Organisationsdatum{
						Registreringsdatum: "1990-01-15",
					},
					PostadressOrganisation: &PostadressOrganisation{
						Postadress: &Postadress{
							Utdelningsadress: "Testgatan 1",
							Postnummer:       "12345",
							Postort:          "STOCKHOLM",
						},
					},
					NaringsgrenOrganisation: &NaringsgrenOrganisation{
						SNI: []KodKlartext{{Kod: "62010", Klartext: "Dataprogrammering"}},
					},
					Reklamsparr: &Reklamsparr{Kod: JaNejJA},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	result, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: "5560125790"})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Company == nil {
		t.Fatal("Company is nil")
	}
	if result.Company.OrganizationNumber != "5560125790" {
		t.Errorf("OrganizationNumber = %q, want %q", result.Company.OrganizationNumber, "5560125790")
	}
	if result.Company.Name != "TEST AB" {
		t.Errorf("Name = %q, want %q", result.Company.Name, "TEST AB")
	}
	if result.Company.OrganizationForm != "AB - Aktiebolag" {
		t.Errorf("OrganizationForm = %q, want %q", result.Company.OrganizationForm, "AB - Aktiebolag")
	}
	if result.Company.LegalForm != "49 - Övriga aktiebolag" {
		t.Errorf("LegalForm = %q, want %q", result.Company.LegalForm, "49 - Övriga aktiebolag")
	}
	if !result.Company.IsActive {
		t.Error("Expected IsActive=true")
	}
	if result.Company.RegistrationDate != "1990-01-15" {
		t.Errorf("RegistrationDate = %q, want %q", result.Company.RegistrationDate, "1990-01-15")
	}
	if !result.Company.AdBlockEnabled {
		t.Error("Expected AdBlockEnabled=true")
	}
	if len(result.Company.IndustryCodes) != 1 {
		t.Errorf("Expected 1 industry code, got %d", len(result.Company.IndustryCodes))
	}
}

func TestGetCompanyMCP_ValidationError(t *testing.T) {
	client, _ := NewClient(WithCredentials("test-id", "test-secret"))

	_, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: ""})
	if err == nil {
		t.Error("Expected validation error for empty org number")
	}
}

func TestGetCompanyMCP_NoCompanyFound(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		resp := OrganisationerSvar{Organisationer: []Organisation{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	_, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: "5560125790"})
	if err == nil {
		t.Error("Expected error when no company found")
	}
}

func TestGetCompanyMCP_Deregistered(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		resp := OrganisationerSvar{
			Organisationer: []Organisation{
				{
					Organisationsidentitet: &Identitetsbeteckning{
						Identitetsbeteckning: "5560125790",
					},
					AvregistreradOrganisation: &AvregistreradOrganisation{
						Avregistreringsdatum: "2024-01-15",
					},
					Avregistreringsorsak: &Avregistreringsorsak{
						Klartext: "Avregistrerad på egen begäran",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	result, err := client.GetCompanyMCP(context.Background(), GetCompanyArgs{OrgNumber: "5560125790"})
	if err != nil {
		t.Fatalf("GetCompanyMCP failed: %v", err)
	}

	if result.Company.IsActive {
		t.Error("Expected IsActive=false for deregistered company")
	}
	if result.Company.DeregisteredDate != "2024-01-15" {
		t.Errorf("DeregisteredDate = %q, want %q", result.Company.DeregisteredDate, "2024-01-15")
	}
	if result.Company.DeregisteredReason != "Avregistrerad på egen begäran" {
		t.Errorf("DeregisteredReason = %q, want %q", result.Company.DeregisteredReason, "Avregistrerad på egen begäran")
	}
}

func TestGetDocumentListMCP_Success(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		resp := DokumentlistaSvar{
			Dokument: []Dokument{
				{
					DokumentID:             "doc-123",
					Filformat:              "iXBRL",
					RapporteringsperiodTom: "2023-12-31",
					Registreringstidpunkt:  "2024-03-15T10:00:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	result, err := client.GetDocumentListMCP(context.Background(), GetDocumentListArgs{OrgNumber: "5560125790"})
	if err != nil {
		t.Fatalf("GetDocumentListMCP failed: %v", err)
	}

	if result.OrganizationNumber != "5560125790" {
		t.Errorf("OrganizationNumber = %q, want %q", result.OrganizationNumber, "5560125790")
	}
	if result.Count != 1 {
		t.Errorf("Count = %d, want 1", result.Count)
	}
	if len(result.Documents) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(result.Documents))
	}
	if result.Documents[0].DocumentID != "doc-123" {
		t.Errorf("DocumentID = %q, want %q", result.Documents[0].DocumentID, "doc-123")
	}
}

func TestCheckStatusMCP_Available(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	result, err := client.CheckStatusMCP(context.Background(), CheckStatusArgs{})
	if err != nil {
		t.Fatalf("CheckStatusMCP failed: %v", err)
	}

	if !result.Available {
		t.Error("Expected Available=true")
	}
	if result.CircuitBreakerStatus != "closed" {
		t.Errorf("CircuitBreakerStatus = %q, want %q", result.CircuitBreakerStatus, "closed")
	}
}

func TestCheckStatusMCP_Unavailable(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	result, err := client.CheckStatusMCP(context.Background(), CheckStatusArgs{})
	if err != nil {
		t.Fatalf("CheckStatusMCP should not return error: %v", err)
	}

	if result.Available {
		t.Error("Expected Available=false")
	}
}

func TestDownloadDocumentMCP_Success(t *testing.T) {
	zipContent := []byte("PK\x03\x04fake zip content")
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipContent)
	})

	result, err := client.DownloadDocumentMCP(context.Background(), DownloadDocumentArgs{DocumentID: "doc-123"})
	if err != nil {
		t.Fatalf("DownloadDocumentMCP failed: %v", err)
	}

	if result.DocumentID != "doc-123" {
		t.Errorf("DocumentID = %q, want %q", result.DocumentID, "doc-123")
	}
	if result.FileFormat != "application/zip" {
		t.Errorf("FileFormat = %q, want %q", result.FileFormat, "application/zip")
	}
	if result.SizeBytes != len(zipContent) {
		t.Errorf("SizeBytes = %d, want %d", result.SizeBytes, len(zipContent))
	}

	// Verify base64 encoding
	decoded, err := base64.StdEncoding.DecodeString(result.ContentB64)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	if string(decoded) != string(zipContent) {
		t.Error("Decoded content mismatch")
	}
}

func TestDownloadDocumentMCP_EmptyID(t *testing.T) {
	client, _ := NewClient(WithCredentials("test-id", "test-secret"))

	_, err := client.DownloadDocumentMCP(context.Background(), DownloadDocumentArgs{DocumentID: ""})
	if err == nil {
		t.Error("Expected error for empty document ID")
	}
}

// =============================================================================
// Context Cancellation Tests
// =============================================================================

func TestClient_ContextCancellation(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("OK"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.GetCompany(ctx, "5560125790")
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// =============================================================================
// JSON Parsing Tests
// =============================================================================

func TestParseOrganisationerSvar(t *testing.T) {
	jsonData := `{
		"organisationer": [
			{
				"organisationsidentitet": {
					"identitetsbeteckning": "5560125790",
					"typ": {"kod": "ORGNO", "klartext": "Organisationsnummer"}
				},
				"organisationsnamn": {
					"organisationsnamnLista": [
						{"namn": "VOLVO AB", "organisationsnamntyp": {"kod": "NAMN", "klartext": "Organisationsnamn"}}
					],
					"dataproducent": "Bolagsverket"
				},
				"organisationsform": {
					"kod": "AB",
					"klartext": "Aktiebolag",
					"dataproducent": "Bolagsverket"
				},
				"verksamOrganisation": {
					"kod": "JA",
					"dataproducent": "SCB"
				},
				"organisationsdatum": {
					"registreringsdatum": "1927-04-14",
					"dataproducent": "Bolagsverket"
				}
			}
		]
	}`

	var resp OrganisationerSvar
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(resp.Organisationer) != 1 {
		t.Fatalf("Expected 1 organisation, got %d", len(resp.Organisationer))
	}

	org := resp.Organisationer[0]
	if org.GetOrgNumber() != "5560125790" {
		t.Errorf("OrgNumber = %q, want %q", org.GetOrgNumber(), "5560125790")
	}
	if org.GetName() != "VOLVO AB" {
		t.Errorf("Name = %q, want %q", org.GetName(), "VOLVO AB")
	}
	if org.GetFormCode() != "AB" {
		t.Errorf("FormCode = %q, want %q", org.GetFormCode(), "AB")
	}
	if !org.IsActive() {
		t.Error("Expected IsActive=true")
	}
}

func TestParseDokumentlistaSvar(t *testing.T) {
	jsonData := `{
		"dokument": [
			{
				"dokumentId": "ABC123",
				"filformat": "iXBRL",
				"rapporteringsperiodTom": "2023-12-31",
				"registreringstidpunkt": "2024-03-15T10:30:00Z"
			}
		]
	}`

	var resp DokumentlistaSvar
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(resp.Dokument) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(resp.Dokument))
	}
	if resp.Dokument[0].DokumentID != "ABC123" {
		t.Errorf("DokumentID = %q, want %q", resp.Dokument[0].DokumentID, "ABC123")
	}
}

func TestParseAPIError(t *testing.T) {
	jsonData := `{
		"type": "about:blank",
		"instance": "/organisationer",
		"status": 404,
		"timestamp": "2024-01-15T10:00:00Z",
		"requestId": "req-123",
		"title": "Not Found",
		"detail": "Organisation med identitetsbeteckning 0000000000 finns ej"
	}`

	var apiErr APIError
	if err := json.Unmarshal([]byte(jsonData), &apiErr); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if apiErr.Status != 404 {
		t.Errorf("Status = %d, want 404", apiErr.Status)
	}
	if apiErr.Title != "Not Found" {
		t.Errorf("Title = %q, want %q", apiErr.Title, "Not Found")
	}
}

func TestParseTokenResponse(t *testing.T) {
	jsonData := `{
		"access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
		"token_type": "Bearer",
		"expires_in": 3600,
		"scope": "vardefulla-datamangder:read vardefulla-datamangder:ping"
	}`

	var resp TokenResponse
	if err := json.Unmarshal([]byte(jsonData), &resp); err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", resp.TokenType, "Bearer")
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want 3600", resp.ExpiresIn)
	}
}
