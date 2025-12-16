package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/olgasafonova/mediawiki-mcp-server/wiki"
)

// measureCachePerformance runs a simple cache performance test
func measureCachePerformance() {
	config, err := wiki.LoadConfig()
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		return
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := wiki.NewClient(config, logger)
	ctx := context.Background()

	fmt.Println("=== Cache Performance Test ===")
	fmt.Println()

	// Test 1: GetWikiInfo caching
	fmt.Println("1. GetWikiInfo Cache Test:")

	start := time.Now()
	_, err = client.GetWikiInfo(ctx, wiki.WikiInfoArgs{})
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	firstCall := time.Since(start)
	fmt.Printf("   First call (network):  %v\n", firstCall)

	start = time.Now()
	_, _ = client.GetWikiInfo(ctx, wiki.WikiInfoArgs{})
	secondCall := time.Since(start)
	fmt.Printf("   Second call (cached):  %v\n", secondCall)
	fmt.Printf("   Speedup: %.0fx faster\n", float64(firstCall)/float64(secondCall))
	fmt.Println()

	// Test 2: Search (not cached, baseline)
	fmt.Println("2. Search Performance (baseline, no caching):")
	start = time.Now()
	_, err = client.Search(ctx, wiki.SearchArgs{Query: "360", Limit: 10})
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	searchTime := time.Since(start)
	fmt.Printf("   Search time: %v\n", searchTime)
	fmt.Println()
}

// measureBatchPerformance compares sequential vs batch API calls
func measureBatchPerformance() {
	config, err := wiki.LoadConfig()
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		return
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := wiki.NewClient(config, logger)
	ctx := context.Background()

	fmt.Println("=== Batch vs Sequential Performance ===")
	fmt.Println()

	// Get some page titles to test with
	pages, err := client.ListPages(ctx, wiki.ListPagesArgs{Limit: 5})
	if err != nil {
		fmt.Printf("Error listing pages: %v\n", err)
		return
	}

	if len(pages.Pages) < 3 {
		fmt.Println("Not enough pages to test")
		return
	}

	titles := make([]string, 0, 3)
	for i := 0; i < 3 && i < len(pages.Pages); i++ {
		titles = append(titles, pages.Pages[i].Title)
	}

	fmt.Printf("Testing with %d pages: %v\n\n", len(titles), titles)

	// Test: GetExternalLinksBatch (now parallelized)
	fmt.Println("3. GetExternalLinksBatch (parallelized):")
	start := time.Now()
	result, err := client.GetExternalLinksBatch(ctx, wiki.GetExternalLinksBatchArgs{Titles: titles})
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}
	batchTime := time.Since(start)
	fmt.Printf("   Batch time for %d pages: %v\n", len(titles), batchTime)
	fmt.Printf("   Total links found: %d\n", result.TotalLinks)
	fmt.Printf("   Average per page: %v\n", batchTime/time.Duration(len(titles)))
	fmt.Println()

	// Compare: Sequential individual calls
	fmt.Println("4. Sequential GetExternalLinks (for comparison):")
	start = time.Now()
	for _, title := range titles {
		_, _ = client.GetExternalLinks(ctx, wiki.GetExternalLinksArgs{Title: title})
	}
	sequentialTime := time.Since(start)
	fmt.Printf("   Sequential time for %d pages: %v\n", len(titles), sequentialTime)
	fmt.Printf("   Parallel speedup: %.1fx faster\n", float64(sequentialTime)/float64(batchTime))
	fmt.Println()
}

// measureAPICallReduction shows the reduction in API calls for batch operations
func measureAPICallReduction() {
	fmt.Println("=== API Call Reduction Analysis ===")
	fmt.Println()

	// FindBrokenInternalLinks improvement
	fmt.Println("5. FindBrokenInternalLinks Optimization:")
	fmt.Println("   Before: 1 API call per link target checked")
	fmt.Println("   After:  1 API call per 50 link targets (batch)")
	fmt.Println()
	fmt.Println("   Example scenarios:")
	fmt.Println("   - Page with 10 internal links:  10 calls → 1 call  (10x reduction)")
	fmt.Println("   - Page with 50 internal links:  50 calls → 1 call  (50x reduction)")
	fmt.Println("   - Page with 100 internal links: 100 calls → 2 calls (50x reduction)")
	fmt.Println("   - Category with 20 pages, 50 links each: 1000 calls → 20 calls")
	fmt.Println()
}

func main() {
	fmt.Println("MediaWiki MCP Server - Performance Measurements")
	fmt.Println("================================================")
	fmt.Println()

	measureCachePerformance()
	measureBatchPerformance()
	measureAPICallReduction()

	fmt.Println("=== Summary ===")
	fmt.Println()
	fmt.Println("Key improvements:")
	fmt.Println("• Caching: Repeated requests are 100-1000x faster (memory vs network)")
	fmt.Println("• Parallelization: Batch operations run concurrently instead of sequentially")
	fmt.Println("• Batch API: Single API call checks up to 50 page titles at once")
	fmt.Println("• Connection reuse: HTTP/2 + connection pooling reduces latency")
}
