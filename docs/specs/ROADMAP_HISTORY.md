# Aha Studio Roadmap

This document outlines the planned phases and features for Aha Studio development.

## Overview

Aha Studio provides AQL (Aha Query Language) - a JQL-like query layer for Aha.io. The roadmap is organized into phases, each building upon the previous to create a comprehensive query and management tool for Aha.io data.

## Phase 1: Core AQL Engine (Completed)

**Status:** ✅ Complete

The foundation of Aha Studio with basic query capabilities.

### Features

| Feature | Status | Description |
|---------|--------|-------------|
| Lexer | ✅ | Tokenization of AQL queries |
| Parser | ✅ | Recursive descent parser with operator precedence |
| AST | ✅ | Abstract syntax tree representation |
| Validator | ✅ | Semantic validation against schema |
| Planner | ✅ | Query planning with filter pushdown |
| Executor | ✅ | Query execution against Aha API |
| CLI | ✅ | Command-line interface with Cobra |

### Supported Syntax

```sql
FROM <entity>
[WHERE <conditions>]
[ORDER BY <field> [ASC|DESC]]
[LIMIT <n>]
```

### Entities

- `features` - Product features
- `ideas` - Customer ideas
- `releases` - Product releases
- `initiatives` - Strategic initiatives

### Operators

- Comparison: `=`, `!=`, `<`, `<=`, `>`, `>=`
- Set: `IN`, `NOT IN`
- String: `CONTAINS`, `LIKE`
- Null: `IS NULL`, `IS NOT NULL`
- Logical: `AND`, `OR`, `NOT`

### Functions

- `now()` - Current timestamp
- `duration("30d")` - Duration literal

### Output Formats

- Table (default)
- JSON
- CSV

---

## Phase 2: Extended Query Features

**Status:** ✅ Complete

Add SQL-like analytical capabilities to AQL.

### Aggregations

Support for aggregate functions in SELECT clauses.

```sql
-- Count all features
SELECT COUNT(*) FROM features

-- Sum of votes by status
SELECT status, SUM(votes) FROM ideas GROUP BY status

-- Average effort per initiative
SELECT AVG(effort) FROM initiatives WHERE status = "Active"
```

| Function | Description |
|----------|-------------|
| `COUNT(*)` | Count all records |
| `COUNT(field)` | Count non-null values |
| `SUM(field)` | Sum numeric values |
| `AVG(field)` | Average of numeric values |
| `MIN(field)` | Minimum value |
| `MAX(field)` | Maximum value |

### GROUP BY

Group results by one or more fields.

```sql
-- Ideas by status
SELECT status, COUNT(*) FROM ideas GROUP BY status

-- Features by release and status
SELECT release, status, COUNT(*)
FROM features
GROUP BY release, status
```

### HAVING

Filter grouped results.

```sql
-- Statuses with more than 10 ideas
SELECT status, COUNT(*) as count
FROM ideas
GROUP BY status
HAVING count > 10
```

### JOINs

Join related entities.

```sql
-- Features with their releases
SELECT f.name, r.name as release_name
FROM features f
JOIN releases r ON f.release_id = r.id

-- Ideas linked to features
SELECT i.name, f.name as feature_name
FROM ideas i
LEFT JOIN features f ON i.feature_id = f.id
```

| Join Type | Description |
|-----------|-------------|
| `JOIN` / `INNER JOIN` | Only matching records |
| `LEFT JOIN` | All left + matching right |
| `RIGHT JOIN` | All right + matching left |

### Aliases

Column and table aliases for cleaner queries.

```sql
-- Column aliases
SELECT name AS title, votes AS score FROM ideas

-- Table aliases
SELECT f.name, r.name
FROM features AS f
JOIN releases AS r ON f.release_id = r.id
```

### DISTINCT

Remove duplicate rows.

```sql
-- Unique statuses
SELECT DISTINCT status FROM features

-- Unique status/release combinations
SELECT DISTINCT status, release FROM features
```

### Subqueries

**Status:** ✅ Complete

Nested queries for complex filtering.

```sql
-- Ideas with above-average votes
SELECT * FROM ideas
WHERE votes > (SELECT AVG(votes) FROM ideas)

-- Features in active releases
SELECT * FROM features
WHERE release_id IN (SELECT id FROM releases WHERE released = false)

-- Features NOT in closed releases
SELECT * FROM features
WHERE release_id NOT IN (SELECT id FROM releases WHERE released = true)
```

| Subquery Type | Example | Description |
|---------------|---------|-------------|
| Scalar | `votes > (SELECT AVG(votes) ...)` | Returns single value for comparison |
| List (IN) | `id IN (SELECT id ...)` | Returns list for IN clause |
| List (NOT IN) | `id NOT IN (SELECT id ...)` | Returns list for NOT IN clause |

**Implementation Notes:**

- Subqueries are executed first, then their results are used in the main query
- Scalar subqueries should return a single aggregate value (COUNT, SUM, AVG, MIN, MAX)
- List subqueries should SELECT a single column
- Nested subqueries (subquery within subquery) are supported
- Subqueries can include WHERE, GROUP BY, and HAVING clauses

### Implementation Notes

- Aggregations require fetching all matching records ✅
- JOINs use multiple API calls and client-side joining ✅
- Consider caching for frequently joined entities
- Subqueries use multi-pass execution (subqueries first, then main query) ✅

---

## Phase 3: Additional Entities

**Status:** ✅ Complete

Expand entity coverage to include all major Aha.io objects.

### Entity Status

| Entity | Status | Description |
|--------|--------|-------------|
| `products` | ✅ Complete | Workspaces (Get/List) |
| `users` | ✅ Complete | Account users (Get/List) |
| `goals` | ✅ Complete | Strategic goals (Get/List/Create/Update) |
| `epics` | ✅ Complete | Feature epics (Get/List/Create/Update) |
| `requirements` | ✅ Complete | Feature requirements (Get/List/Create/Update/Delete) |
| `comments` | ✅ Complete | Comments on entities (Get/List/Create/Update/Delete) |
| `tags` | ⚠️ Partial | Available as properties on entities |
| `workflows` | ⚠️ Partial | Available via workflow_status |
| `custom_fields` | ⚠️ Partial | Available as properties on features |

### Example Queries

```sql
-- All strategic goals
FROM goals WHERE status = "Active" ORDER BY progress DESC

-- Epics in a release
FROM epics WHERE release = "REL-1" ORDER BY position

-- Requirements for a feature
FROM requirements WHERE feature_id = "FEAT-123"

-- Comments on a feature
FROM comments WHERE feature_id = "FEAT-123"

-- Comments on an idea
FROM comments WHERE idea_id = "IDEA-456"

-- Comments in a product
FROM comments WHERE product_id = "PROD"

-- Active users
FROM users WHERE role IN ("product_owner", "contributor")

-- Products with ideas enabled
FROM products WHERE has_ideas = true
```

### Schema Updates

Each new entity requires:

1. OpenAPI spec additions in `aha-go`
2. Wrapper types and methods in `aha-go`
3. Schema definitions in `aha-studio/schema/entities.go`
4. Executor support in `aha-studio/executor/`

---

## Phase 4: Mutations (Write Operations)

**Status:** ✅ Complete

Add support for creating, updating, and deleting records.

### INSERT

Create new records.

```sql
-- Create a feature
INSERT INTO features (name, description, release, status)
VALUES ("New Feature", "Description here", "REL-1", "In Progress")

-- Create an idea
INSERT INTO ideas (name, description)
VALUES ("Customer Request", "They want this feature")

-- Create with subquery
INSERT INTO features (name, release)
SELECT name, "REL-2" FROM features WHERE release = "REL-1" AND status = "Backlog"
```

### UPDATE

Modify existing records.

```sql
-- Update single record
UPDATE features
SET status = "Done", due_date = "2024-03-01"
WHERE reference_num = "FEAT-123"

-- Bulk update
UPDATE features
SET status = "In Progress"
WHERE release = "REL-1" AND status = "Ready"

-- Update with expression
UPDATE ideas
SET name = CONCAT("[REVIEWED] ", name)
WHERE reviewed = true AND name NOT LIKE "[REVIEWED]%"
```

### DELETE

Remove records (where supported by API).

```sql
-- Delete spam ideas
DELETE FROM ideas
WHERE spam = true AND created_at < now() - duration("90d")

-- Delete with confirmation
DELETE FROM features WHERE reference_num = "FEAT-123"
-- CLI will prompt: "Delete 1 feature? [y/N]"
```

### Safety Features

| Feature | Description |
|---------|-------------|
| Dry run | `--dry-run` flag shows what would change |
| Confirmation | Prompt before destructive operations |
| Transaction log | Log all mutations for audit |
| Undo support | Generate reverse operations |

### CLI Examples

```bash
# Create a feature
ahastudio exec "INSERT INTO features (name) VALUES ('New Feature')" --release REL-1

# Update with dry run
ahastudio exec --dry-run "UPDATE features SET status = 'Done' WHERE release = 'REL-1'"

# Delete with confirmation
ahastudio exec "DELETE FROM ideas WHERE spam = true" --confirm
```

---

## Phase 5: Interactive Mode & Developer Experience

**Status:** ✅ Complete

Improve the developer experience with interactive features.

### REPL Mode

Interactive query shell.

```bash
$ ahastudio shell
Aha Studio v0.2.0 - Connected to mycompany.aha.io
Type 'help' for commands, 'exit' to quit.

aql> FROM features LIMIT 5
┌─────────────┬──────────────────┬─────────────┐
│ REFERENCE   │ NAME             │ STATUS      │
├─────────────┼──────────────────┼─────────────┤
│ FEAT-1      │ User Login       │ Done        │
│ FEAT-2      │ Dashboard        │ In Progress │
...

aql> .output json
Output format set to json.

aql> FROM ideas WHERE votes > 10
{
  "entity": "ideas",
  "count": 3,
  "records": [...]
}

aql> .history
1: FROM features LIMIT 5
2: FROM ideas WHERE votes > 10

aql> exit
Goodbye!
```

### REPL Commands

| Command | Description |
|---------|-------------|
| `.help` | Show help |
| `.output <format>` | Set output format (table/json/csv) |
| `.product <id>` | Set default product context |
| `.history` | Show query history |
| `.save <name>` | Save last query |
| `.run <name>` | Run saved query |
| `.clear` | Clear screen |
| `.exit` | Exit shell |

### Tab Completion

Context-aware completion.

```
aql> FROM fea<TAB>
features

aql> FROM features WHERE sta<TAB>
status    start_date

aql> FROM features WHERE status = "<TAB>
"In Progress"  "Done"  "Ready"  "Backlog"
```

### Query History

Persistent history with search.

```bash
# History stored in ~/.ahastudio/history
$ ahastudio shell
aql> <UP>              # Previous query
aql> <CTRL+R>          # Search history
(reverse-i-search): feat
```

### Syntax Highlighting

Colored output for better readability.

- Keywords: blue (`FROM`, `WHERE`, `ORDER BY`)
- Strings: green (`"In Progress"`)
- Numbers: cyan (`10`, `30d`)
- Operators: yellow (`=`, `AND`, `OR`)
- Fields: white
- Errors: red

### Configuration File

`~/.ahastudio.yaml` for persistent settings.

```yaml
# Default settings
defaults:
  output: table
  product: PLATFORM
  per_page: 50

# Connection profiles
profiles:
  work:
    subdomain: mycompany
    api_key_env: AHA_API_KEY_WORK
  personal:
    subdomain: personal
    api_key_env: AHA_API_KEY_PERSONAL

# Saved queries
queries:
  active-features: |
    FROM features
    WHERE status IN ("In Progress", "Ready")
    ORDER BY updated_at DESC
  top-ideas: |
    FROM ideas
    ORDER BY votes DESC
    LIMIT 10
```

### Query Library

Save and reuse named queries.

```bash
# Save a query
ahastudio save "active-features" "FROM features WHERE status = 'In Progress'"

# Run saved query
ahastudio run active-features

# List saved queries
ahastudio list-queries

# Delete saved query
ahastudio delete-query active-features
```

---

## Phase 6: Export & Reporting

**Status:** ✅ Complete (Core Formats)

Advanced export and reporting capabilities.

### Additional Output Formats

| Format | Extension | Status | Description |
|--------|-----------|--------|-------------|
| Markdown | `.md` | ✅ | Markdown table |
| YAML | `.yaml` | ✅ | YAML structure |
| HTML | `.html` | ✅ | Styled HTML report |
| Excel | `.xlsx` | 🔲 | Excel workbook (requires excelize dependency) |
| PDF | `.pdf` | 🔲 | PDF document (requires gofpdf dependency) |

### Excel Export

```bash
# Export to Excel
ahastudio query -o xlsx -f report.xlsx "FROM features"

# Multiple sheets
ahastudio query -o xlsx -f report.xlsx \
  --sheet "Features:FROM features" \
  --sheet "Ideas:FROM ideas"
```

### Markdown Export

```bash
ahastudio query -o markdown "FROM features LIMIT 5"

# Output:
| Reference | Name | Status |
|-----------|------|--------|
| FEAT-1 | User Login | Done |
| FEAT-2 | Dashboard | In Progress |
```

### HTML Reports

```bash
# Basic HTML table
ahastudio query -o html "FROM features" > report.html

# Styled report with template
ahastudio query -o html --template status-report.html "FROM features"
```

### Custom Templates

Go template support for custom output.

```bash
# Template file: feature-list.tmpl
{{range .Records}}
## {{.reference_num}}: {{.name}}
Status: {{.status}}
{{end}}

# Use template
ahastudio query --template feature-list.tmpl "FROM features"
```

### Scheduled Queries

Run queries on a schedule (requires daemon or cron).

```yaml
# ~/.ahastudio/schedules.yaml
schedules:
  daily-summary:
    cron: "0 9 * * *"  # 9 AM daily
    query: "FROM features WHERE updated_at >= now() - duration('24h')"
    output: email
    recipients:
      - team@company.com

  weekly-report:
    cron: "0 8 * * MON"  # Monday 8 AM
    query: "FROM ideas ORDER BY votes DESC LIMIT 20"
    output: file
    path: /reports/weekly-ideas-{{date}}.xlsx
```

### Email Reports

```bash
# Send query results via email
ahastudio query -o email \
  --to team@company.com \
  --subject "Weekly Feature Report" \
  "FROM features WHERE status = 'Done' AND updated_at >= now() - duration('7d')"
```

---

## Phase 7: MCP Server Integration

**Status:** ✅ Complete

Model Context Protocol server for AI assistant integration.

### Overview

Expose Aha Studio capabilities as MCP tools, allowing AI assistants like Claude to query and manage Aha.io data. Built using:

- **OmniSkill Framework**: `github.com/plexusone/omniskill` - Unified skill infrastructure for AI agents
- **Official Go MCP SDK**: `github.com/modelcontextprotocol/go-sdk` - Model Context Protocol implementation

### Architecture

Aha Studio integrates with OmniSkill's skill system:

```
┌─────────────────────────────────────────────────────────────┐
│                      OmniSkill Runtime                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │ AhaSkill    │  │ Other Skills│  │ Registry            │ │
│  │ ├─ query    │  │ ├─ ...      │  │ ├─ aha.query        │ │
│  │ ├─ get      │  │ └─ ...      │  │ ├─ aha.get_feature  │ │
│  │ └─ create   │  │             │  │ └─ ...              │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  Transport Layer: stdio | HTTP/SSE | WebSocket              │
└─────────────────────────────────────────────────────────────┘
```

### Skill Implementation

Create an Aha skill following OmniSkill patterns:

```go
// mcp/skill.go
package mcp

import (
    "context"

    "github.com/plexusone/omniskill/skill"
    "github.com/grokify/aha-studio"
)

type AhaSkill struct {
    skill.BaseSkill
    studio *studio.Studio
}

func NewAhaSkill(s *studio.Studio) *AhaSkill {
    return &AhaSkill{studio: s}
}

func (s *AhaSkill) Name() string        { return "aha" }
func (s *AhaSkill) Description() string { return "Query and manage Aha.io data using AQL" }

func (s *AhaSkill) Tools() []skill.Tool {
    return []skill.Tool{
        skill.NewTool("query", "Execute an AQL query against Aha.io",
            map[string]skill.Parameter{
                "query":   {Type: "string", Required: true, Description: "AQL query string"},
                "product": {Type: "string", Required: false, Description: "Product ID context"},
            },
            s.queryHandler),
        skill.NewTool("get_feature", "Get a feature by reference number",
            map[string]skill.Parameter{
                "reference": {Type: "string", Required: true, Description: "Feature reference (e.g., FEAT-123)"},
            },
            s.getFeatureHandler),
        skill.NewTool("get_idea", "Get an idea by reference number",
            map[string]skill.Parameter{
                "reference": {Type: "string", Required: true, Description: "Idea reference (e.g., IDEA-456)"},
            },
            s.getIdeaHandler),
    }
}

func (s *AhaSkill) queryHandler(ctx context.Context, params map[string]any) (any, error) {
    query := params["query"].(string)
    productID, _ := params["product"].(string)

    result, err := s.studio.Query(ctx, query, studio.WithProduct(productID))
    if err != nil {
        return nil, err
    }
    return result.Records, nil
}
```

### MCP Server

```go
// cmd/ahastudio/mcp.go
package main

import (
    "context"
    "os"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/plexusone/omniskill/mcp/server"
    ahaMCP "github.com/grokify/aha-studio/mcp"
)

func runMCPServer(ctx context.Context, s *studio.Studio) error {
    rt := server.New(&mcp.Implementation{
        Name:    "aha-studio",
        Version: "0.3.0",
    }, nil)

    // Register Aha skill
    ahaSkill := ahaMCP.NewAhaSkill(s)
    rt.RegisterSkillWithPrefix(ahaSkill)  // Tools: aha_query, aha_get_feature, etc.

    // Start server
    return rt.ServeStdio(ctx)
}
```

### CLI Integration

```bash
# Start MCP server (stdio for Claude Desktop)
ahastudio mcp-server

# Start HTTP server
ahastudio mcp-server --http :8080

# With ngrok tunnel
ahastudio mcp-server --http :8080 --ngrok
```

### MCP Tools

| Tool | Description |
|------|-------------|
| `aha_query` | Execute AQL query |
| `aha_get_feature` | Get feature by reference |
| `aha_get_idea` | Get idea by reference |
| `aha_get_release` | Get release by reference |
| `aha_get_initiative` | Get initiative by reference |
| `aha_list_products` | List all products |
| `aha_create_feature` | Create a new feature |
| `aha_update_feature` | Update a feature |

### Tool Schema Example

```go
skill.NewTool("query", "Execute an AQL query against Aha.io",
    map[string]skill.Parameter{
        "query": {
            Type:        "string",
            Required:    true,
            Description: "AQL query (e.g., 'FROM features WHERE status = \"In Progress\" LIMIT 10')",
        },
        "product": {
            Type:        "string",
            Required:    false,
            Description: "Product ID for context",
        },
        "format": {
            Type:        "string",
            Required:    false,
            Enum:        []any{"json", "table", "csv"},
            Default:     "json",
            Description: "Output format",
        },
    },
    s.queryHandler)
```

### Claude Desktop Integration

```json
// ~/.claude/claude_desktop_config.json
{
  "mcpServers": {
    "aha-studio": {
      "command": "ahastudio",
      "args": ["mcp-server"],
      "env": {
        "AHA_SUBDOMAIN": "mycompany",
        "AHA_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Library Mode (Direct Invocation)

OmniSkill supports direct tool invocation without MCP transport overhead:

```go
// Direct library mode
rt := server.New(impl, nil)
rt.RegisterSkill(ahaMCP.NewAhaSkill(s))

// Call tool directly (no JSON-RPC)
result, err := rt.CallTool(ctx, "aha_query", map[string]any{
    "query": "FROM features WHERE status = 'In Progress' LIMIT 10",
})
```

### Natural Language Interface

AI can translate natural language to AQL:

```
User: "Show me the top voted ideas from this month"

Claude: I'll query for top voted ideas from this month.
[Uses aha_query tool with: FROM ideas WHERE created_at >= now() - duration("30d") ORDER BY votes DESC LIMIT 10]

Here are the top voted ideas from this month:
1. IDEA-456: Dark mode support (45 votes)
2. IDEA-789: Mobile app (38 votes)
...
```

### MCP Resources

Expose Aha data as MCP resources for context:

```go
func (s *AhaSkill) Resources() []skill.Resource {
    return []skill.Resource{
        {
            URI:         "aha://products",
            Name:        "Aha Products",
            Description: "List of all Aha products",
            MimeType:    "application/json",
        },
        {
            URITemplate: "aha://features/{reference}",
            Name:        "Aha Feature",
            Description: "Feature details by reference",
            MimeType:    "application/json",
        },
    }
}
```

### Implementation Files

| File | Description |
|------|-------------|
| `mcp/skill.go` | AhaSkill implementation |
| `mcp/tools.go` | Tool handlers |
| `mcp/resources.go` | Resource handlers |
| `cmd/ahastudio/mcp.go` | CLI mcp-server command |

### Dependencies

```go
require (
    github.com/modelcontextprotocol/go-sdk v0.x.x
    github.com/plexusone/omniskill v0.x.x
)

---

## Phase 8: Performance & Scale

**Status:** ✅ Complete (Core Features)

Optimize for large datasets and improve performance.

### Query Caching

**Status:** ✅ In-memory cache implemented

Cache API responses with configurable TTL.

```go
// cache package provides in-memory caching
cache := cache.New(cache.Options{
    TTL:     5 * time.Minute,
    MaxSize: 100,
})
```

Features:

- In-memory LRU cache with TTL
- Cache key generation from query + product
- Automatic eviction of expired entries
- Thread-safe operations

### Parallel Fetches

Concurrent API calls for better throughput.

```yaml
# ~/.ahastudio.yaml
performance:
  max_concurrent: 5
  page_size: 100
```

- Parallel pagination for large result sets
- Concurrent fetches for JOINs
- Connection pooling

### Local Database Sync

SQLite cache for offline queries and better performance.

```bash
# Initial sync
ahastudio sync --product PLATFORM
# Syncing features... 245 records
# Syncing ideas... 1,203 records
# Syncing releases... 34 records
# Sync complete in 12.3s

# Incremental sync
ahastudio sync --since last
# Syncing changes since 2024-01-15T10:30:00Z
# Updated 12 features, 45 ideas
# Sync complete in 1.2s
```

### Offline Mode

Query cached data without API connection.

```bash
# Use local cache only
ahastudio query --offline "FROM features WHERE status = 'In Progress'"

# Hybrid mode: local first, API fallback
ahastudio query --prefer-cache "FROM features"
```

### Query Optimization

- Predicate pushdown to API where possible
- Lazy loading for JOINs
- Result streaming for large datasets
- Query plan caching

### Metrics & Observability

**Status:** ✅ Implemented

```bash
# Show query statistics
ahastudio query --stats "FROM features"

# Output:
# Query Statistics:
#   Execution time: 234ms
#   Records returned: 245
#   Client filters: 1
#   Pagination: enabled
```

---

## Implementation Priority

Recommended implementation order based on value and dependencies.

| Priority | Phase | Status | Rationale |
|----------|-------|--------|-----------|
| 1 | Phase 1 (Core AQL) | ✅ Complete | Foundation |
| 2 | Phase 5 (REPL/Config) | ✅ Complete | Improves daily developer experience |
| 3 | Phase 2 (Aggregations + Subqueries) | ✅ Complete | High value for analytics use cases |
| 4 | Phase 7 (MCP Server) | ✅ Complete | AI integration is strategic |
| 5 | Phase 4 (Mutations) | ✅ Complete | Write operations |
| 6 | Phase 6 (Export) | ✅ Complete | Reporting capabilities (core formats) |
| 7 | Phase 8 (Performance) | ✅ Complete | Scale optimization (cache + stats) |
| 8 | Phase 3 (More Entities) | ✅ Complete | Broader API coverage |
| 9 | Phase 9 (Extended Features) | ✅ Complete | Excel, SQLite sync, tags, custom fields |
| 10 | Phase 10a (Core Write Tools) | ✅ Complete | Essential MCP write operations |
| 11 | Phase 11 (Web UI & HTTP API) | 🔄 In Progress | HTTP server, Lit component, cache modes |
| 12 | Phase 12 (Feature-Release Queries) | 🔄 In Progress | Query features by release date/name |
| 13 | Phase 10b (Extended Write Tools) | 🔄 Mostly Complete | Full CRUD for features, epics, goals, initiatives, requirements, products, strategic models, and comments |
| 14 | Phase 10c (Analytics Tools) | 🔲 Planned | Statistics and metrics |
| 15 | Phase 13 (DuckDB Migration) | 🔲 Planned | Columnar DB for analytics |
| 16 | Phase 10d (Integrations) | 🔲 Planned | Jira, Confluence, AI workflows |

---

## Phase 9: Extended Features

**Status:** 🔄 In Progress

Additional enhancements for enterprise use cases.

### Excel Export

**Status:** ✅ Complete

Full Excel workbook support with multiple sheets.

```bash
# Export to Excel
ahastudio query -o xlsx -f report.xlsx "FROM features"

# Multiple sheets in one workbook
ahastudio query -o xlsx -f report.xlsx \
  --sheet "Features:FROM features WHERE status = 'In Progress'" \
  --sheet "Ideas:FROM ideas ORDER BY votes DESC LIMIT 20"
```

Features:

- Single query to `.xlsx` file
- Multi-sheet workbooks
- Auto-column width
- Header styling
- Date/number formatting

### SQLite Sync

**Status:** ✅ Complete

Local SQLite database for offline queries and improved performance.

```bash
# Initial sync - downloads all entities
ahastudio sync --product PLATFORM
# Syncing features... 245 records
# Syncing ideas... 1,203 records
# Syncing releases... 34 records
# Sync complete in 12.3s

# Incremental sync - only changes since last sync
ahastudio sync --since last
# Syncing changes since 2024-01-15T10:30:00Z
# Updated 12 features, 45 ideas
# Sync complete in 1.2s

# Query local database (no API calls)
ahastudio query --offline "FROM features WHERE status = 'In Progress'"

# Hybrid mode: local first, API fallback for cache misses
ahastudio query --prefer-cache "FROM features"
```

Features:

- Full entity sync to SQLite
- Incremental sync with `updated_since`
- Offline query mode
- Hybrid cache/API mode
- Schema migrations

### Tags as First-Class Entity

**Status:** ✅ Complete

Query tags as a derived entity aggregated from features and epics.

```sql
-- List all tags with usage counts
FROM tags ORDER BY name

-- Most used tags
FROM tags ORDER BY total_count DESC LIMIT 10

-- Tags used on features
FROM tags WHERE feature_count > 0 ORDER BY feature_count DESC

-- Tags used on epics
FROM tags WHERE epic_count > 0 ORDER BY epic_count DESC
```

Fields:

- `name` - Tag name
- `feature_count` - Number of features using this tag
- `epic_count` - Number of epics using this tag
- `total_count` - Total usage count (features + epics)

Note: Tags are a derived entity since the Aha API doesn't have a dedicated tags endpoint. Tags are extracted from features and epics and aggregated with usage counts.

### Custom Fields Enhancement

**Status:** ✅ Complete

Better support for custom field queries.

```sql
-- Query by custom field
FROM features WHERE custom.priority = "High"

-- Select custom fields
SELECT name, custom.priority, custom.team FROM features

-- Order by custom field
FROM features ORDER BY custom.priority DESC

-- Group by custom field
SELECT custom.team, COUNT(*) FROM features GROUP BY custom.team

-- Also works with goals
FROM goals WHERE custom.category = "Strategic"
```

Features:

- Parser supports `custom.fieldname` syntax (qualifier.name)
- Planner detects custom field references and sets `NeedsCustomFields` flag
- Executor fetches full entity details when custom fields are needed
- Custom fields stored in records with `custom.fieldname` keys
- `Record.Get()` handles custom field lookups
- Works with WHERE, SELECT, ORDER BY, and GROUP BY clauses
- Supported entities: Features, Goals (only entities with custom_fields in Aha API)

---

## Phase 10: MCP Tool Enhancements

**Status:** 🔄 In Progress

Expand MCP tools to provide comprehensive Aha.io management capabilities, enabling AI agents to perform full product management workflows.

### Overview

Based on analysis of alternate MCP implementations, this phase adds write operations, lookup helpers, statistics tools, and integrations to make the MCP server a complete product management assistant.

---

### Feature Management Tools

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `create_feature` | 🔴 High | ✅ | Create a new feature with name, description, release, status |
| `update_feature` | 🔴 High | ✅ | Update feature properties (name, description, status, release, tags, dates) |
| `assign_user_to_feature` | 🔴 High | ✅ | Assign a user to a feature by user ID or email |
| `change_feature_status` | 🔴 High | ✅ | Change feature workflow status (by name or ID) |
| `assign_feature_release` | 🔴 High | ✅ | Assign a feature to a release (by name or ID) |
| `add_feature_comment` | 🔴 High | ✅ | Add a comment to a feature |
| `list_features` | 🟡 Medium | ✅ | List features for a product with filters and pagination |
| `list_features_by_release` | 🟡 Medium | ✅ | List all features in a specific release (`list_release_features`) |
| `get_feature_ideas` | 🟡 Medium | 🔲 | Get ideas promoted to a feature |

#### Example: Create Feature
```json
{
  "tool": "create_feature",
  "params": {
    "product_id": "PROD-1",
    "name": "User Authentication",
    "description": "Implement OAuth2 login flow",
    "release": "Q1-2024",
    "status": "Ready for Development"
  }
}
```

---

### Lookup/Reference Tools

These tools enable AI agents to resolve human-readable names to IDs required by other tools.

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `list_workflow_statuses` | 🔴 High | ✅ | List all workflow statuses for a product (enables status lookup by name) |
| `list_releases` | 🔴 High | ✅ | List all releases for a product (enables release lookup by name) |
| `list_users` | 🟡 Medium | ✅ | List users in the account (enables user lookup by name/email) |
| `list_goals` | 🟡 Medium | ✅ | List goals for a product |
| `list_custom_fields` | 🟡 Medium | ✅ | List custom field definitions for a product or all products |
| `list_custom_field_options` | 🟡 Medium | ✅ | List options for a select/choice custom field |
| `list_idea_categories` | 🟡 Medium | 🔲 | List idea categories with caching |

#### Workflow Status Lookup Example
```
User: "Move feature FEAT-123 to 'In Progress'"

Agent:
1. Calls list_workflow_statuses to find ID for "In Progress"
2. Calls change_feature_status with feature_id and status_id
```

---

### Idea Management Tools

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `create_idea` | 🟡 Medium | 🔲 | Create a new idea |
| `update_idea` | 🟡 Medium | ✅ | Update idea properties (name, description, status, visibility) |
| `delete_idea` | 🟠 Low | 🔲 | Delete an idea |
| `add_idea_comment` | 🔴 High | ✅ | Add an internal comment to an idea |
| `update_comment` | 🟡 Medium | ✅ | Update an existing comment (feature/idea/epic) |
| `delete_comment` | 🟠 Low | ✅ | Delete a comment |
| `list_feature_comments` / `list_idea_comments` / `list_epic_comments` | 🟡 Medium | ✅ | List comments on a feature, idea, or epic |
| `list_ideas_by_category` | 🟡 Medium | 🔲 | List ideas filtered by category |
| `list_ideas_without_release` | 🟠 Low | 🔲 | List ideas in status without a release |

---

### Release Management Tools

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `create_release` | 🟡 Medium | 🔲 | Create a new release |
| `update_release` | 🟡 Medium | ✅ | Update release properties (name, dates, parking_lot) |

---

### Requirement Management Tools

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `list_requirements` | 🟡 Medium | ✅ | List requirements for a feature (`list_feature_requirements`) |
| `create_requirement` | 🟡 Medium | ✅ | Create a requirement on a feature |
| `update_requirement` | 🟡 Medium | ✅ | Update a requirement |
| `delete_requirement` | 🟠 Low | ✅ | Delete a requirement |

---

### Goal Management Tools

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `list_goals` / `list_product_goals` | 🟡 Medium | ✅ | List goals, optionally scoped to a product |
| `create_goal` | 🟡 Medium | ✅ | Create a new goal in a product |
| `update_goal` | 🟡 Medium | ✅ | Update goal properties (name, description, status, progress) |
| `add_goal_to_feature` | 🟡 Medium | 🔲 | Link a goal to a feature |
| `remove_goal_from_feature` | 🟠 Low | 🔲 | Unlink a goal from a feature |

---

### Epic Management Tools

Not originally scoped in Phase 10, added alongside the extended write tools.

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `list_epics` / `list_product_epics` | 🟡 Medium | ✅ | List epics, optionally scoped to a product |
| `create_epic` | 🟡 Medium | ✅ | Create a new epic in a release |
| `update_epic` | 🟡 Medium | ✅ | Update epic properties (name, description, status, progress) |
| `list_epic_comments` | 🟡 Medium | ✅ | List comments for an epic |

---

### Initiative Management Tools

Not originally scoped in Phase 10, added alongside the extended write tools.

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `list_initiatives` / `list_product_initiatives` | 🟡 Medium | ✅ | List initiatives, optionally scoped to a product |
| `create_initiative` | 🟡 Medium | ✅ | Create a new initiative in a product |
| `update_initiative` | 🟡 Medium | ✅ | Update initiative properties (name, description, dates, value, effort, color, presented) |

---

### Product & Strategic Model Tools

Not originally scoped in Phase 10, added alongside the extended write tools.

| Tool | Priority | Status | Description |
|------|----------|--------|-------------|
| `create_product` | 🟡 Medium | ✅ | Create a new product (workspace) |
| `update_product` | 🟡 Medium | ✅ | Update an existing product |
| `list_strategic_models` / `list_product_strategic_models` | 🟡 Medium | ✅ | List strategic models, optionally scoped to a product |
| `get_strategic_model` | 🟡 Medium | ✅ | Get a strategic model by ID |
| `create_strategic_model` | 🟡 Medium | ✅ | Create a new strategic model |
| `update_strategic_model` | 🟡 Medium | ✅ | Update a strategic model |
| `get_current_user` | 🟡 Medium | ✅ | Get the authenticated user |

---

### Statistics & Analytics Tools

Aggregated metrics tools for efficient reporting without fetching all records.

| Tool | Priority | Description |
|------|----------|-------------|
| `get_ideas_statistics` | 🟡 Medium | Aggregated idea metrics (counts by status/category, vote stats, top ideas) |
| `get_features_statistics` | 🟡 Medium | Aggregated feature metrics (counts by release/status, requirements summary) |
| `get_idea_voter_domains` | 🟠 Low | Unique voter domain analytics for an idea |

#### Statistics Tool Example
```json
{
  "tool": "get_ideas_statistics",
  "params": {
    "group_by": ["workflow_status", "category"],
    "min_votes": 5
  }
}
// Returns: counts, vote totals, top 5 ideas per group
```

---

### Integration Tools (External Systems)

| Tool | Priority | Description |
|------|----------|-------------|
| `fetch_jira_issue` | 🟠 Low | Fetch Jira issue details |
| `search_jira_issues` | 🟠 Low | Search Jira issues |
| `read_confluence_page` | 🟠 Low | Read Confluence page content |
| `search_confluence` | 🟠 Low | Search Confluence |

---

### AI Agent Workflow Tools

Specialized tools for AI-assisted product management workflows.

| Tool | Priority | Description |
|------|----------|-------------|
| `review_idea` | 🟠 Low | Comprehensive PM-level idea analysis with competitive research guidance |
| `triage_new_ideas` | 🟠 Low | Multi-step workflow for triaging new ideas |
| `groom_feature` | 🟠 Low | Feature grooming workflow with requirements generation |

---

### Implementation Order

**Phase 10a - Core Write Operations (High Priority)** ✅ Complete

1. ✅ `list_workflow_statuses` - Required for status changes
2. ✅ `list_releases` - Required for release assignments
3. ✅ `create_feature` - Create new features
4. ✅ `change_feature_status` - Update feature status
5. ✅ `assign_feature_release` - Assign to release
6. ✅ `assign_user_to_feature` - Assign owner
7. ✅ `add_feature_comment` - Add comments
8. ✅ `add_idea_comment` - Add idea comments

**Phase 10b - Extended Write Operations (Medium Priority)** 🔄 Mostly Complete

1. ✅ `update_feature` - Full feature updates
2. ✅ `list_features` - List with pagination
3. ✅ `list_features_by_release` - Release-scoped list (`list_release_features`)
4. 🔲 `create_idea` / ✅ `update_idea` - Idea management
5. 🔲 `create_release` / ✅ `update_release` - Release management
6. ✅ `list_requirements` / ✅ `create_requirement` / ✅ `update_requirement` / ✅ `delete_requirement` - Requirements
7. ✅ `list_goals` / 🔲 `add_goal_to_feature` - Goal linking
8. ✅ `list_users` - User lookup

Delivered beyond the original scope: full CRUD for epics, initiatives, products,
and strategic models; comment update/delete/list (feature, idea, epic); custom
field definitions and options lookup; `get_current_user`. See the Epic,
Initiative, and Product & Strategic Model tables above for details.

**Phase 10c - Analytics & Statistics (Medium Priority)**

1. `get_ideas_statistics` - Idea metrics
2. `get_features_statistics` - Feature metrics
3. `list_idea_categories` - Category lookup

**Phase 10d - Integrations & AI Workflows (Low Priority)**

1. Jira integration tools
2. Confluence integration tools
3. AI agent workflow tools

---

### Tool Schema Examples

#### create_feature
```go
skill.NewTool("create_feature", "Create a new feature in Aha!",
    map[string]skill.Parameter{
        "product_id": {Type: "string", Required: true, Description: "Product ID"},
        "name": {Type: "string", Required: true, Description: "Feature name"},
        "description": {Type: "string", Required: false, Description: "Feature description (HTML supported)"},
        "release_id": {Type: "string", Required: false, Description: "Release ID to assign"},
        "release_name": {Type: "string", Required: false, Description: "Release name (will lookup ID)"},
        "workflow_status": {Type: "string", Required: false, Description: "Initial status name"},
        "assigned_to_user": {Type: "string", Required: false, Description: "User email to assign"},
    },
    s.handlers.CreateFeature)
```

#### list_workflow_statuses
```go
skill.NewTool("list_workflow_statuses", "List all workflow statuses for a product",
    map[string]skill.Parameter{
        "product_id": {Type: "string", Required: true, Description: "Product ID"},
    },
    s.handlers.ListWorkflowStatuses)
// Returns: [{id, name, color, position, complete}...]
```

#### change_feature_status
```go
skill.NewTool("change_feature_status", "Change the workflow status of a feature",
    map[string]skill.Parameter{
        "feature_id": {Type: "string", Required: true, Description: "Feature ID or reference"},
        "status": {Type: "string", Required: true, Description: "New status name or ID"},
    },
    s.handlers.ChangeFeatureStatus)
```

---

### Dependencies

These tools require corresponding endpoints in `aha-go`:

| aha-go Method | Required For |
|---------------|--------------|
| `CreateFeature()` | create_feature |
| `UpdateFeature()` | update_feature, change_feature_status, assign_* |
| `ListWorkflows()` | list_workflow_statuses |
| `CreateRelease()` | create_release |
| `ListReleases()` | list_releases |
| `CreateIdea()` | create_idea |
| `UpdateIdea()` | update_idea |
| `DeleteIdea()` | delete_idea |
| `CreateComment()` | add_feature_comment, add_idea_comment |
| `CreateRequirement()` | create_requirement |
| `UpdateRequirement()` | update_requirement |
| `DeleteRequirement()` | delete_requirement |

---

## Phase 11: Web UI & HTTP API

**Status:** 🔄 In Progress

Web-based AQL query interface with REST API server and Lit web component.

### Overview

Provides a browser-based AQL console that can query the HTTP API server, enabling:

- Interactive AQL queries without CLI installation
- Embeddable query component for MkDocs documentation
- REST API for programmatic access
- Offline/cached query support

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Browser (Lit Component)                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Server URL  │  │ AQL Editor  │  │ Results Table/JSON      │  │
│  │ + API Key   │  │ + Mode      │  │ + Export + Source       │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP POST /api/query
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Go HTTP Server                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────────────────┐   │
│  │ CORS     │→ │ Auth     │→ │ Query Handler                │   │
│  │ Middleware│  │ Middleware│  │ (studio + sync/SQLite)      │   │
│  └──────────┘  └──────────┘  └──────────────────────────────┘   │
│                                       │                          │
│                    ┌──────────────────┼──────────────────┐      │
│                    ▼                  ▼                  ▼      │
│              ┌──────────┐      ┌──────────┐      ┌──────────┐  │
│              │ Aha API  │      │ SQLite   │      │ Neo4j    │  │
│              │ (live)   │      │ (cache)  │      │ (graph)  │  │
│              └──────────┘      └──────────┘      └──────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### HTTP Server Package

**Status:** ✅ Complete

| File | Status | Description |
|------|--------|-------------|
| `httpserver/server.go` | ✅ | Server struct, config, initialization, query modes |
| `httpserver/routes.go` | ✅ | Route definitions for all endpoints |
| `httpserver/handlers.go` | ✅ | Query, sync, and helper handlers |
| `httpserver/middleware.go` | ✅ | CORS, auth, logging, recovery middleware |
| `httpserver/server_test.go` | ✅ | Unit tests |

### API Endpoints

| Endpoint | Method | Status | Description |
|----------|--------|--------|-------------|
| `/health` | GET | ✅ | Health check |
| `/api/version` | GET | ✅ | Server version and capabilities |
| `/api/query` | GET | ✅ | Execute AQL (query string) |
| `/api/query` | POST | ✅ | Execute AQL (JSON body) |
| `/api/entities` | GET | ✅ | List available entities |
| `/api/syntax` | GET | ✅ | AQL syntax help |
| `/api/products` | GET | ✅ | List products |
| `/api/cache/status` | GET | ✅ | Cache configuration status |
| `/api/sync` | POST | ✅ | Trigger sync for product |
| `/api/sync/status` | GET | ✅ | Get sync status |
| `/api/filters` | GET | ✅ | List saved filters |
| `/api/filters` | POST | ✅ | Create saved filter |
| `/api/filters/{id}` | GET | ✅ | Get filter by ID |
| `/api/filters/{id}` | PUT | ✅ | Update filter |
| `/api/filters/{id}` | DELETE | ✅ | Delete filter |
| `/api/search` | GET | ✅ | Full-text search |
| `/api/graph/status` | GET | ✅ | Graph client status |
| `/api/graph/features/{id}/connected` | GET | ✅ | Connected features |
| `/api/graph/path` | GET | ✅ | Shortest path |
| `/api/graph/releases/{id}/dependencies` | GET | ✅ | Release dependencies |
| `/api/graph/initiatives/{id}/impact` | GET | ✅ | Initiative impact |
| `/api/graph/products/{id}/overview` | GET | ✅ | Product overview |
| `/api/graph/cypher` | POST | ✅ | Raw Cypher query |

### Query Modes

| Mode | Description |
|------|-------------|
| `api` | Query live Aha API (default without cache) |
| `offline` | Query only local SQLite cache |
| `prefer-cache` | Try cache first, fallback to API (default with cache) |

### CLI `serve` Command

**Status:** ✅ Complete

```bash
# Start HTTP server
aha-studio serve

# Custom port
aha-studio serve --addr :3000

# Restrict CORS
aha-studio serve --cors "https://grokify.github.io"

# Local dev mode (no auth)
aha-studio serve --no-auth

# Offline-only mode
aha-studio serve --offline

# Set default query mode
aha-studio serve --prefer api

# Enable background sync
aha-studio serve --background-sync --product PLATFORM --sync-interval 15m
```

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | Listen address |
| `--cors` | `*` | Allowed CORS origins |
| `--no-auth` | `false` | Disable API key authentication |
| `--offline` | `false` | Offline-only mode (no API calls) |
| `--prefer` | `prefer-cache` | Default query mode |
| `--background-sync` | `false` | Enable automatic background sync |
| `--sync-interval` | `15m` | Background sync interval |
| `--product` | `""` | Default product for sync |

### Lit Web Component

**Status:** ✅ Complete

Package: `@grokify/aql-console` in `web-tools` monorepo

Location: `/Users/johnwang/go/src/github.com/grokify/web-tools/packages/aql-console/`

```html
<aql-console
  server-url="http://localhost:8080"
  api-key="your-api-key"
  theme="dark"
  show-history
></aql-console>
```

| Attribute | Type | Description |
|-----------|------|-------------|
| `server-url` | string | HTTP server URL |
| `api-key` | string | Aha API key |
| `theme` | `light` \| `dark` | UI theme |
| `show-history` | boolean | Show query history |

Features:

- Server URL and API key inputs
- Query mode selector (API/Offline/Prefer Cache)
- AQL textarea with placeholder examples
- Execute button (+ Ctrl+Enter)
- Results table with sortable columns
- JSON view toggle
- Query history (localStorage)
- Data source badge (API/Cache)
- Export to CSV/JSON

### MkDocs Static Page

**Status:** ✅ Complete

File: `docs/console.html`

Standalone HTML page that loads the `@grokify/aql-console` component, suitable for GitHub Pages or MkDocs deployment.

### Remaining Features

| Feature | Priority | Status | Description |
|---------|----------|--------|-------------|
| Saved Filters API | 🟡 Medium | ✅ | `GET/POST/PUT/DELETE /api/filters` for named queries |
| Full-Text Search Endpoint | 🟡 Medium | ✅ | `/api/search?q=...` using FTS5 |
| Neo4j Graph Endpoints | 🟡 Medium | ✅ | Expose graph analytics via HTTP |
| OpenAPI/Swagger Spec | 🟡 Medium | 🔲 | OpenAPI 3.0 specification |
| Metrics Endpoint | 🟠 Low | 🔲 | `/metrics` Prometheus endpoint |
| WebSocket Streaming | 🟠 Low | 🔲 | Real-time query results |

### Saved Filters API

**Status:** ✅ Complete

Saved filters are stored in the same SQLite database as the cache (`~/.ahastudio/cache.db`).

```sql
CREATE TABLE saved_filters (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  aql TEXT NOT NULL,
  product TEXT,
  description TEXT,
  is_favorite INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

Endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/filters` | List all saved filters (favorites first) |
| POST | `/api/filters` | Create filter `{name, aql, product?, description?, is_favorite?}` |
| GET | `/api/filters/{id}` | Get filter by ID |
| PUT | `/api/filters/{id}` | Update filter |
| DELETE | `/api/filters/{id}` | Delete filter |

Features:

- AQL syntax validation on create/update
- Unique name constraint
- Optional product context
- Favorite flag for prioritization

### Neo4j Graph Endpoints

**Status:** ✅ Complete

Exposes the existing `graph/` package via HTTP. Requires Neo4j connection via environment variables:

- `NEO4J_URI` - Connection URI (default: `neo4j://localhost:7687`)
- `NEO4J_USERNAME` - Username (default: `neo4j`)
- `NEO4J_PASSWORD` - Password (required to enable graph features)
- `NEO4J_DATABASE` - Database name (default: `neo4j`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/graph/status` | GET | Check if graph is enabled |
| `/api/graph/features/{id}/connected` | GET | Find features connected to an entity |
| `/api/graph/path` | GET | Find shortest path between entities |
| `/api/graph/releases/{id}/dependencies` | GET | Get feature dependencies in a release |
| `/api/graph/initiatives/{id}/impact` | GET | Get entities connected to an initiative |
| `/api/graph/products/{id}/overview` | GET | Get product entity counts |
| `/api/graph/cypher` | POST | Execute raw Cypher query |

Example - Find connected features:

```
GET /api/graph/features/feature-123/connected?type=Feature&depth=2
```

Example - Execute Cypher:

```json
POST /api/graph/cypher
{
  "cypher": "MATCH (f:Feature) RETURN f LIMIT 10",
  "params": {}
}
```

### Full-Text Search Endpoint

**Status:** ✅ Complete

Exposes the existing FTS5 full-text search from the `sync/` package via HTTP.

```
GET /api/search?q=authentication&entities=features,ideas&limit=20
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `q` | Yes | - | Search query string |
| `entities` | No | all | Comma-separated list: features, ideas, initiatives, epics |
| `limit` | No | 100 | Maximum results to return |

Response:

```json
{
  "query": "authentication",
  "count": 5,
  "results": [
    {
      "entity": "features",
      "id": "feature-123",
      "name": "User Authentication",
      "description": "OAuth2 login flow...",
      "score": -2.5
    }
  ]
}
```

Note: Lower scores indicate better matches (BM25 ranking).

### Implementation Files

| File | Description |
|------|-------------|
| `httpserver/server.go` | Server with query modes |
| `httpserver/routes.go` | Route definitions |
| `httpserver/handlers.go` | Query and sync handlers |
| `httpserver/middleware.go` | CORS, auth, logging |
| `cmd/aha-studio/main.go` | `serve` command |
| `web-tools/packages/aql-console/` | Lit component |
| `docs/console.html` | Standalone HTML page |

---

## Phase 12: Feature-Release Query Support

**Status:** 🔄 In Progress

Enable querying features by release date or name from the local cache.

### Overview

Currently, the feature sync captures only basic metadata without release assignment. This phase enhances the sync and query layer to:

1. Capture feature→release relationships during sync
2. Enable AQL queries filtering by release date/name
3. Support efficient release-based feature lookups

### Use Case

```sql
-- Query features in OBSERV with release date 2026-10-31
FROM features
WHERE product = 'OBSERV' AND release.date = '2026-10-31'

-- Query features by release name
FROM features
WHERE product = 'OBSERV' AND release.name = 'Q4 2026'

-- Join features with releases
SELECT f.name, r.name as release_name, r.release_date
FROM features f
JOIN releases r ON f.release_id = r.id
WHERE r.release_date >= '2026-01-01'
```

### Implementation

| Item | Status | Description |
|------|--------|-------------|
| Enhance feature sync | ✅ Complete | Fetch full feature details including release_id via GraphQL |
| Store release_id | ✅ Complete | Add release_id column to features table with migration |
| Feature→Release relationship | 🔲 Planned | Create relationship in relationships table |
| `GetFeaturesByReleaseDate` | 🔲 Planned | Query method in sync/db.go |
| `GetFeaturesByReleaseName` | 🔲 Planned | Query method in sync/db.go |
| AQL `release.date` support | 🔲 Planned | Executor support for release.date in WHERE |
| AQL `release.name` support | 🔲 Planned | Executor support for release.name in WHERE |
| MCP tool exposure | 🔲 Planned | `list_features_by_release` tool |

### Schema Changes

```sql
-- Add release_id to features table (migration)
ALTER TABLE features ADD COLUMN release_id TEXT;
CREATE INDEX idx_features_release ON features(release_id);

-- Feature→Release relationship
INSERT INTO relationships (from_type, from_id, rel_type, to_type, to_id, product)
VALUES ('feature', 'feat-123', 'BELONGS_TO', 'release', 'rel-456', 'OBSERV');
```

### Dependencies

Requires aha-go SDK enhancement to fetch features with release details.

---

## Phase 13: DuckDB Migration (Future)

**Status:** 🔲 Planned

Migrate from SQLite to DuckDB for improved analytical query performance.

### Rationale

- **Columnar storage** - Better for analytical scans and aggregations
- **Faster JOINs** - More efficient for cross-entity queries
- **Native Parquet** - Easy data export for external analysis
- **Modern SQL** - Better window function and CTE support

### Migration Strategy

1. Keep SQLite for sync metadata tracking
2. Use DuckDB for entity data and AQL queries
3. Gradual migration with feature flag
4. Validate performance improvements

### Timeline

- Implement after Phase 12 is complete and validated
- Learn from real query patterns before optimizing

---

## Contributing

When implementing a phase:

1. Create a feature branch: `feature/phase-N-description`
2. Update this roadmap with implementation status
3. Add tests for new functionality
4. Update documentation
5. Submit PR for review

---

## Version History

See [CHANGELOG.md](https://github.com/grokify/aha-studio/blob/main/CHANGELOG.md)
and the [release notes](../releases/v0.9.0.md) for accurate, per-release history with
commit references. The phase-to-version mapping below is directional planning
only, not a record of what actually shipped in each release.

| Phase | Target Version | Status | Notes |
|-------|-----------------|--------|-------|
| Phase 10a | v0.10.0 or earlier | ✅ Complete | Core MCP write tools (create, update, status, assign) |
| Phase 11 | TBD | 🔄 In Progress | Web UI & HTTP API (server, Lit component, cache modes) |
| Phase 12 | TBD | 🔄 In Progress | Feature-Release Query Support (release.date, release.name) |
| Phase 10b | v0.10.0 | 🔄 Mostly Complete | Extended CRUD tools |
| Phase 10c | TBD | 🔲 Planned | Statistics and analytics tools |
| Phase 13 | TBD | 🔲 Planned | DuckDB migration for analytics performance |
| Phase 10d | TBD | 🔲 Planned | Integrations and AI workflows |

## Dependencies

### Core

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/grokify/aha-go` | v0.5.0 | Aha.io API client |
| `github.com/spf13/cobra` | v1.9.x | CLI framework |
| `github.com/c-bata/go-prompt` | v0.2.6 | Interactive REPL |
| `github.com/olekukonko/tablewriter` | v0.0.5 | Table formatting |
| `gopkg.in/yaml.v3` | v3.0.1 | Configuration |

### MCP Integration (Phase 7)

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/modelcontextprotocol/go-sdk` | latest | Official MCP Go SDK |
| `github.com/plexusone/omniskill` | latest | Unified skill infrastructure |

### Extended Features (Phase 9)

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/xuri/excelize/v2` | v2.8.x | Excel file generation |
| `modernc.org/sqlite` | latest | Pure Go SQLite driver |

### Web UI & HTTP API (Phase 11)

| Dependency | Version | Purpose |
|------------|---------|---------|
| `net/http` | stdlib | HTTP server (Go standard library) |
| `@grokify/aql-console` | latest | Lit web component (npm) |
| `lit` | v3.x | Web component framework |
| `vite` | v6.x | Build tooling |
