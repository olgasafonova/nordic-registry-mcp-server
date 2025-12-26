# MediaWiki MCP Server Evaluations

This directory contains evaluation test suites for validating LLM tool selection accuracy with the MediaWiki MCP server.

## Overview

MCP (Model Context Protocol) evaluations test whether LLMs correctly:
1. **Select the right tool** for a given natural language request
2. **Disambiguate similar tools** that could easily be confused
3. **Extract correct arguments** from natural language inputs

## Test Suites

### `tool_selection.json`
Tests for correct tool selection across all tool categories:
- Search (wiki-wide vs page-specific)
- Read (pages, sections, info)
- History (revisions, contributions, recent changes)
- Categories and links
- Quality checks (terminology, translations, audits)
- Write operations (edit, find-replace, formatting)

### `confusion_pairs.json`
Tests for disambiguating commonly confused tool pairs:
- `mediawiki_search` vs `mediawiki_search_in_page`
- `mediawiki_get_page` vs `mediawiki_get_sections`
- `mediawiki_edit_page` vs `mediawiki_find_replace`
- `mediawiki_find_replace` vs `mediawiki_bulk_replace`
- `mediawiki_apply_formatting` vs `mediawiki_find_replace`
- And more...

### `argument_correctness.json`
Tests for correct argument extraction:
- Required arguments are present
- Expected values match
- Forbidden arguments are not used
- Types are correct (arrays, booleans, etc.)

## Running Evals

### View eval summary
```bash
go run ./cmd/evals -dir ./evals -suite all
```

### View specific suite with details
```bash
go run ./cmd/evals -dir ./evals -suite tool_selection -verbose
```

### Run eval tests
```bash
go test ./evals/...
```

## Integrating with LLM Testing

To run actual evaluations with an LLM, implement the `ToolSelector` interface:

```go
type ToolSelector interface {
    SelectTool(input string) (toolName string, args map[string]interface{}, err error)
}
```

Then use the evaluation functions:

```go
import "github.com/olgasafonova/mediawiki-mcp-server/evals"

// Load suites
toolSelection, confusionPairs, arguments, _ := evals.LoadAllEvals("./evals")

// Create your LLM-backed selector
selector := &MyLLMSelector{...}

// Run evaluations
metrics1, _ := evals.EvaluateToolSelection(toolSelection, selector)
metrics2, _ := evals.EvaluateConfusionPairs(confusionPairs, selector)
metrics3, _ := evals.EvaluateArguments(arguments, selector)

// Print results
fmt.Println(evals.FormatMetrics(metrics1, "Tool Selection"))
fmt.Println(evals.FormatMetrics(metrics2, "Confusion Pairs"))
fmt.Println(evals.FormatMetrics(metrics3, "Argument Correctness"))
```

## Metrics

The evaluation framework tracks:

- **Accuracy**: Percentage of tests passed
- **By Category**: Breakdown by tool category
- **By Tool**: Per-tool precision and recall
- **False Positives**: Times wrong tool was selected
- **False Negatives**: Times correct tool was missed

## Best Practices for Tool Descriptions

Based on evaluation research, effective tool descriptions include:

1. **USE WHEN** section: Natural language triggers
2. **NOT FOR** section: Disambiguation from similar tools
3. **PARAMETERS**: With types and defaults
4. **RETURNS**: What the tool outputs
5. **EXAMPLES**: Real usage patterns

Example:
```
USE WHEN: User asks "find pages about X", "where is X documented"
NOT FOR: Searching within a specific known page (use mediawiki_search_in_page)
PARAMETERS:
- query: Search text (required)
- limit: Max results (default 10)
RETURNS: Page titles, snippets with highlights
```

## Adding New Tests

Follow the existing JSON schema when adding tests:

```json
{
  "id": "unique-test-id",
  "category": "search|read|write|etc",
  "input": "natural language user request",
  "expected_tool": "mediawiki_tool_name",
  "expected_args": {"arg": "value"},
  "not_tools": ["tools_that_should_not_be_selected"]
}
```

## References

- [MCP Evals Best Practices](https://mcpevals.io)
- [GitHub's MCP Server Evaluation Approach](https://github.blog/engineering/building-github-mcp-server/)
- [Neon Case Study on Tool Selection](https://neon.tech/blog/mcp-tool-selection)
