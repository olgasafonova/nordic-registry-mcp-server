package denmark

import (
	"strings"
	"testing"
)

// TestTruncateBody_ShortBodyUnchanged verifies bodies under the cap pass
// through untouched (preserves diagnostic value for normal upstream errors).
func TestTruncateBody_ShortBodyUnchanged(t *testing.T) {
	body := []byte(`{"detail":"invalid_grant"}`)
	got := truncateBody(body)
	if got != string(body) {
		t.Errorf("short body modified: got %q, want %q", got, string(body))
	}
}

// TestTruncateBody_LongBodyCapped verifies bodies over the cap are bounded.
// HG-2 regression guard: an unbounded HTML 4xx page or upstream stack trace
// must not flow verbatim into the MCP caller's error string.
func TestTruncateBody_LongBodyCapped(t *testing.T) {
	body := []byte(strings.Repeat("X", maxBodyInError*4))
	got := truncateBody(body)
	if len(got) != maxBodyInError+len("...") {
		t.Errorf("expected length %d (cap + ellipsis), got %d", maxBodyInError+3, len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected truncated body to end with '...', got %q", got[len(got)-10:])
	}
}

// TestTruncateBody_BoundaryLengths verifies the off-by-one boundary at the cap.
func TestTruncateBody_BoundaryLengths(t *testing.T) {
	// Exactly at the cap: not truncated.
	atCap := []byte(strings.Repeat("Y", maxBodyInError))
	if got := truncateBody(atCap); got != string(atCap) {
		t.Errorf("body at exact cap was truncated: len=%d", len(got))
	}
	// One byte over: truncated.
	overCap := []byte(strings.Repeat("Z", maxBodyInError+1))
	if got := truncateBody(overCap); !strings.HasSuffix(got, "...") {
		t.Error("body at cap+1 was not truncated")
	}
}
