# Aha Studio Product Requirements

## Product Summary

Aha Studio provides a query language (AQL), MCP server, and analytics platform for Aha.io product management data. It enables product managers, engineers, and AI agents to query, analyze, and manage Aha.io data through SQL-like syntax, programmatic APIs, and natural language via MCP.

## Personas

| Persona | Description | Primary Use |
|---------|-------------|-------------|
| Product Manager | Uses Aha.io daily for roadmap and idea management | AQL for ad-hoc queries, reports, and bulk operations |
| Engineering Lead | Builds integrations and automation around product data | Library mode, HTTP API, MCP server for AI workflows |
| AI Agent | Claude or other LLM assistants via MCP | Natural language product management through 40+ MCP tools |
| Data Analyst | Analyzes product signals and trends | SQLite offline queries, Excel export, graph analytics |

## Use Cases

### Query and Reporting

- Execute SQL-like queries against Aha.io data (features, ideas, releases, epics, goals, initiatives, requirements)
- Aggregate and group data (COUNT, SUM, AVG, MIN, MAX with GROUP BY/HAVING)
- Join related entities (features with releases, ideas with features)
- Export results in multiple formats (table, JSON, CSV, YAML, Markdown, HTML, Excel)

### AI-Assisted Product Management

- AI agents query and manage Aha.io data via MCP tools
- Natural language to AQL translation
- Full CRUD operations on features, ideas, releases, epics, goals, initiatives, requirements, and comments
- Workflow status management and user assignment
- Strategic model management

### Offline Analysis

- Sync Aha.io data to local SQLite database
- Full-text search across all entities
- Incremental sync with change tracking
- Hybrid query modes (live API, offline cache, or prefer-cache)

### Graph Analytics

- Feature dependency analysis via Neo4j
- Release dependency mapping
- Initiative impact assessment
- Cross-entity relationship exploration

### OmniSignal Integration

- Map Aha Ideas to OmniSignal enhancement signals
- Normalize Aha-specific fields (votes, categories, custom fields) into vendor-neutral signal metrics
- Enable frustration scoring (votes x age) across Aha and non-Aha sources

## Functional Requirements

### AQL Engine

- Full SELECT/FROM/WHERE/ORDER BY/LIMIT/GROUP BY/HAVING/JOIN syntax
- INSERT/UPDATE/DELETE mutations with dry-run and confirmation
- Subqueries (scalar and list)
- DISTINCT, aliases, and aggregate functions
- Custom field queries via `custom.fieldname` syntax
- Tags as a derived entity with usage counts

### MCP Server

- 40+ tools covering all major Aha.io entity types
- Lookup/reference tools for resolving names to IDs
- Comment management across features, ideas, and epics
- Custom field definitions and options listing
- OmniSkill-based skill registration with stdio and HTTP transport

### Interactive Experience

- REPL with tab completion, syntax highlighting, and query history
- Saved queries in configuration file
- Configuration profiles for multiple Aha.io accounts

### HTTP API

- REST endpoints for AQL queries, sync, filters, search, and graph analytics
- CORS and API key authentication middleware
- Query mode selection per request
- Background sync scheduling

### Data Management

- SQLite sync with schema migrations
- FTS5 full-text search
- In-memory LRU cache with TTL
- Neo4j graph sync and Cypher query support

## Non-Functional Requirements

- Pure Go SQLite driver (no CGo dependency)
- Concurrent API pagination and JOIN fetches
- Thread-safe cache operations
- CLI commands follow Cobra conventions
- Go module at `github.com/grokify/aha-studio`
