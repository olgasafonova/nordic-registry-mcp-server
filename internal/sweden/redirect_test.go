package sweden

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestHTTPClient_Refuses307_NoCredentialLeak is the regression test for the
// missing CheckRedirect on the Sweden client. The bug shape:
//
//   - The Sweden client carries OAuth client credentials (token endpoint
//     POST) and Bearer tokens (authenticated data calls).
//   - Without CheckRedirect, Go's default policy follows up to 10
//     redirects and preserves method+body cross-origin for 307/308.
//   - A wiki (or proxy in front of it, or MITM during DNS/TLS bootstrap)
//     returning 307 with Location: https://attacker/ would cause Go to
//     re-POST the credentials (or re-send the Bearer header) to the
//     attacker.
//
// This test asserts the fix: the client refuses redirects, so an
// attacker server set up to receive the redirected request never
// receives anything.
func TestHTTPClient_Refuses307_NoCredentialLeak(t *testing.T) {
	var attackerHits int32
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attackerHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer attacker.Close()

	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", attacker.URL+"/exfil")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer wiki.Close()

	client, err := NewClient(
		WithCredentials("test-client-id", "test-client-secret"),
		WithHTTPClient(&http.Client{
			Timeout: 5 * time.Second,
			// Mirror the production policy explicitly here so the test
			// validates the policy independently of the production
			// constructor.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// Send a raw request through the configured httpClient. Use the wiki
	// URL so the response is the 307; CheckRedirect must short-circuit.
	req, err := http.NewRequest(http.MethodPost, wiki.URL, strings.NewReader("client_id=X&client_secret=SECRET"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := client.httpClient.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d (CheckRedirect didn't short-circuit)", http.StatusTemporaryRedirect, resp.StatusCode)
	}
	if hits := atomic.LoadInt32(&attackerHits); hits != 0 {
		t.Errorf("attacker received %d requests; expected 0 (CheckRedirect must refuse)", hits)
	}
}

// TestHTTPClient_ProductionConstructor_HasCheckRedirect verifies the policy
// is set by the default constructor (not just when WithHTTPClient overrides).
func TestHTTPClient_ProductionConstructor_HasCheckRedirect(t *testing.T) {
	client, err := NewClient(WithCredentials("id", "secret"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.httpClient.CheckRedirect == nil {
		t.Fatal("production constructor missing CheckRedirect (regression — credential leak vector)")
	}
	// The policy must refuse, not no-op.
	got := client.httpClient.CheckRedirect(nil, nil)
	if got != http.ErrUseLastResponse {
		t.Errorf("CheckRedirect returned %v, want http.ErrUseLastResponse", got)
	}
}
