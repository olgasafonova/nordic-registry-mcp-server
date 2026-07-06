//nolint:misspell // Swedish API uses "Organisation" spelling throughout
package sweden

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// =============================================================================
// GetDocumentList Tests
// =============================================================================

func TestClient_GetDocumentList_NormalizesOrgNumber(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		var reqBody DokumentlistaBegaran
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if reqBody.Identitetsbeteckning != "5560125790" {
			t.Errorf("Identitetsbeteckning = %q, want normalized %q", reqBody.Identitetsbeteckning, "5560125790")
		}

		resp := DokumentlistaSvar{Dokument: []Dokument{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Dashed input must be normalized before hitting the API
	_, err := client.GetDocumentList(context.Background(), "556012-5790")
	if err != nil {
		t.Fatalf("GetDocumentList failed: %v", err)
	}
}

// =============================================================================
// DownloadDocument Request Shape Tests
// =============================================================================

func TestClient_DownloadDocument_RequestShape(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/dokument/doc-123" {
			t.Errorf("Path = %q, want %q", r.URL.Path, "/dokument/doc-123")
		}
		if accept := r.Header.Get("Accept"); accept != "application/zip" {
			t.Errorf("Accept = %q, want %q", accept, "application/zip")
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}

		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("zip-bytes"))
	})

	data, err := client.DownloadDocument(context.Background(), "doc-123")
	if err != nil {
		t.Fatalf("DownloadDocument failed: %v", err)
	}
	if string(data) != "zip-bytes" {
		t.Errorf("Content = %q, want %q", string(data), "zip-bytes")
	}
}

func TestClient_DownloadDocument_EscapesDocumentID(t *testing.T) {
	client := createTestClient(t, nil, func(w http.ResponseWriter, r *http.Request) {
		// PathEscape must keep the traversal-shaped ID inside a single segment
		if r.URL.EscapedPath() != "/dokument/doc%2F..%2Fsecret" {
			t.Errorf("EscapedPath = %q, want %q", r.URL.EscapedPath(), "/dokument/doc%2F..%2Fsecret")
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("ok"))
	})

	_, err := client.DownloadDocument(context.Background(), "doc/../secret")
	if err != nil {
		t.Fatalf("DownloadDocument failed: %v", err)
	}
}

// =============================================================================
// readBoundedDocument Tests
// =============================================================================

// failingReader returns an error after yielding no data.
type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}

func TestClient_ReadBoundedDocument_Success(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	data, err := client.readBoundedDocument(strings.NewReader("small document"))
	if err != nil {
		t.Fatalf("readBoundedDocument failed: %v", err)
	}
	if string(data) != "small document" {
		t.Errorf("data = %q, want %q", string(data), "small document")
	}

	stats := client.CircuitBreakerStats()
	if stats.ConsecutiveFails != 0 {
		t.Errorf("ConsecutiveFails = %d, want 0 after successful read", stats.ConsecutiveFails)
	}
}

func TestClient_ReadBoundedDocument_ReadError(t *testing.T) {
	client, err := NewClient(WithCredentials("test-id", "test-secret"))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, err = client.readBoundedDocument(failingReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader")
	}
	if !strings.Contains(err.Error(), "reading document") {
		t.Errorf("Expected 'reading document' error, got: %v", err)
	}

	stats := client.CircuitBreakerStats()
	if stats.ConsecutiveFails == 0 {
		t.Error("Read error should record a circuit breaker failure")
	}
}
