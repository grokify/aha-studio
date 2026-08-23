# Aha Studio

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/aha-studio/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/aha-studio/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/aha-studio/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/aha-studio/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/aha-studio/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/aha-studio/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/aha-studio
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/aha-studio
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://grokify.github.io/aha-studio
 [viz-svg]: https://img.shields.io/badge/visualization-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Faha-studio
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/aha-studio/blob/main/LICENSE

CLI tools for [Aha!](https://www.aha.io/) product management with AQL (Aha Query Language), MCP server integration, local SQLite sync, and Neo4j graph analytics.

## Overview

Aha Studio provides two command-line tools:

| Binary | Purpose |
|--------|---------|
| `aha-studio` | AQL query CLI with SQLite sync and interactive shell |
| `aha-mcp-server` | MCP server (87 tools) for Claude Desktop and AI assistants |

## Features

- 🔍 **AQL (Aha Query Language)** - SQL-like syntax for querying Aha.io data
- 🛠️ **87 MCP Tools** - Features, Ideas, Releases, Initiatives, Graph queries, analytics, and more
- 📊 **HTTP API** - REST endpoints with OpenAPI spec and Prometheus metrics
- 📡 **OmniSignal provider** - Normalizes Aha Ideas into vendor-neutral signals for the ProductContext signal pipeline
- 📡 **OmniSignalCache provider** - Cache-backed OmniSignal provider with voter/customer-org tiering (registered as `aha-studio`) — no Aha API traffic
- 🗺️ **OmniRoadmap provider** - Serves the local cache as tool-agnostic roadmap data (registered as `aha-studio` in the [omniroadmap](https://github.com/grokify/omniroadmap) ecosystem) — no Aha API traffic
- 💾 **Local SQLite sync** - Offline queries and fast local caching
- 🔗 **Neo4j integration** - Graph analytics and relationship queries
- 🌐 **Browser automation** - Strategic template creation via headless Chrome

## Installation

```bash
# Install AQL CLI
go install github.com/grokify/aha-studio/cmd/aha-studio@latest

# Install MCP Server
go install github.com/grokify/aha-studio/cmd/aha-mcp-server@latest
```

## Configuration

Set the following environment variables:

```bash
export AHA_SUBDOMAIN=mycompany    # Required: your Aha.io subdomain
export AHA_API_KEY=xxx            # Required: your Aha.io API key
export AHA_DEFAULT_PRODUCT=PROD   # Optional: default product for queries
```

For Neo4j graph features (optional):

```bash
export NEO4J_URI=bolt://localhost:7687
export NEO4J_USERNAME=neo4j
export NEO4J_PASSWORD=password
```

For browser automation (optional):

```bash
export AHA_EMAIL=user@example.com
export AHA_PASSWORD=secret
```

## Quick Start

### AQL CLI

```bash
# Basic query
aha-studio query "FROM features LIMIT 10"

# Query with filter
aha-studio query "FROM ideas WHERE status = 'Shipped' LIMIT 5"

# Interactive shell
aha-studio shell

# Sync data to SQLite for offline queries
aha-studio sync --product PROD

# Export features to Excel ordered by rank
aha-studio query -o xlsx -f features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace FROM features ORDER BY position ASC"
```

### MCP Server (Claude Desktop)

Add to your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "aha": {
      "command": "aha-mcp-server",
      "env": {
        "AHA_SUBDOMAIN": "mycompany",
        "AHA_API_KEY": "your-api-key"
      }
    }
  }
}
```

### CLI Tool Invocation

Every MCP tool can also be called directly from the shell via `aha-mcp-server tool`. This is useful for AI agent sessions that can't register a new or updated MCP server mid-session — the agent shells out instead of restarting.

```bash
# List all available tools
aha-mcp-server tool list

# Print the JSON Schema for a tool's parameters
aha-mcp-server tool schema list_initiative_features

# Call a tool with JSON parameters (as an argument or via stdin)
aha-mcp-server tool call get_feature '{"reference":"FEAT-123"}'
echo '{"initiative_id":"PROD-S-34"}' | aha-mcp-server tool call list_initiative_features
```

## AQL Syntax

AQL provides a SQL-like interface for querying Aha.io data:

```sql
-- Basic query
FROM features LIMIT 10

-- Filter by status
FROM ideas WHERE status = 'In Progress'

-- Order by field
FROM releases ORDER BY release_date DESC LIMIT 5

-- Select specific fields
SELECT name, status, created_at FROM features WHERE tag = 'v2'

-- Aggregate queries
SELECT status, COUNT(*) as count FROM features GROUP BY status
```

### Supported Entities

| Entity | Description |
|--------|-------------|
| `features` | Product features |
| `ideas` | Customer ideas |
| `releases` | Product releases |
| `epics` | Feature epics |
| `initiatives` | Strategic initiatives |
| `goals` | Product goals |
| `users` | Workspace users |
| `products` | Products/workspaces |
| `comments` | Entity comments |
| `requirements` | Feature requirements |
| `tags` | Entity tags |

## MCP Tools

The MCP server provides 87 tools organized by category.

**Backend Legend:** Tools marked with 🌐 require browser credentials. Tools marked with 🔗 require Neo4j.

### Query Tools (Aha API)

| Tool | Description |
|------|-------------|
| `query` | Execute AQL queries |
| `describe_aql` | Get AQL syntax help |

### Get Tools (Aha API)

| Tool | Description |
|------|-------------|
| `get_feature` | Get feature by reference |
| `get_idea` | Get idea by reference |
| `get_release` | Get release by reference |
| `get_initiative` | Get initiative by reference |
| `get_epic` | Get epic by ID |
| `get_goal` | Get goal by ID |
| `get_comment` | Get comment by ID |
| `get_requirement` | Get requirement by ID |
| `get_user` | Get user by ID |
| `get_current_user` | Get current authenticated user |
| `get_key_result` | Get key result by ID |
| `get_persona` | Get persona by ID |
| `get_team` | Get team by ID |
| `get_workflow` | Get workflow by ID |
| `get_strategic_model` | Get strategic model by ID |
| `get_initiative_with_features` | Get an initiative with its linked features in one call |

### List Tools (Aha API)

| Tool | Description |
|------|-------------|
| `list_ideas` | List ideas with filters |
| `list_products` | List all products |
| `list_features` | List features with filters |
| `list_release_features` | List features for a release |
| `list_epics` | List epics with filters |
| `list_product_epics` | List epics for a product |
| `list_goals` | List goals with filters |
| `list_product_goals` | List goals for a product |
| `list_initiatives` | List initiatives with filters |
| `list_product_initiatives` | List initiatives for a product |
| `list_initiative_features` | List features linked to a specific initiative |
| `list_initiatives_by_tag` | List initiatives filtered by tag value from custom fields |
| `list_feature_requirements` | List requirements for a feature |
| `list_users` | List workspace users |
| `list_workflow_statuses` | List workflow statuses |
| `list_releases` | List releases for product |
| `list_custom_fields` | List custom field definitions |
| `list_custom_field_options` | List options for select custom fields |
| `list_strategic_models` | List strategic models |
| `list_product_strategic_models` | List strategic models for a product |
| `search_documents` | Search documents via GraphQL |

### Create Tools (Aha API)

| Tool | Description |
|------|-------------|
| `create_feature` | Create a new feature in a release |
| `create_epic` | Create a new epic in a release |
| `create_goal` | Create a new goal in a product |
| `create_initiative` | Create a new initiative in a product |
| `create_requirement` | Create a new requirement for a feature |
| `create_product` | Create a new product |
| `create_strategic_model` | Create a new strategic model |

### Update Tools (Aha API)

| Tool | Description |
|------|-------------|
| `update_feature` | Update feature fields |
| `update_epic` | Update epic fields |
| `update_goal` | Update goal fields |
| `update_initiative` | Update initiative fields |
| `update_requirement` | Update requirement fields |
| `update_release` | Update release fields, including `description` (Aha's release theme) |
| `update_idea` | Update idea fields |
| `update_product` | Update product fields |
| `update_strategic_model` | Update strategic model fields |
| `update_comment` | Update comment body |
| `change_feature_status` | Change feature workflow status |
| `assign_feature_release` | Assign feature to release |
| `assign_user_to_feature` | Assign user to feature |

### Comment Tools (Aha API)

| Tool | Description |
|------|-------------|
| `add_feature_comment` | Add comment to feature |
| `add_idea_comment` | Add comment to idea |
| `list_feature_comments` | List comments for a feature |
| `list_idea_comments` | List comments for an idea |
| `list_epic_comments` | List comments for an epic |

### Delete Tools (Aha API)

| Tool | Description |
|------|-------------|
| `delete_requirement` | Delete a requirement |
| `delete_comment` | Delete a comment |

### Template Tools

| Tool | Description |
|------|-------------|
| `list_predefined_templates` | List strategic templates |
| 🌐 `browser_create_template` | Create template via browser automation |

### Sync Tools (Local SQLite Cache)

| Tool | Description |
|------|-------------|
| `sync_data` | Sync Aha data to the local cache (optionally `detailed` for custom fields); requires `AHA_DB_PATH` |

### Graph Tools (Neo4j) 🔗

| Tool | Description |
|------|-------------|
| `graph_sync` | Sync Aha data to Neo4j |
| `graph_query` | Execute Cypher query |
| `graph_find_path` | Find path between entities |
| `graph_search` | Full-text search |
| `graph_initiative_impact` | Initiative impact analysis |
| `graph_release_deps` | Release dependency analysis |

## Development

### Building

```bash
go build ./cmd/aha-studio
go build ./cmd/aha-mcp-server
```

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
