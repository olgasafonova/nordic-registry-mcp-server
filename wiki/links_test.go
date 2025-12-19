package wiki

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Helper function to create a mock server for links tests
func createLinksMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		action := r.FormValue("action")
		meta := r.FormValue("meta")

		// Handle userinfo query (session check)
		if action == "query" && meta == "userinfo" {
			response := map[string]interface{}{
				"query": map[string]interface{}{
					"userinfo": map[string]interface{}{
						"id":   float64(1),
						"name": "TestUser",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		// Handle login token request
		if action == "query" && (meta == "tokens") {
			tokenType := r.FormValue("type")
			response := map[string]interface{}{
				"query": map[string]interface{}{
					"tokens": map[string]interface{}{},
				},
			}
			tokens := response["query"].(map[string]interface{})["tokens"].(map[string]interface{})
			if tokenType == "login" {
				tokens["logintoken"] = "test-login-token"
			} else if tokenType == "csrf" {
				tokens["csrftoken"] = "test-csrf-token"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		// Handle login action
		if action == "login" {
			response := map[string]interface{}{
				"login": map[string]interface{}{
					"result": "Success",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
			return
		}

		// Delegate to custom handler
		handler(w, r)
	}))
}

func createLinksTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	config := &Config{
		BaseURL:    server.URL,
		Username:   "TestUser",
		Password:   "TestPass",
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		UserAgent:  "TestClient/1.0",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewClient(config, logger)
}

func TestGetExternalLinks_Success(t *testing.T) {
	server := createLinksMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"query": map[string]interface{}{
				"pages": map[string]interface{}{
					"123": map[string]interface{}{
						"pageid": float64(123),
						"title":  "Test Page",
						"extlinks": []interface{}{
							map[string]interface{}{
								"*":   "https://example.com",
								"url": "https://example.com",
							},
							map[string]interface{}{
								"*":   "http://test.org/page",
								"url": "http://test.org/page",
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createLinksTestClient(t, server)
	defer client.Close()

	result, err := client.GetExternalLinks(context.Background(), GetExternalLinksArgs{
		Title: "Test Page",
	})
	if err != nil {
		t.Fatalf("GetExternalLinks failed: %v", err)
	}

	if len(result.Links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(result.Links))
	}
	if result.Links[0].URL != "https://example.com" {
		t.Errorf("First link URL = %q, want 'https://example.com'", result.Links[0].URL)
	}
	if result.Links[0].Protocol != "https" {
		t.Errorf("First link protocol = %q, want 'https'", result.Links[0].Protocol)
	}
}

func TestGetExternalLinks_EmptyTitle(t *testing.T) {
	config := &Config{
		BaseURL:    "https://test.wiki.com/api.php",
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		UserAgent:  "TestClient/1.0",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewClient(config, logger)
	defer client.Close()

	_, err := client.GetExternalLinks(context.Background(), GetExternalLinksArgs{
		Title: "",
	})
	if err == nil {
		t.Error("Expected error for empty title")
	}
}

func TestGetExternalLinks_PageNotFound(t *testing.T) {
	server := createLinksMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"query": map[string]interface{}{
				"pages": map[string]interface{}{
					"-1": map[string]interface{}{
						"missing": "",
						"title":   "Missing Page",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createLinksTestClient(t, server)
	defer client.Close()

	_, err := client.GetExternalLinks(context.Background(), GetExternalLinksArgs{
		Title: "Missing Page",
	})
	if err == nil {
		t.Error("Expected error for missing page")
	}
}

func TestGetBacklinks_Success(t *testing.T) {
	server := createLinksMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"query": map[string]interface{}{
				"backlinks": []interface{}{
					map[string]interface{}{
						"pageid": float64(101),
						"title":  "Page One",
						"ns":     float64(0),
					},
					map[string]interface{}{
						"pageid": float64(102),
						"title":  "Page Two",
						"ns":     float64(0),
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createLinksTestClient(t, server)
	defer client.Close()

	result, err := client.GetBacklinks(context.Background(), GetBacklinksArgs{
		Title: "Target Page",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}

	if len(result.Backlinks) != 2 {
		t.Errorf("Expected 2 backlinks, got %d", len(result.Backlinks))
	}
	if result.Backlinks[0].Title != "Page One" {
		t.Errorf("First backlink title = %q, want 'Page One'", result.Backlinks[0].Title)
	}
}

func TestGetBacklinks_EmptyTitle(t *testing.T) {
	config := &Config{
		BaseURL:    "https://test.wiki.com/api.php",
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		UserAgent:  "TestClient/1.0",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewClient(config, logger)
	defer client.Close()

	_, err := client.GetBacklinks(context.Background(), GetBacklinksArgs{
		Title: "",
	})
	if err == nil {
		t.Error("Expected error for empty title")
	}
}

func TestGetBacklinks_Continuation(t *testing.T) {
	server := createLinksMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"query": map[string]interface{}{
				"backlinks": []interface{}{
					map[string]interface{}{
						"pageid": float64(101),
						"title":  "Page One",
					},
				},
			},
			"continue": map[string]interface{}{
				"blcontinue": "0|continue-token",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := createLinksTestClient(t, server)
	defer client.Close()

	result, err := client.GetBacklinks(context.Background(), GetBacklinksArgs{
		Title: "Target",
	})
	if err != nil {
		t.Fatalf("GetBacklinks failed: %v", err)
	}

	if !result.HasMore {
		t.Error("Expected HasMore to be true")
	}
}

func TestGetExternalLinksBatch_EmptyTitles(t *testing.T) {
	config := &Config{
		BaseURL:    "https://test.wiki.com/api.php",
		Timeout:    5 * time.Second,
		MaxRetries: 1,
		UserAgent:  "TestClient/1.0",
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewClient(config, logger)
	defer client.Close()

	_, err := client.GetExternalLinksBatch(context.Background(), GetExternalLinksBatchArgs{
		Titles: []string{},
	})
	if err == nil {
		t.Error("Expected error for empty titles")
	}
}
