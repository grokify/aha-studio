# Aha Studio

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
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
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/aha-studio
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/aha-studio
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
| `aha-mcp-server` | MCP server (34 tools) for Claude Desktop and AI assistants |

## Features

- 🔍 **AQL (Aha Query Language)** - SQL-like syntax for querying Aha.io data
- 🛠️ **34 MCP Tools** - Features, Ideas, Releases, Initiatives, Graph queries, and more
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

The MCP server provides 34 tools organized by category:

### Query Tools

| Tool | Description |
|------|-------------|
| `query` | Execute AQL queries |
| `describe_aql` | Get AQL syntax help |

### Get Tools

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
| `get_key_result` | Get key result by ID |
| `get_persona` | Get persona by ID |
| `get_team` | Get team by ID |
| `get_workflow` | Get workflow by ID |

### List Tools

| Tool | Description |
|------|-------------|
| `list_ideas` | List ideas with filters |
| `list_products` | List all products |
| `list_workflow_statuses` | List workflow statuses |
| `list_releases` | List releases for product |
| `search_documents` | Search documents via GraphQL |

### Write Tools

| Tool | Description |
|------|-------------|
| `create_feature` | Create a new feature |
| `change_feature_status` | Change feature workflow status |
| `assign_feature_release` | Assign feature to release |
| `assign_user_to_feature` | Assign user to feature |
| `add_feature_comment` | Add comment to feature |
| `add_idea_comment` | Add comment to idea |

### Browser Tools

| Tool | Description |
|------|-------------|
| `list_predefined_templates` | List strategic templates |
| `browser_create_template` | Create template via browser |

### Graph Tools (Neo4j)

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
