package finland

import (
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
