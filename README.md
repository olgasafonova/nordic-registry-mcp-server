# MediaWiki MCP Server

A Model Context Protocol (MCP) server that enables AI assistants to interact with MediaWiki-powered wikis. Search, read, and edit wiki content directly from Claude, Cursor, or any MCP-compatible client.

## Features

- **Search** - Full-text search across wiki pages
- **Read** - Get page content in wikitext or HTML format
- **Browse** - List pages, categories, and recent changes
- **Edit** - Create and modify wiki pages (with bot password authentication)
- **Parse** - Preview wikitext rendering before saving

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/olgasafonova/mediawiki-mcp-server.git
cd mediawiki-mcp-server

# Build
go build -o mediawiki-mcp-server .
```

### Configuration

Set the required environment variable:

```bash
export MEDIAWIKI_URL="https://wiki.software-innovation.com/api.php"
```

For editing capabilities, also set:

```bash
export MEDIAWIKI_USERNAME="YourUsername@BotName"
export MEDIAWIKI_PASSWORD="your-bot-password"
```

### Usage with Claude Desktop

Add to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "mediawiki": {
      "command": "/path/to/mediawiki-mcp-server",
      "env": {
        "MEDIAWIKI_URL": "https://wiki.software-innovation.com/api.php"
      }
    }
  }
}
```

### Usage with Claude Code

```bash
claude mcp add mediawiki /path/to/mediawiki-mcp-server \
  -e MEDIAWIKI_URL="https://wiki.software-innovation.com/api.php"
```

## Available Tools

### Read Operations

| Tool | Description |
|------|-------------|
| `mediawiki_search` | Search for pages by text |
| `mediawiki_get_page` | Get page content (wikitext or HTML) |
| `mediawiki_list_pages` | List all pages with pagination |
| `mediawiki_list_categories` | List all categories |
| `mediawiki_get_category_members` | Get pages in a category |
| `mediawiki_get_page_info` | Get page metadata |
| `mediawiki_get_recent_changes` | Get recent wiki changes |
| `mediawiki_get_wiki_info` | Get wiki information and statistics |
| `mediawiki_parse` | Parse wikitext to HTML |

### Write Operations

| Tool | Description |
|------|-------------|
| `mediawiki_edit_page` | Create or edit a page |

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `MEDIAWIKI_URL` | Yes | Wiki API endpoint (e.g., `https://wiki.example.com/api.php`) |
| `MEDIAWIKI_USERNAME` | No | Bot username for editing |
| `MEDIAWIKI_PASSWORD` | No | Bot password for editing |
| `MEDIAWIKI_TIMEOUT` | No | Request timeout (default: 30s) |
| `MEDIAWIKI_USER_AGENT` | No | Custom User-Agent header |
| `MEDIAWIKI_MAX_RETRIES` | No | Max retry attempts (default: 3) |

## Authentication

For read-only operations, no authentication is required.

For editing, you need a **bot password**:

1. Go to `Special:BotPasswords` on your wiki
2. Create a new bot password with the permissions you need
3. Use the format `Username@BotName` for `MEDIAWIKI_USERNAME`
4. Use the generated password for `MEDIAWIKI_PASSWORD`

## Examples

### Search for pages

```
Search the wiki for "installation guide"
```

### Get page content

```
Get the content of the page "Main Page" in wikitext format
```

### List categories

```
List all categories that start with "360"
```

### Edit a page

```
Create a new page called "Test Page" with the content "Hello, wiki!"
```

## Response Formats

All tools return structured JSON responses with:
- Pagination support (`has_more`, `continue_from`)
- Character limits (25,000 chars max per response)
- Clear truncation indicators when content is cut off

## Development

### Build

```bash
go build -o mediawiki-mcp-server .
```

### Test

```bash
# Set environment
export MEDIAWIKI_URL="https://wiki.software-innovation.com/api.php"

# Run (will wait for MCP client connection)
./mediawiki-mcp-server
```

## License

MIT License

## Credits

- Built with the [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
- Powered by [MediaWiki API](https://www.mediawiki.org/wiki/API:Main_page)
