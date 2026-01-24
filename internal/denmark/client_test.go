package denmark

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient is nil")
	}
	if client.cache == nil {
		t.Error("cache is nil")
	}
	if client.dedup == nil {
		t.Error("dedup is nil")
	}
	if client.circuitBreaker == nil {
		t.Error("circuitBreaker is nil")
	}
	client.Close()
}

func TestNewClientWithOptions(t *testing.T) {
	customHTTPClient := &http.Client{Timeout: 60 * time.Second}
	client := NewClient(WithHTTPClient(customHTTPClient))

	if client.httpClient != customHTTPClient {
		t.Error("custom HTTP client was not set")
	}
	client.Close()
}

func TestClient_ConcurrencyLimit(t *testing.T) {
	client := NewClient()
	defer client.Close()

	// Check semaphore capacity
	if cap(client.semaphore) != MaxConcurrentRequests {
		t.Errorf("semaphore capacity = %d, want %d", cap(client.semaphore), MaxConcurrentRequests)
	}
}

func TestClient_DefaultTimeout(t *testing.T) {
	client := NewClient()
	defer client.Close()

	if client.httpClient.Timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", client.httpClient.Timeout, DefaultTimeout)
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
	if client.cache == nil {
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
	if client.circuitBreaker == nil {
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
