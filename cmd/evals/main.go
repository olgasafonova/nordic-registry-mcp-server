// Command evals runs MCP tool selection evaluations.
//
// Usage:
//
//	go run ./cmd/evals -dir ./evals -suite all
//
// This command loads evaluation test suites and reports on test coverage
// and expected behavior patterns. For actual LLM evaluation, integrate
// the evals package with your LLM testing framework.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olgasafonova/mediawiki-mcp-server/evals"
)

func main() {
	dir := flag.String("dir", "./evals", "Directory containing eval JSON files")
	suite := flag.String("suite", "all", "Suite to load: tool_selection, confusion_pairs, arguments, or all")
	verbose := flag.Bool("verbose", false, "Show detailed test information")
	flag.Parse()

	fmt.Println("MediaWiki MCP Server - Evaluation Framework")
	fmt.Println("============================================")
	fmt.Println()

	switch *suite {
	case "tool_selection":
		loadToolSelection(*dir, *verbose)
	case "confusion_pairs":
		loadConfusionPairs(*dir, *verbose)
	case "arguments":
		loadArguments(*dir, *verbose)
	case "all":
		loadAll(*dir, *verbose)
	default:
		fmt.Fprintf(os.Stderr, "Unknown suite: %s\n", *suite)
		os.Exit(1)
	}
}

func loadToolSelection(dir string, verbose bool) {
	path := filepath.Join(dir, "tool_selection.json")
	suite, err := evals.LoadToolSelectionSuite(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tool selection suite: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tool Selection Suite: %s\n", suite.Name)
	fmt.Printf("Version: %s\n", suite.Version)
	fmt.Printf("Description: %s\n", suite.Description)
	fmt.Printf("Total Tests: %d\n", len(suite.Tests))
	fmt.Println()

	// Count by category
	categories := make(map[string]int)
	tools := make(map[string]int)
	for _, test := range suite.Tests {
		categories[test.Category]++
		tools[test.ExpectedTool]++
	}

	fmt.Println("Tests by Category:")
	for cat, count := range categories {
		fmt.Printf("  %-15s: %d\n", cat, count)
	}
	fmt.Println()

	fmt.Println("Tests by Tool:")
	for tool, count := range tools {
		fmt.Printf("  %-40s: %d\n", tool, count)
	}
	fmt.Println()

	if verbose {
		fmt.Println("Test Cases:")
		for _, test := range suite.Tests {
			fmt.Printf("  [%s] %s\n", test.ID, test.Input)
			fmt.Printf("    → %s\n", test.ExpectedTool)
			if len(test.NotTools) > 0 {
				fmt.Printf("    ✗ %v\n", test.NotTools)
			}
		}
	}
}

func loadConfusionPairs(dir string, verbose bool) {
	path := filepath.Join(dir, "confusion_pairs.json")
	suite, err := evals.LoadConfusionPairSuite(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading confusion pairs suite: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Confusion Pairs Suite: %s\n", suite.Name)
	fmt.Printf("Version: %s\n", suite.Version)
	fmt.Printf("Description: %s\n", suite.Description)
	fmt.Printf("Total Pairs: %d\n", len(suite.Pairs))

	totalTests := 0
	for _, pair := range suite.Pairs {
		totalTests += len(pair.Tests)
	}
	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Println()

	fmt.Println("Confusion Pairs:")
	for _, pair := range suite.Pairs {
		fmt.Printf("\n  %s:\n", pair.ID)
		fmt.Printf("    Tools: %v\n", pair.Tools)
		fmt.Printf("    Rule: %s\n", pair.Disambiguation)
		fmt.Printf("    Tests: %d\n", len(pair.Tests))

		if verbose {
			for _, test := range pair.Tests {
				fmt.Printf("      \"%s\"\n", test.Input)
				fmt.Printf("        → %s (%s)\n", test.Expected, test.Reason)
			}
		}
	}
	fmt.Println()
}

func loadArguments(dir string, verbose bool) {
	path := filepath.Join(dir, "argument_correctness.json")
	suite, err := evals.LoadArgumentSuite(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading argument suite: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Argument Suite: %s\n", suite.Name)
	fmt.Printf("Version: %s\n", suite.Version)
	fmt.Printf("Description: %s\n", suite.Description)
	fmt.Printf("Total Tests: %d\n", len(suite.Tests))
	fmt.Println()

	// Count by tool
	tools := make(map[string]int)
	for _, test := range suite.Tests {
		tools[test.Tool]++
	}

	fmt.Println("Tests by Tool:")
	for tool, count := range tools {
		fmt.Printf("  %-40s: %d\n", tool, count)
	}
	fmt.Println()

	fmt.Println("Validation Rules:")
	fmt.Printf("  Title Format: %s\n", suite.ValidationRules.TitleFormat)
	fmt.Printf("  Category Format: %s\n", suite.ValidationRules.CategoryFormat)
	fmt.Printf("  Boolean Handling: %s\n", suite.ValidationRules.BooleanHandling)
	fmt.Printf("  Array Handling: %s\n", suite.ValidationRules.ArrayHandling)
	fmt.Printf("  Preview Default: %s\n", suite.ValidationRules.PreviewDefault)
	fmt.Println()

	if verbose {
		fmt.Println("Test Cases:")
		for _, test := range suite.Tests {
			fmt.Printf("  [%s] %s\n", test.ID, test.Input)
			fmt.Printf("    Tool: %s\n", test.Tool)
			fmt.Printf("    Required: %v\n", test.RequiredArgs)
			fmt.Printf("    Expected: %v\n", test.ExpectedArgs)
			if len(test.ForbiddenArgs) > 0 {
				fmt.Printf("    Forbidden: %v\n", test.ForbiddenArgs)
			}
			if test.ArgNotes != "" {
				fmt.Printf("    Notes: %s\n", test.ArgNotes)
			}
		}
	}
}

func loadAll(dir string, verbose bool) {
	toolSelection, confusionPairs, arguments, err := evals.LoadAllEvals(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading evals: %v\n", err)
		os.Exit(1)
	}

	// Count totals
	totalTests := len(toolSelection.Tests)
	for _, pair := range confusionPairs.Pairs {
		totalTests += len(pair.Tests)
	}
	totalTests += len(arguments.Tests)

	fmt.Printf("Loaded all evaluation suites from: %s\n\n", dir)

	fmt.Println("Summary:")
	fmt.Println("--------")
	fmt.Printf("Tool Selection Tests:   %d\n", len(toolSelection.Tests))

	confusionTests := 0
	for _, pair := range confusionPairs.Pairs {
		confusionTests += len(pair.Tests)
	}
	fmt.Printf("Confusion Pair Tests:   %d (across %d pairs)\n", confusionTests, len(confusionPairs.Pairs))
	fmt.Printf("Argument Tests:         %d\n", len(arguments.Tests))
	fmt.Printf("──────────────────────────\n")
	fmt.Printf("Total Evaluation Tests: %d\n", totalTests)
	fmt.Println()

	// Show tool coverage
	toolCoverage := make(map[string]bool)
	for _, test := range toolSelection.Tests {
		toolCoverage[test.ExpectedTool] = true
	}
	for _, pair := range confusionPairs.Pairs {
		for _, tool := range pair.Tools {
			toolCoverage[tool] = true
		}
	}
	for _, test := range arguments.Tests {
		toolCoverage[test.Tool] = true
	}

	fmt.Printf("Tool Coverage: %d unique tools tested\n", len(toolCoverage))

	if verbose {
		fmt.Println("\nCovered Tools:")
		for tool := range toolCoverage {
			fmt.Printf("  ✓ %s\n", tool)
		}
	}

	fmt.Println()
	fmt.Println("To run with LLM integration, implement the evals.ToolSelector interface")
	fmt.Println("and use EvaluateToolSelection(), EvaluateConfusionPairs(), EvaluateArguments()")
}
