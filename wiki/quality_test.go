package wiki

import (
	"testing"
)

func TestParseWikiTableGlossary(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantTerms []GlossaryTerm
	}{
		{
			name: "parses simple wikitable",
			content: `{| class="wikitable"
|-
! Incorrect !! Correct !! Pattern !! Notes
|-
| Avanade || Avanade Inc. || || Company name
|-
| avanade || Avanade Inc. || || Lowercase
|}`,
			wantCount: 2,
			wantTerms: []GlossaryTerm{
				{Incorrect: "Avanade", Correct: "Avanade Inc.", Pattern: "Company name", Notes: ""},
				{Incorrect: "avanade", Correct: "Avanade Inc.", Pattern: "Lowercase", Notes: ""},
			},
		},
		{
			name: "parses mcp-glossary class table",
			content: `{| class="mcp-glossary wikitable"
|-
| backend || back-end ||  || Hyphenated form preferred
|}`,
			wantCount: 1,
			wantTerms: []GlossaryTerm{
				{Incorrect: "backend", Correct: "back-end", Pattern: "Hyphenated form preferred", Notes: ""},
			},
		},
		{
			name: "handles double-pipe cell separators",
			content: `{| class="wikitable"
|-
| Cell1 || Cell2 || Cell3 || Cell4
|}`,
			wantCount: 1,
		},
		{
			name: "skips header rows",
			content: `{| class="wikitable"
|-
! Header1 !! Header2 !! Header3
|-
| value1 || value2 || value3
|}`,
			wantCount: 1,
		},
		{
			name: "skips empty rows",
			content: `{| class="wikitable"
|-

|-
| value1 || value2
|}`,
			wantCount: 1,
		},
		{
			name: "skips terms where incorrect equals correct",
			content: `{| class="wikitable"
|-
| same || same
|}`,
			wantCount: 0,
		},
		{
			name: "skips terms with empty incorrect",
			content: `{| class="wikitable"
|-
|  || correct value
|}`,
			wantCount: 0,
		},
		{
			name:      "handles empty content",
			content:   "",
			wantCount: 0,
		},
		{
			name:      "handles content without tables",
			content:   "Just some regular text without any wiki tables",
			wantCount: 0,
		},
		{
			name: "handles table without wikitable class",
			content: `{| class="sometable"
|-
| value1 || value2
|}`,
			wantCount: 0,
		},
		{
			name: "extracts pattern column",
			content: `{| class="wikitable"
|-
| typo || correct || \btypo\b ||
|}`,
			wantCount: 1,
			wantTerms: []GlossaryTerm{
				{Incorrect: "typo", Correct: "correct", Pattern: `\btypo\b`, Notes: ""},
			},
		},
		{
			name: "handles multiple tables",
			content: `{| class="wikitable"
|-
| term1 || correct1
|}
Some text between
{| class="wikitable"
|-
| term2 || correct2
|}`,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terms := parseWikiTableGlossary(tt.content)
			if len(terms) != tt.wantCount {
				t.Errorf("parseWikiTableGlossary() returned %d terms, want %d", len(terms), tt.wantCount)
				for i, term := range terms {
					t.Logf("  term[%d]: %+v", i, term)
				}
			}
			if tt.wantTerms != nil {
				for i, want := range tt.wantTerms {
					if i >= len(terms) {
						t.Errorf("missing term[%d]: want %+v", i, want)
						continue
					}
					got := terms[i]
					if got.Incorrect != want.Incorrect {
						t.Errorf("term[%d].Incorrect = %q, want %q", i, got.Incorrect, want.Incorrect)
					}
					if got.Correct != want.Correct {
						t.Errorf("term[%d].Correct = %q, want %q", i, got.Correct, want.Correct)
					}
					if got.Pattern != want.Pattern {
						t.Errorf("term[%d].Pattern = %q, want %q", i, got.Pattern, want.Pattern)
					}
					if got.Notes != want.Notes {
						t.Errorf("term[%d].Notes = %q, want %q", i, got.Notes, want.Notes)
					}
				}
			}
		})
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		name      string
		row       string
		wantCells []string
	}{
		{
			name:      "parses double-pipe separated cells",
			row:       "| cell1 || cell2 || cell3",
			wantCells: []string{"cell1", "cell2", "cell3"},
		},
		{
			name:      "parses cells on separate lines",
			row:       "| cell1\n| cell2\n| cell3",
			wantCells: []string{"cell1", "cell2", "cell3"},
		},
		{
			name:      "handles mixed format",
			row:       "| cell1 || cell2\n| cell3",
			wantCells: []string{"cell1", "cell2", "cell3"},
		},
		{
			name:      "trims whitespace",
			row:       "|   cell1   ||   cell2   ",
			wantCells: []string{"cell1", "cell2"},
		},
		{
			name:      "skips empty cells",
			row:       "| cell1 || || cell3",
			wantCells: []string{"cell1", "cell3"},
		},
		{
			name:      "skips header lines",
			row:       "! Header1 !! Header2",
			wantCells: []string{},
		},
		{
			name:      "handles empty row",
			row:       "",
			wantCells: []string{},
		},
		{
			name:      "handles row with only whitespace",
			row:       "   \n   ",
			wantCells: []string{},
		},
		{
			name:      "handles cells without leading pipe",
			row:       "cell1 || cell2",
			wantCells: []string{"cell1", "cell2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cells := parseTableRow(tt.row)
			if len(cells) != len(tt.wantCells) {
				t.Errorf("parseTableRow() returned %d cells, want %d: got %v", len(cells), len(tt.wantCells), cells)
				return
			}
			for i, want := range tt.wantCells {
				if cells[i] != want {
					t.Errorf("parseTableRow()[%d] = %q, want %q", i, cells[i], want)
				}
			}
		})
	}
}

func TestExtractContext(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		start      int
		end        int
		contextLen int
		want       string
	}{
		{
			name:       "extracts context around match",
			line:       "This is some text with a keyword in the middle of it",
			start:      25,
			end:        32, // "keyword"
			contextLen: 10,
			want:       "...xt with a keyword in the mi...",
		},
		{
			name:       "no prefix ellipsis at start",
			line:       "keyword at the start",
			start:      0,
			end:        7,
			contextLen: 10,
			want:       "keyword at the st...",
		},
		{
			name:       "no suffix ellipsis at end",
			line:       "text ending with keyword",
			start:      17,
			end:        24,
			contextLen: 10,
			want:       "...ding with keyword",
		},
		{
			name:       "short line no ellipsis",
			line:       "short keyword",
			start:      6,
			end:        13,
			contextLen: 20,
			want:       "short keyword",
		},
		{
			name:       "handles zero context length",
			line:       "some keyword here",
			start:      5,
			end:        12,
			contextLen: 0,
			want:       "...keyword...",
		},
		{
			name:       "handles match at exact boundaries",
			line:       "abc",
			start:      0,
			end:        3,
			contextLen: 5,
			want:       "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContext(tt.line, tt.start, tt.end, tt.contextLen)
			if got != tt.want {
				t.Errorf("extractContext() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripCodeBlocksForTerminology(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		contains string
		excludes string
	}{
		{
			name:     "removes syntaxhighlight content",
			content:  "Before <syntaxhighlight>SI.Data.Class</syntaxhighlight> After",
			contains: "Before",
			excludes: "SI.Data",
		},
		{
			name:     "removes source tag content",
			content:  "Text <source lang=\"python\">import SI.Data</source> more",
			contains: "Text",
			excludes: "import",
		},
		{
			name:     "removes pre tag content",
			content:  "Before <pre>namespace.Method()</pre> After",
			contains: "Before",
			excludes: "namespace",
		},
		{
			name:     "removes code tag content",
			content:  "Use <code>API.endpoint</code> here",
			contains: "Use",
			excludes: "API",
		},
		{
			name:     "preserves line count",
			content:  "Line1\n<code>Line2\nLine3</code>\nLine4",
			contains: "Line1",
			excludes: "", // just check structure is preserved
		},
		{
			name:     "handles nested newlines",
			content:  "Start\n<syntaxhighlight>\ncode\nmore code\n</syntaxhighlight>\nEnd",
			contains: "Start",
			excludes: "more code",
		},
		{
			name:     "handles case insensitive tags",
			content:  "Text <CODE>inner</CODE> more",
			contains: "Text",
			excludes: "inner",
		},
		{
			name:     "handles content without code blocks",
			content:  "Just regular content here",
			contains: "Just regular content here",
			excludes: "",
		},
		{
			name:     "handles empty content",
			content:  "",
			contains: "",
			excludes: "",
		},
		{
			name:     "handles multiple code blocks",
			content:  "A <code>x</code> B <pre>y</pre> C",
			contains: "A",
			excludes: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripCodeBlocksForTerminology(tt.content)
			if tt.contains != "" && !containsString(result, tt.contains) {
				t.Errorf("stripCodeBlocksForTerminology() should contain %q, got %q", tt.contains, result)
			}
			if tt.excludes != "" && containsString(result, tt.excludes) {
				t.Errorf("stripCodeBlocksForTerminology() should NOT contain %q, got %q", tt.excludes, result)
			}
		})
	}
}

func TestStripCodeBlocksForTerminology_PreservesLineNumbers(t *testing.T) {
	content := "Line 1\n<code>\nLine 3 code\nLine 4 code\n</code>\nLine 6"
	result := stripCodeBlocksForTerminology(content)

	// Count newlines - should be preserved
	originalNewlines := countNewlines(content)
	resultNewlines := countNewlines(result)

	if originalNewlines != resultNewlines {
		t.Errorf("Newline count changed from %d to %d", originalNewlines, resultNewlines)
	}
}

// Helper functions

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if len(substr) == 0 {
			return true
		}
		if i+len(substr) <= len(s) && s[i:i+len(substr)] == substr {
			return true
		}
	}
	return len(substr) == 0
}

func countNewlines(s string) int {
	count := 0
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	return count
}
