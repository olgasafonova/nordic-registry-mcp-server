# Changelog

All notable changes to MediaWiki MCP Server are documented here.

## [1.20.0] - 2025-12-19

### Added
- Increased test coverage for wiki package to 40%+
- Comprehensive test suite improvements

## [1.19.0] - 2025-12-19

### Added
- Metadata-driven tool registry for cleaner tool definitions
- Improved tool organization and maintainability

## [1.18.0] - 2025-12-19

### Added
- Audit logging for all write operations
- Track who made changes and when

### Documentation
- Added ARCHITECTURE.md
- Added CONTRIBUTING.md

## [1.17.8] - 2025-12-19

### Security
- Fixed gosec security scan findings
- Added Unicode NFC normalization for input sanitization
- Added fail-closed DNS handling with structured error codes
- Fixed DNS rebinding vulnerability in CheckLinks (SSRF protection)

### Refactored
- Split methods.go into logical modules for better maintainability

## [1.17.3] - 2025-12-18

### Fixed
- Fixed enum struct tag crash on Windows

## [1.17.2] - 2025-12-17

### Fixed
- Fixed unchecked errors and data race in cache
- Fixed unchecked error returns in tests

## [1.16.0] - 2025-12-17

### Added
- `mediawiki_convert_markdown` tool for Markdown to MediaWiki conversion
- Theme support: tieto, neutral, dark
- Options: add_css, reverse_changelog, prettify_checks

## [1.15.0] - 2025-12-16

### Added
- Revision info and undo instructions in edit responses
- Edit operations now return old/new revision IDs and diff URLs

## [1.14.0] - 2025-12-16

### Added
- `mediawiki_find_similar_pages` - Find related content based on term overlap
- `mediawiki_compare_topic` - Compare how topics are described across pages
- Content discovery tools for finding inconsistencies

### Fixed
- Fixed nil slice JSON serialization in content discovery tools

## [1.13.0] - 2025-12-15

### Added
- `aggregate_by` parameter for `get_recent_changes`
- Aggregate by user, page, or change type for compact summaries

## [1.12.0] - 2025-12-14

### Added
- `mediawiki_get_sections` - Get section structure or specific section content
- `mediawiki_get_related` - Find related pages via categories/links
- `mediawiki_get_images` - Get images used on a page
- `mediawiki_upload_file` - Upload files from URL
- `mediawiki_search_in_file` - Search text in PDFs and text files

## [1.11.0] - 2025-12-13

### Added
- HTTP transport mode for ChatGPT, n8n integration
- Bearer token authentication
- CORS origin validation
- Rate limiting (configurable requests per minute)

## [1.10.0] - 2025-12-12

### Added
- Google ADK integration support (Go and Python)
- Streamable HTTP transport

## [1.0.0] - 2025-12-01

### Added
- Initial release
- Core MediaWiki API operations
- Search, read, and edit wiki content
- Link analysis and quality checks
- History and revision tracking
- Claude Desktop, Claude Code, Cursor support
