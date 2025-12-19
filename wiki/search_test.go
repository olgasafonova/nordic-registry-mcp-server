package wiki

import (
	"testing"
)

func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "extracts integer",
			input: "timeout 30",
			want:  "30",
		},
		{
			name:  "extracts decimal",
			input: "rate 1.5",
			want:  "1.5",
		},
		{
			name:  "extracts first number from multiple",
			input: "range 10 to 20",
			want:  "10",
		},
		{
			name:  "handles number with units",
			input: "30s timeout",
			want:  "30",
		},
		{
			name:  "handles no numbers",
			input: "enabled",
			want:  "enabled",
		},
		{
			name:  "trims and lowercases text-only values",
			input: "  TRUE  ",
			want:  "true",
		},
		{
			name:  "extracts from complex string",
			input: "max_connections=100",
			want:  "100",
		},
		{
			name:  "handles percentage",
			input: "completion: 85%",
			want:  "85",
		},
		{
			name:  "handles empty string",
			input: "",
			want:  "",
		},
		{
			name:  "handles only whitespace",
			input: "   ",
			want:  "",
		},
		{
			name:  "extracts from version string",
			input: "version 1.2.3",
			want:  "1.2",
		},
		{
			name:  "handles negative-looking pattern",
			input: "offset -10",
			want:  "10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeValue(tt.input)
			if got != tt.want {
				t.Errorf("normalizeValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Note: TestStripHTMLTags is in validation_test.go
