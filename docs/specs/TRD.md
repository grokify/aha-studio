# Aha Studio Technical Requirements

## Scope

This document specifies the technical design constraints and implementation requirements for Aha Studio. See [ARCHITECTURE.md](ARCHITECTURE.md) for the system overview and [ROADMAP.md](ROADMAP.md) for phase-by-phase feature status.

## AQL Pipeline

The query engine is a compiler pipeline with strict package boundaries.

| Stage | Package | Requirements |
|-------|---------|--------------|
| Lexer | `aql/lexer/` | Tokenize AQL strings; keywords, identifiers, strings, numbers, durations, operators |
| Parser | `aql/parser/` | Recursive descent with operator precedence; produce AST for SELECT, FROM, WHERE, JOIN, GROUP BY, HAVING, ORDER BY, LIMIT, INSERT, UPDATE, DELETE, subqueries |
| AST | `aql/ast/` | Typed nodes only; no execution logic |
| Validator | `aql/validator/` | Semantic validation against entity schema (`schema/`); reject unknown entities/fields before planning |
| Planner | `planner/` | Produce a `Plan` with filter pushdown (API-supported predicates go server-side), custom field detection (`NeedsCustomFields`), JOIN strategy, aggregation plan |
| Executor | `executor/` | Execute the plan against the data layer; files split by concern: `filter.go`, `join.go`, `aggregate.go`, `mutation.go` |
| Result | `result/` | Records plus metadata; formatting in `formatter.go`, Excel in `excel.go` |

### Execution requirements

- Predicates the Aha API supports must be pushed down; remaining predicates filter client-side.
- Subqueries execute multi-pass: subqueries first, results substituted into the main query. Scalar subqueries must yield a single aggregate value; list subqueries a single column. Nesting is supported.
- JOINs fetch each side via the API (or cache) and join client-side. Concurrent fetches are required for JOIN sides and pagination.
- Custom fields (`custom.fieldname`) require full entity detail fetches; the planner flags this so the executor only pays the cost when needed. Supported in WHERE, SELECT, ORDER BY, GROUP BY for entities exposing `custom_fields` (features, goals).
- Mutations require `--dry-run` support and confirmation prompts for destructive operations.

## MCP Server

The MCP layer wraps studio functionality as an OmniSkill skill; the MCP protocol itself is provided through `github.com/plexusone/omniskill` (which brings in the official MCP Go SDK).

| File | Requirement |
|------|-------------|
| `mcp/skill.go` | `AhaSkill` implementing the OmniSkill `Skill` interface; declares all tools |
| `mcp/handlers.go` | One handler per tool; handlers call `studio/` facade, never the Aha API directly |
| `mcp/server.go` | Server setup, transport selection (stdio for Claude Desktop, HTTP) |
| `mcp/config.go` | Server configuration (subdomain, API key env, defaults) |

### Tool requirements

- Tools accept human-readable names where practical and resolve them to IDs via lookup tools (`list_workflow_statuses`, `list_releases`, `list_users`).
- Write tools must validate inputs before calling the API and return structured errors.
- Library mode (direct `CallTool` invocation without JSON-RPC) must remain supported for embedding.
- Integration tests (`handlers_integration_test.go`) gate live-API behavior; unit tests must not require network access.

## SQLite Sync

Package `sync/` provides the local cache using `modernc.org/sqlite` (pure Go, no CGo).

- Full sync per product; incremental sync via `updated_since` watermark (`sync.go`, `scheduler.go`).
- Schema migrations must be additive and versioned (`db.go`); e.g., the `release_id` column on features ships as a migration with an index.
- FTS5 virtual tables provide full-text search across features, ideas, initiatives, epics with BM25 ranking.
- Saved filters persist in the same database (`saved_filters` table) with unique-name constraint and AQL validation on write.
- Query modes: `api` (live only), `offline` (cache only, no network), `prefer-cache` (cache first, API fallback). The mode is selectable per query and per HTTP request.
- Background sync runs on a configurable interval when enabled (`--background-sync`, `--sync-interval`).

## Neo4j Graph

Package `graph/` is optional at runtime; enabled only when `NEO4J_PASSWORD` is set.

- `client.go` — connection via `neo4j-go-driver/v5`; config from `NEO4J_URI`, `NEO4J_USERNAME`, `NEO4J_PASSWORD`, `NEO4J_DATABASE`.
- `schema.go` — node labels (Feature, Idea, Release, Initiative, Epic, Product) and relationship types (e.g., `BELONGS_TO`).
- `sync.go` — populate the graph from synced entities.
- `queries.go` — canned analytics: connected features, shortest path, release dependencies, initiative impact, product overview; plus raw Cypher passthrough.

All graph HTTP endpoints must degrade gracefully (return a disabled status) when Neo4j is not configured.

## HTTP API

Package `httpserver/` uses the Go standard library `net/http` only.

- `routes.go` — route table; `handlers.go` — handlers calling `studio/` and `sync/`; `middleware.go` — CORS, API-key auth, logging, panic recovery, applied in that order.
- Auth is API-key based; `--no-auth` is permitted only for local development.
- Query endpoints accept a mode parameter and report the data source (API vs cache) in responses.
- Endpoint inventory and status live in ROADMAP.md Phase 11.

## Web Component

The `@grokify/aql-console` Lit component lives in the separate `web-tools` monorepo, not this repository. Aha Studio's obligations are limited to:

- Keeping the HTTP API backward compatible with the component's contract (`/api/query`, `/api/entities`, `/api/syntax`).
- Shipping `docs/console.html` as a standalone page loading the component.

## Cross-Cutting Requirements

- Errors follow the global policy: return errors up; never discard with `_`.
- All new dependencies must be version-verified before adding (releases page / pkg.go.dev / `go list -m -versions`).
- `gofmt` and `golangci-lint run` must pass; tests via `go test ./...`.
- The `replace github.com/grokify/aha-go => ../aha-go` directive in `go.mod` is local-development only and must be removed before pushing (per the pre-push checklist).

## OmniSignal Adapter (Planned)

The adapter converts Aha Ideas into vendor-neutral OmniSignal enhancement signals.

- Input: Aha Idea (via `aha-go` or the local SQLite cache).
- Output: OmniSignal IR — `type: enhancement`, `source: {provider: aha, object: idea, id}`, normalized `product`/`category`, metrics (`votes`, `subscribers`, `organizations`, `namedCustomers`, `opportunities`, `estimatedARR`), lifecycle (`status`, `ageDays`).
- Derived metrics (frustration score = votes × ageDays) are computed by OmniSignal, not the adapter; the adapter supplies raw facts only.
- The adapter must not leak Aha-specific concepts (idea portals, custom field internals) into the IR.

See PLAN.md for sequencing.
