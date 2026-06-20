# Quick Start

## Using with Claude Desktop

### 1. Configure Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "aha": {
      "command": "/path/to/aha-mcp-server",
      "env": {
        "AHA_DOMAIN": "your_subdomain",
        "AHA_API_TOKEN": "your_api_token"
      }
    }
  }
}
```

### 2. Restart Claude Desktop

Close and reopen Claude Desktop to load the new configuration.

### 3. Start Using

You can now ask Claude to interact with your Aha! data:

- "Search for documents about product roadmap"
- "Get the details for feature FEAT-123"
- "Show me epic EPIC-456"
- "List all ideas tagged with 'mobile'"

## Using aha-studio CLI

The `aha-studio` CLI provides AQL (Aha Query Language) for powerful querying and export:

```bash
# Set credentials
export AHA_SUBDOMAIN=mycompany
export AHA_API_KEY=your-api-key

# Basic query
aha-studio query "FROM features LIMIT 10"

# Query with filter
aha-studio query "FROM ideas WHERE status = 'Shipped' LIMIT 5"

# Sync data for offline queries
aha-studio sync --product PROD

# Export features to Excel ordered by rank
aha-studio query -o xlsx -f features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features ORDER BY position ASC"

# Export features for a specific release
aha-studio query -o xlsx -f release-features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features WHERE release_id = 'PROD-R-123' ORDER BY position ASC"

# Interactive shell
aha-studio shell
```

## Using aha-mcp-server CLI

The MCP server also works as a CLI tool for testing and scripting:

```bash
# Search documents
aha-mcp-server search-documents "authentication"

# Get a feature
aha-mcp-server get-feature FEAT-123

# List ideas with filtering
aha-mcp-server list-ideas --query "mobile" --workflow-status "Under consideration"

# Get an epic
aha-mcp-server get-epic EPIC-456

# Pretty print output
aha-mcp-server get-idea IDEA-789 --output pretty
```

## Example Workflows

### Research a Topic

```
You: Search Aha! for documents about our API redesign

Claude: I'll search for documents about API redesign.
[Uses search_documents tool]

Found 5 documents:
1. "API v2 Roadmap" (Page)
2. "REST API Design Guidelines" (Page)
...
```

### Review a Feature

```
You: Get the details for feature FEAT-123 and its requirements

Claude: I'll get the feature details.
[Uses get_feature tool]

Feature FEAT-123: "User Authentication Overhaul"
Status: In Development
...
```

### Explore Ideas

```
You: List the most recent ideas tagged "mobile"

Claude: I'll list ideas with the mobile tag.
[Uses list_ideas tool]

Found 12 ideas:
1. IDEA-789: "Mobile app offline mode"
2. IDEA-790: "Push notifications for updates"
...
```
