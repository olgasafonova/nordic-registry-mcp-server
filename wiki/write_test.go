package wiki

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "string equal to max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "string longer than max",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "very long string",
			input:  "this is a very long string that needs truncation",
			maxLen: 20,
			want:   "this is a very lo...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "unicode string truncation",
			input:  "こんにちは世界",
			maxLen: 15,
			want:   "こんにち...", // len() uses bytes not runes, so 4 chars (12 bytes) + "..." = 15 bytes
		},
		{
			name:   "max length of 3",
			input:  "test",
			maxLen: 3,
			want:   "...", // 0 chars + "..."
		},
		{
			name:   "max length of 4",
			input:  "testing",
			maxLen: 4,
			want:   "t...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestTruncateString_AlwaysProducesValidLength(t *testing.T) {
	testCases := []struct {
		input  string
		maxLen int
	}{
		{"short", 100},
		{"this is a longer string", 10},
		{"exactly ten", 11},
		{"", 5},
		{"test", 3},
	}

	for _, tc := range testCases {
		result := truncateString(tc.input, tc.maxLen)
		if len(result) > tc.maxLen {
			t.Errorf("truncateString(%q, %d) produced result of length %d, exceeds maxLen",
				tc.input, tc.maxLen, len(result))
		}
	}
}
