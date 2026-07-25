# Aha Studio Architecture

## Overview

Aha Studio is a Go application providing AQL (Aha Query Language), MCP server integration, local SQLite sync, Neo4j graph analytics, and an HTTP API for Aha.io product management data.

## System Architecture

```text
                    ┌──────────────────────────────┐
                    │         Entry Points          │
                    │                               │
                    │  CLI (cobra)    MCP (stdio)   │
                    │  REPL           HTTP API      │
                    └──────────┬───────────────────┘
                               │
                    ┌──────────▼───────────────────┐
                    │       Studio (Facade)         │
                    │       studio/studio.go        │
                    └──────────┬───────────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                     │
 ┌────────▼────────┐ ┌────────▼────────┐  ┌────────▼────────┐
 │  AQL Pipeline   │ │   MCP Server    │  │   HTTP Server   │
 │                 │ │   mcp/          │  │   httpserver/   │
 │ lexer → parser  │ │                 │  │                 │
 │   → ast         │ │ OmniSkill skill │  │ REST endpoints  │
 │   → validator   │ │ 40+ tools       │  │ CORS/Auth       │
 │   → planner     │ │ Resources       │  │ Query modes     │
 │   → executor    │ │                 │  │                 │
 └────────┬────────┘ └────────┬────────┘  └────────┬────────┘
          │                   │                     │
          └────────────┬──────┴─────────────────────┘
                       │
          ┌────────────▼────────────────────────────┐
          │              Data Layer                  │
          │                                          │
          │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │
          │  │ Aha API  │ │ SQLite   │ │ Neo4j    │ │
          │  │ (live)   │ │ (cache)  │ │ (graph)  │ │
          │  │ aha-go   │ │ sync/    │ │ graph/   │ │
          │  └──────────┘ └──────────┘ └──────────┘ │
          └─────────────────────────────────────────┘
```

## AQL Pipeline

The query engine follows a classic compiler pipeline:

```text
AQL string
    │
    ▼
Lexer (aql/lexer/)
    │  Tokenization
    ▼
Parser (aql/parser/)
    │  Recursive descent, operator precedence
    ▼
AST (aql/ast/)
    │  SELECT, FROM, WHERE, JOIN, GROUP BY, HAVING,
    │  ORDER BY, LIMIT, INSERT, UPDATE, DELETE
    ▼
Validator (aql/validator/)
    │  Semantic validation against entity schema
    ▼
Planner (planner/)
    │  Filter pushdown, custom field detection,
    │  JOIN strategy, aggregation planning
    ▼
Executor (executor/)
    │  API calls, client-side filtering, aggregation,
    │  JOIN resolution, mutation execution
    ▼
Result (result/)
    │  Records, metadata, formatting
    ▼
Output (table / JSON / CSV / YAML / Markdown / HTML / XLSX)
```

## MCP Server

Built on the OmniSkill framework (`github.com/plexusone/omniskill`), which provides:

- Skill/Tool abstraction over the MCP protocol
- Transport options: stdio (Claude Desktop), HTTP/SSE
- Direct library-mode invocation without JSON-RPC overhead

The MCP server exposes 40+ tools across entity types (features, ideas, releases, epics, goals, initiatives, requirements, products, strategic models, comments) with full CRUD operations plus lookup/reference helpers.

### Key files

| File | Responsibility |
|------|----------------|
| `mcp/skill.go` | AhaSkill definition, tool registration |
| `mcp/handlers.go` | Tool handler implementations |
| `mcp/config.go` | MCP server configuration |

## Data Layer

### Aha API (aha-go)

Primary data source. All live queries go through `github.com/grokify/aha-go`, which wraps the Aha.io REST API.

### SQLite (sync/)

Local cache for offline and hybrid query modes.

- Full entity sync with incremental updates via `updated_since`
- Schema migrations for evolving entity models
- FTS5 full-text search across entities
- Saved filters persistence
- Three query modes: `api` (live only), `offline` (cache only), `prefer-cache` (cache first, API fallback)

### Neo4j (graph/)

Graph analytics for entity relationships.

- Feature dependency graphs
- Release dependency analysis
- Initiative impact mapping
- Product overview aggregation
- Raw Cypher query support

### In-Memory Cache (cache/)

LRU cache with TTL for API response caching. Thread-safe, configurable size and expiration.

## HTTP API

REST server (`httpserver/`) exposing AQL queries, sync management, saved filters, full-text search, and Neo4j graph endpoints. See ROADMAP.md Phase 11 for the full endpoint list.

## Ecosystem Position

```text
Grokify Org
─────────────────────────────────
aha-go          Aha.io API client
aha-studio      AQL + MCP + sync + graph

         │
         │ OmniSignal adapter
         │ (Aha Ideas → enhancement signals)
         ▼

PlexusOne Org
─────────────────────────────────
omniskill       MCP skill framework
omnisignal      Signal normalization

         │
         │ references
         ▼

ProductBuildersHQ Org
─────────────────────────────────
market-spec     Market intelligence entities
organization-spec  Customer/prospect entities
prism-roadmap   Strategic planning
```

Aha Studio bridges Aha.io into the broader ProductContext ecosystem. The OmniSignal adapter converts Aha-specific concepts (Ideas, Idea Portals, custom fields) into vendor-neutral signals that OmniSignal can aggregate with evidence from other sources.

## Dependencies

| Module | Purpose |
|--------|---------|
| `github.com/grokify/aha-go` | Aha.io REST API client |
| `github.com/plexusone/omniskill` | MCP skill infrastructure |
| `github.com/modelcontextprotocol/go-sdk` | Official MCP Go SDK |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/c-bata/go-prompt` | Interactive REPL |
| `github.com/xuri/excelize/v2` | Excel export |
| `modernc.org/sqlite` | Pure Go SQLite driver |
| `github.com/neo4j/neo4j-go-driver` | Neo4j graph database |
