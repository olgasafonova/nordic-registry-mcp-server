package wiki

import (
	"testing"
)

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected float64
	}{
		{
			name:     "identical strings",
			s1:       "hello world",
			s2:       "hello world",
			expected: 1.0,
		},
		{
			name:     "completely different",
			s1:       "hello world",
			s2:       "foo bar",
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			s1:       "hello world test",
			s2:       "hello world foo",
			expected: 0.5, // 2 common out of 4 unique words
		},
		{
			name:     "empty first string",
			s1:       "",
			s2:       "hello world",
			expected: 0.0,
		},
		{
			name:     "empty second string",
			s1:       "hello world",
			s2:       "",
			expected: 0.0,
		},
		{
			name:     "both empty",
			s1:       "",
			s2:       "",
			expected: 1.0, // identical
		},
		{
			name:     "one common word",
			s1:       "hello world",
			s2:       "hello there",
			expected: 1.0 / 3.0, // 1 common, 3 unique (hello, world, there)
		},
		{
			name:     "whitespace only first",
			s1:       "   ",
			s2:       "hello",
			expected: 0.0,
		},
		{
			name:     "whitespace only second",
			s1:       "hello",
			s2:       "   ",
			expected: 0.0,
		},
		{
			name:     "different word order",
			s1:       "world hello",
			s2:       "hello world",
			expected: 1.0, // same words
		},
		{
			name:     "duplicate words",
			s1:       "hello hello world",
			s2:       "world world hello",
			expected: 1.0, // sets are {hello, world} in both
		},
		{
			name:     "single word match",
			s1:       "test",
			s2:       "test",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateSimilarity(tt.s1, tt.s2)
			// Allow small floating point tolerance
			diff := got - tt.expected
			if diff > 0.01 || diff < -0.01 {
				t.Errorf("calculateSimilarity(%q, %q) = %f, want %f", tt.s1, tt.s2, got, tt.expected)
			}
		})
	}
}

func TestCalculateSimilarity_Symmetry(t *testing.T) {
	// Test that similarity is symmetric
	testCases := [][2]string{
		{"hello world", "world hello"},
		{"foo bar baz", "bar foo qux"},
		{"a b c d", "c d e f"},
	}

	for _, tc := range testCases {
		forward := calculateSimilarity(tc[0], tc[1])
		backward := calculateSimilarity(tc[1], tc[0])
		if forward != backward {
			t.Errorf("Similarity not symmetric: (%q, %q) = %f but (%q, %q) = %f",
				tc[0], tc[1], forward, tc[1], tc[0], backward)
		}
	}
}

func TestCalculateSimilarity_Range(t *testing.T) {
	// Test that similarity is always between 0 and 1
	testCases := [][2]string{
		{"hello world", "foo bar"},
		{"a b c d e f g h", "x y z"},
		{"single", "multiple words here"},
		{"", "not empty"},
	}

	for _, tc := range testCases {
		sim := calculateSimilarity(tc[0], tc[1])
		if sim < 0 || sim > 1 {
			t.Errorf("calculateSimilarity(%q, %q) = %f, should be between 0 and 1",
				tc[0], tc[1], sim)
		}
	}
}
