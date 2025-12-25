package wiki

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
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

func TestGetSections_Success(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "parse" {
			response := map[string]interface{}{
				"parse": map[string]interface{}{
					"title":  "Test Page",
					"pageid": float64(1),
					"sections": []interface{}{
						map[string]interface{}{
							"toclevel":   float64(1),
							"level":      "2",
							"line":       "Introduction",
							"number":     "1",
							"index":      "1",
							"fromtitle":  "Test_Page",
							"byteoffset": float64(0),
							"anchor":     "Introduction",
						},
						map[string]interface{}{
							"toclevel":   float64(1),
							"level":      "2",
							"line":       "Details",
							"number":     "2",
							"index":      "2",
							"fromtitle":  "Test_Page",
							"byteoffset": float64(100),
							"anchor":     "Details",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetSections(ctx, GetSectionsArgs{Title: "Test Page"})

	if err != nil {
		t.Fatalf("GetSections failed: %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Page")
	}
	if len(result.Sections) != 2 {
		t.Errorf("Sections count = %d, want 2", len(result.Sections))
	}
}

func TestGetSections_WithSection(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "parse" {
			section := r.FormValue("section")
			if section != "" {
				response := map[string]interface{}{
					"parse": map[string]interface{}{
						"title":  "Test Page",
						"pageid": float64(1),
						"wikitext": map[string]interface{}{
							"*": "== Introduction ==\nSection content here",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"parse":{"sections":[]}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetSections(ctx, GetSectionsArgs{Title: "Test Page", Section: 1})

	if err != nil {
		t.Fatalf("GetSections with section failed: %v", err)
	}
	if result.SectionContent == "" {
		t.Error("Expected section content, got empty")
	}
}

func TestParse_Success(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "parse" {
			response := map[string]interface{}{
				"parse": map[string]interface{}{
					"title":  "Test",
					"pageid": float64(0),
					"text": map[string]interface{}{
						"*": "<p><b>Hello</b> world</p>",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.Parse(ctx, ParseArgs{
		Wikitext: "'''Hello''' world",
		Title:    "Test",
	})

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.HTML == "" {
		t.Error("Expected HTML output, got empty")
	}
}

func TestParse_EmptyWikitext(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()
	_, err := client.Parse(ctx, ParseArgs{
		Wikitext: "",
	})

	if err == nil {
		t.Error("Expected error for empty wikitext")
	}
}

func TestParse_WithCategoriesAndLinks(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "parse" {
			response := map[string]interface{}{
				"parse": map[string]interface{}{
					"title":  "Test",
					"pageid": float64(0),
					"text": map[string]interface{}{
						"*": "<p>Content with [[Link]] and [[Category:Test]]</p>",
					},
					"categories": []interface{}{
						map[string]interface{}{"*": "Test Category"},
						map[string]interface{}{"*": "Another Category"},
					},
					"links": []interface{}{
						map[string]interface{}{"*": "Link One", "ns": float64(0)},
						map[string]interface{}{"*": "Link Two", "ns": float64(0)},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.Parse(ctx, ParseArgs{
		Wikitext: "Content with [[Link]] and [[Category:Test]]",
	})

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(result.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(result.Categories))
	}
	if len(result.Links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(result.Links))
	}
}

func TestResolveTitle_Fuzzy(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		list := r.FormValue("list")
		if action == "query" && list == "search" {
			response := map[string]interface{}{
				"query": map[string]interface{}{
					"searchinfo": map[string]interface{}{"totalhits": float64(3)},
					"search": []interface{}{
						map[string]interface{}{
							"pageid":  float64(1),
							"title":   "Test Page",
							"snippet": "A test page",
						},
						map[string]interface{}{
							"pageid":  float64(2),
							"title":   "Testing Guide",
							"snippet": "Guide to testing",
						},
						map[string]interface{}{
							"pageid":  float64(3),
							"title":   "Tests Overview",
							"snippet": "Overview of tests",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ResolveTitle(ctx, ResolveTitleArgs{
		Title:      "test",
		Fuzzy:      true,
		MaxResults: 5,
	})

	if err != nil {
		t.Fatalf("ResolveTitle failed: %v", err)
	}
	if len(result.Suggestions) != 3 {
		t.Errorf("Expected 3 suggestions, got %d", len(result.Suggestions))
	}
}

func TestResolveTitle_EmptyTitle(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()
	_, err := client.ResolveTitle(ctx, ResolveTitleArgs{
		Title: "",
	})

	if err == nil {
		t.Error("Expected error for empty title")
	}
}

func TestResolveTitle_Success(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		list := r.FormValue("list")
		if action == "query" && list == "search" {
			response := map[string]interface{}{
				"query": map[string]interface{}{
					"searchinfo": map[string]interface{}{"totalhits": float64(2)},
					"search": []interface{}{
						map[string]interface{}{
							"pageid":  float64(1),
							"title":   "API Guide",
							"snippet": "The API guide for...",
						},
						map[string]interface{}{
							"pageid":  float64(2),
							"title":   "API Reference",
							"snippet": "API reference docs...",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ResolveTitle(ctx, ResolveTitleArgs{Title: "API"})

	if err != nil {
		t.Fatalf("ResolveTitle failed: %v", err)
	}
	if len(result.Suggestions) == 0 {
		t.Error("Expected suggestions, got none")
	}
}

func TestResolveTitle_NoMatches(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "query" {
			prop := r.FormValue("prop")
			list := r.FormValue("list")
			// GetPageInfo query
			if prop != "" && strings.Contains(prop, "info") {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"-1": map[string]interface{}{
								"ns":      float64(0),
								"title":   "Xyznonexistent",
								"missing": "",
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			// Search query
			if list == "search" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"searchinfo": map[string]interface{}{"totalhits": float64(0)},
						"search":     []interface{}{},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ResolveTitle(ctx, ResolveTitleArgs{Title: "xyznonexistent"})

	if err != nil {
		t.Fatalf("ResolveTitle failed: %v", err)
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("Expected no suggestions, got %d", len(result.Suggestions))
	}
}

func TestResolveTitle_ExactMatch(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "query" {
			prop := r.FormValue("prop")
			// GetPageInfo query - return an existing page
			if prop != "" && strings.Contains(prop, "info") {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"123": map[string]interface{}{
								"pageid":       float64(123),
								"ns":           float64(0),
								"title":        "Test Page",
								"touched":      "2024-01-01T00:00:00Z",
								"length":       float64(1000),
								"lastrevid":    float64(456),
								"contentmodel": "wikitext",
								"pagelanguage": "en",
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ResolveTitle(ctx, ResolveTitleArgs{Title: "Test Page"})

	if err != nil {
		t.Fatalf("ResolveTitle failed: %v", err)
	}
	if !result.ExactMatch {
		t.Error("Expected ExactMatch = true")
	}
	if result.ResolvedTitle != "Test Page" {
		t.Errorf("ResolvedTitle = %q, want 'Test Page'", result.ResolvedTitle)
	}
	if result.PageID != 123 {
		t.Errorf("PageID = %d, want 123", result.PageID)
	}
}

func TestResolveTitle_HighSimilarity(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "query" {
			prop := r.FormValue("prop")
			list := r.FormValue("list")
			// GetPageInfo query - page doesn't exist
			if prop != "" && strings.Contains(prop, "info") {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"-1": map[string]interface{}{
								"ns":      float64(0),
								"title":   "Test Pag",
								"missing": "",
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			// Search query - return very similar result
			if list == "search" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"searchinfo": map[string]interface{}{"totalhits": float64(1)},
						"search": []interface{}{
							map[string]interface{}{
								"pageid":  float64(456),
								"title":   "Test Page", // Very similar to "Test Pag"
								"snippet": "The test page content",
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ResolveTitle(ctx, ResolveTitleArgs{Title: "Test Pag"})

	if err != nil {
		t.Fatalf("ResolveTitle failed: %v", err)
	}
	if result.ExactMatch {
		t.Error("Expected ExactMatch = false")
	}
	if len(result.Suggestions) == 0 {
		t.Error("Expected at least one suggestion")
	}
	// Should have a message about similar pages
	if result.Message == "" {
		t.Error("Expected a message, got empty string")
	}
}

func TestGetRelated_Success(t *testing.T) {
	callCount := 0
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		callCount++

		if action == "query" {
			prop := r.FormValue("prop")
			if prop == "categories" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"1": map[string]interface{}{
								"pageid": float64(1),
								"title":  "Test Page",
								"categories": []interface{}{
									map[string]interface{}{"title": "Category:Technology"},
									map[string]interface{}{"title": "Category:Software"},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			// Category members query
			if r.FormValue("list") == "categorymembers" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"categorymembers": []interface{}{
							map[string]interface{}{"pageid": float64(2), "title": "Related Page 1"},
							map[string]interface{}{"pageid": float64(3), "title": "Related Page 2"},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		// Default response
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetRelated(ctx, GetRelatedArgs{Title: "Test Page", Limit: 5})

	if err != nil {
		t.Fatalf("GetRelated failed: %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Page")
	}
}

func TestGetPageHTML_Success(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "parse" {
			response := map[string]interface{}{
				"parse": map[string]interface{}{
					"title":  "Test Page",
					"pageid": float64(1),
					"text": map[string]interface{}{
						"*": "<p>This is the page content in HTML.</p>",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetPage(ctx, GetPageArgs{Title: "Test Page", Format: "html"})

	if err != nil {
		t.Fatalf("GetPage with HTML failed: %v", err)
	}
	if result.Content == "" {
		t.Error("Expected HTML content, got empty")
	}
}

func TestGetImages_Success(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		prop := r.FormValue("prop")

		if action == "query" {
			if prop == "images" {
				// First request: get list of images on the page
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"1": map[string]interface{}{
								"pageid": float64(1),
								"title":  "Test Page",
								"images": []interface{}{
									map[string]interface{}{"title": "File:Logo.png"},
									map[string]interface{}{"title": "File:Icon.svg"},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			if prop == "imageinfo" {
				// Second request: get image details
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"1": map[string]interface{}{
								"pageid": float64(100),
								"title":  "File:Logo.png",
								"imageinfo": []interface{}{
									map[string]interface{}{
										"url":      "https://wiki.example.com/images/logo.png",
										"thumburl": "https://wiki.example.com/images/thumb/logo.png",
										"width":    float64(200),
										"height":   float64(100),
										"size":     float64(5000),
										"mime":     "image/png",
									},
								},
							},
							"2": map[string]interface{}{
								"pageid": float64(101),
								"title":  "File:Icon.svg",
								"imageinfo": []interface{}{
									map[string]interface{}{
										"url":      "https://wiki.example.com/images/icon.svg",
										"thumburl": "https://wiki.example.com/images/thumb/icon.svg",
										"width":    float64(64),
										"height":   float64(64),
										"size":     float64(1200),
										"mime":     "image/svg+xml",
									},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetImages(ctx, GetImagesArgs{Title: "Test Page"})

	if err != nil {
		t.Fatalf("GetImages failed: %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Page")
	}
	if result.Count != 2 {
		t.Errorf("Count = %d, want 2", result.Count)
	}
}

func TestGetImages_NoImages(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"query": map[string]interface{}{
				"pages": map[string]interface{}{
					"1": map[string]interface{}{
						"pageid": float64(1),
						"title":  "Test Page",
						// No images field
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetImages(ctx, GetImagesArgs{Title: "Test Page"})

	if err != nil {
		t.Fatalf("GetImages failed: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("Count = %d, want 0", result.Count)
	}
}

func TestGetImages_EmptyTitle(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	_, err := client.GetImages(ctx, GetImagesArgs{Title: ""})

	if err == nil {
		t.Fatal("Expected error for empty title")
	}
}

func TestListPages_WithPrefix(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		prefix := r.FormValue("apprefix")

		if prefix == "API" {
			response := map[string]interface{}{
				"query": map[string]interface{}{
					"allpages": []interface{}{
						map[string]interface{}{"pageid": float64(1), "title": "API Guide"},
						map[string]interface{}{"pageid": float64(2), "title": "API Reference"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{"allpages":[]}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.ListPages(ctx, ListPagesArgs{Prefix: "API"})

	if err != nil {
		t.Fatalf("ListPages failed: %v", err)
	}
	if len(result.Pages) != 2 {
		t.Errorf("Pages count = %d, want 2", len(result.Pages))
	}
}

func TestGetRelated_LinksMethod(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "query" {
			prop := r.FormValue("prop")
			if prop == "links" {
				// Return linked pages
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"1": map[string]interface{}{
								"pageid": float64(1),
								"title":  "Test Page",
								"links": []interface{}{
									map[string]interface{}{"ns": float64(0), "title": "Linked Page 1"},
									map[string]interface{}{"ns": float64(0), "title": "Linked Page 2"},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetRelated(ctx, GetRelatedArgs{
		Title:  "Test Page",
		Method: "links",
		Limit:  10,
	})

	if err != nil {
		t.Fatalf("GetRelated failed: %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Page")
	}
	if len(result.RelatedPages) != 2 {
		t.Errorf("RelatedPages count = %d, want 2", len(result.RelatedPages))
	}
}

func TestGetRelated_AllMethod(t *testing.T) {
	requestCount := 0
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_ = r.ParseForm()
		action := r.FormValue("action")
		if action == "query" {
			prop := r.FormValue("prop")
			if strings.Contains(prop, "categories") {
				// Return page categories
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"1": map[string]interface{}{
								"pageid": float64(1),
								"title":  "Test Page",
								"categories": []interface{}{
									map[string]interface{}{"title": "Category:Testing"},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			if prop == "links" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"pages": map[string]interface{}{
							"1": map[string]interface{}{
								"pageid": float64(1),
								"title":  "Test Page",
								"links": []interface{}{
									map[string]interface{}{"ns": float64(0), "title": "Related Topic"},
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			list := r.FormValue("list")
			if list == "categorymembers" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"categorymembers": []interface{}{
							map[string]interface{}{"pageid": float64(2), "title": "Category Sibling"},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
			if list == "backlinks" {
				response := map[string]interface{}{
					"query": map[string]interface{}{
						"backlinks": []interface{}{
							map[string]interface{}{"pageid": float64(3), "title": "Linking Page"},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetRelated(ctx, GetRelatedArgs{
		Title:  "Test Page",
		Method: "all",
		Limit:  10,
	})

	if err != nil {
		t.Fatalf("GetRelated failed: %v", err)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Page")
	}
}

func TestGetPage_HTMLFormat(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")

		if action == "parse" {
			response := map[string]interface{}{
				"parse": map[string]interface{}{
					"title":  "Test Page",
					"pageid": float64(1),
					"text": map[string]interface{}{
						"*": "<p>This is <b>HTML</b> content.</p>",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		response := map[string]interface{}{
			"query": map[string]interface{}{
				"pages": map[string]interface{}{
					"1": map[string]interface{}{
						"pageid": float64(1),
						"title":  "Test Page",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetPage(ctx, GetPageArgs{
		Title:  "Test Page",
		Format: "html",
	})

	if err != nil {
		t.Fatalf("GetPage HTML failed: %v", err)
	}

	if result.Format != "html" {
		t.Errorf("Format = %q, want 'html'", result.Format)
	}
}

func TestGetWikiInfo_AllStats(t *testing.T) {
	server := mockMediaWikiServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		meta := r.FormValue("meta")

		if meta == "siteinfo" {
			response := map[string]interface{}{
				"query": map[string]interface{}{
					"general": map[string]interface{}{
						"sitename":    "Test Wiki",
						"mainpage":    "Main Page",
						"base":        "https://wiki.example.com",
						"generator":   "MediaWiki 1.39",
						"logo":        "https://wiki.example.com/logo.png",
						"articlepath": "/wiki/$1",
						"servername":  "wiki.example.com",
						"timezone":    "UTC",
						"timeoffset":  float64(0),
						"wikiid":      "testwiki",
						"phpversion":  "8.1.0",
					},
					"statistics": map[string]interface{}{
						"pages":       float64(1000),
						"articles":    float64(500),
						"edits":       float64(5000),
						"images":      float64(200),
						"users":       float64(50),
						"activeusers": float64(10),
						"admins":      float64(3),
					},
					"namespaces": map[string]interface{}{
						"0":  map[string]interface{}{"id": float64(0), "*": "Main", "canonical": ""},
						"1":  map[string]interface{}{"id": float64(1), "*": "Talk", "canonical": "Talk"},
						"10": map[string]interface{}{"id": float64(10), "*": "Template", "canonical": "Template"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query":{}}`))
	})
	defer server.Close()

	client := createMockClient(t, server)
	defer client.Close()

	ctx := context.Background()
	result, err := client.GetWikiInfo(ctx, WikiInfoArgs{})

	if err != nil {
		t.Fatalf("GetWikiInfo failed: %v", err)
	}

	if result.SiteName != "Test Wiki" {
		t.Errorf("SiteName = %q, want 'Test Wiki'", result.SiteName)
	}

	if result.Statistics.Pages != 1000 {
		t.Errorf("Pages = %d, want 1000", result.Statistics.Pages)
	}
}
