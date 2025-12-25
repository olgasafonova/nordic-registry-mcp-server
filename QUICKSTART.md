# Quick Start

Get MediaWiki MCP Server running in 2 minutes.

## 1. Install

**macOS/Linux (one-liner):**
```bash
curl -fsSL https://raw.githubusercontent.com/olgasafonova/mediawiki-mcp-server/main/install.sh | sh
```

**Or download manually:**
Go to [Releases](https://github.com/olgasafonova/mediawiki-mcp-server/releases) and download for your platform.

## 2. Configure Claude Code

```bash
claude mcp add mediawiki mediawiki-mcp-server \
  -e MEDIAWIKI_URL="https://en.wikipedia.org/w/api.php"
```

That's it! Ask Claude: *"Search the wiki for Albert Einstein"*

---

## Other AI Tools

### Claude Desktop (Mac)

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mediawiki": {
      "command": "mediawiki-mcp-server",
      "env": {
        "MEDIAWIKI_URL": "https://en.wikipedia.org/w/api.php"
      }
    }
  }
}
```

Restart Claude Desktop.

### Cursor

Edit `~/Library/Application Support/Cursor/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`:

```json
{
  "mcpServers": {
    "mediawiki": {
      "command": "mediawiki-mcp-server",
      "env": {
        "MEDIAWIKI_URL": "https://en.wikipedia.org/w/api.php"
      }
    }
  }
}
```

Restart Cursor.

---

## Need to Edit Pages?

Add bot credentials:

```bash
claude mcp add mediawiki mediawiki-mcp-server \
  -e MEDIAWIKI_URL="https://your-wiki.com/api.php" \
  -e MEDIAWIKI_USERNAME="User@BotName" \
  -e MEDIAWIKI_PASSWORD="your-bot-password"
```

Create a bot password at `Special:BotPasswords` on your wiki.

---

## Try These Prompts

- *"Search the wiki for getting started"*
- *"What does the API page say?"*
- *"Who edited this page last week?"*
- *"Check for broken links on the docs page"*

See [README.md](README.md) for full documentation.
