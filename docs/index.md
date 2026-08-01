# Aha Studio

CLI tools for [Aha!](https://www.aha.io/) product management with AQL (Aha Query Language), MCP server integration, local SQLite sync, and Neo4j graph analytics.

## Overview

Aha Studio provides two command-line tools:

| Binary | Purpose |
|--------|---------|
| `aha-studio` | AQL query CLI with SQLite sync and interactive shell |
| `aha-mcp-server` | MCP server (81 tools) for Claude Desktop and AI assistants |

## Features

- **AQL (Aha Query Language)** - SQL-like syntax for querying Aha.io data
- **81 MCP Tools** - Features, Ideas, Releases, Initiatives, Graph queries, analytics, and more
- **HTTP API** - REST endpoints with OpenAPI spec and Prometheus metrics
- **Local SQLite sync** - Offline queries and fast local caching
- **Neo4j integration** - Graph analytics and relationship queries
- **Browser automation** - Strategic template creation via headless Chrome
- **OmniSignal provider** - Emit signals from Aha ideas to ProductContext pipeline

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io/) is an open standard that enables AI assistants to securely connect to external data sources and tools. The `aha-mcp-server` acts as a bridge between AI assistants (like Claude) and your Aha! workspace.

## Available Tools

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

### List & Search Tools

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

## Quick Start

```bash
# Install
go install github.com/grokify/aha-studio/cmd/aha-studio@latest
go install github.com/grokify/aha-studio/cmd/aha-mcp-server@latest

# Configure credentials
export AHA_SUBDOMAIN="your_subdomain"
export AHA_API_KEY="your_api_key"

# Run AQL query
aha-studio query "FROM features LIMIT 10"

# Run MCP server
aha-mcp-server
```

## Next Steps

- [Installation](getting-started/installation.md) - Install the tools
- [Setup](getting-started/setup.md) - Configure your credentials
- [Quick Start](getting-started/quickstart.md) - Start using the tools
- [AQL Syntax](aql/syntax.md) - Learn the query language
- [Tools Reference](tools/overview.md) - Detailed tool documentation
