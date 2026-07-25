# Aha Studio — Roadmap

**Initiative:** `INIT-AHASTUDIO-001`
**Repository:** `github.com/grokify/aha-studio`
**Status:** Planned — 0 of 7 phases completed

> RMI IDs are stable and permanent. Commits implementing an item carry the trailer `Refs: RMI-AHASTUDIO-NNN`. Phase status is derived from member RMIs — a phase is complete only when all its required RMIs are complete. Completed historical work (legacy phases 1–10a and most of 10b) is recorded in [ROADMAP_HISTORY.md](ROADMAP_HISTORY.md); this file covers remaining work only. See [PLAN.md](PLAN.md) for workstream rationale and sequencing.

## Phase 1 — Release Query Completion

**Theme:** Finish feature-release queries and clear the release blocker.
**Status:** Not started — 0 of 5 items completed

- [ ] `RMI-AHASTUDIO-001` Feature→Release relationship rows during sync
  - Acceptance: sync inserts `BELONGS_TO` rows into the relationships table for every feature with a `release_id`
- [ ] `RMI-AHASTUDIO-002` `GetFeaturesByReleaseDate` and `GetFeaturesByReleaseName` in `sync/db.go`
  - Depends on: `RMI-AHASTUDIO-001`
  - Acceptance: date-indexed and name-based lookups joining features to releases, with unit tests
- [ ] `RMI-AHASTUDIO-003` AQL `release.date` and `release.name` qualifier support in planner and executor
  - Depends on: `RMI-AHASTUDIO-002`
  - Acceptance: `FROM features WHERE release.date = '2026-10-31'` and `release.name = 'Q4 2026'` work in offline and prefer-cache modes; extends the existing `custom.*` qualifier mechanism
- [ ] `RMI-AHASTUDIO-004` `list_features_by_release_date` MCP tool
  - Depends on: `RMI-AHASTUDIO-002`
  - Acceptance: thin wrapper over the db methods, registered in the Aha skill
- [ ] `RMI-AHASTUDIO-005` Remove local `replace github.com/grokify/aha-go => ../aha-go` from `go.mod`
  - Acceptance: required `aha-go` changes released and tagged upstream; `go.mod` pins a published version; pre-push checklist passes

## Phase 2 — Write Tool Gaps

**Theme:** Close out remaining legacy Phase 10b MCP write tools as one batch release.
**Status:** Not started — 0 of 6 items completed

- [ ] `RMI-AHASTUDIO-006` `create_idea` MCP tool
- [ ] `RMI-AHASTUDIO-007` `create_release` MCP tool
  - Acceptance: pairs with the existing `update_release`
- [ ] `RMI-AHASTUDIO-008` `add_goal_to_feature` and `remove_goal_from_feature` MCP tools
- [ ] `RMI-AHASTUDIO-009` `get_feature_ideas` MCP tool (ideas promoted to a feature)
- [ ] `RMI-AHASTUDIO-010` `list_idea_categories` MCP tool with caching
- [ ] `RMI-AHASTUDIO-011` `delete_idea` MCP tool with confirmation semantics
  - Acceptance: destructive operation requires explicit confirmation parameter; documented in tool description

## Phase 3 — OmniSignal Adapter

**Theme:** Bridge Aha Ideas into the ProductContext signal layer.
**Status:** Not started — 0 of 6 items completed

- [ ] `RMI-AHASTUDIO-012` IR schema coordination with signal-spec and omnisignal
  - Acceptance: cross-repo gate — `plexusone/signal-spec` defines the enhancement signal type (schema is owned there, not here); field mapping for Aha Idea agreed and recorded in TRD.md before implementation starts
- [ ] `RMI-AHASTUDIO-013` Adapter package mapping Aha Idea to enhancement signal IR
  - Depends on: `RMI-AHASTUDIO-012`
  - Acceptance: raw facts only (votes, dates, counts); derived metrics such as frustration score are computed by OmniSignal, not here
- [ ] `RMI-AHASTUDIO-014` Product and category normalization to canonical IDs
  - Depends on: `RMI-AHASTUDIO-013`
- [ ] `RMI-AHASTUDIO-015` Metrics and lifecycle extraction: votes, subscribers, organizations, named customers, opportunities, estimated ARR; workflow status mapping and `ageDays`
  - Depends on: `RMI-AHASTUDIO-013`
- [ ] `RMI-AHASTUDIO-016` Batch export command (`aha-studio signals export`) and/or provider registration with `plexusone/omnisignal`
  - Depends on: `RMI-AHASTUDIO-014`, `RMI-AHASTUDIO-015`
- [ ] `RMI-AHASTUDIO-017` Round-trip tests against recorded fixtures
  - Depends on: `RMI-AHASTUDIO-016`
  - Acceptance: no live API calls in unit tests

## Phase 4 — HTTP API Remainder

**Theme:** Complete the legacy Phase 11 HTTP surface.
**Status:** Not started — 0 of 3 items completed

- [ ] `RMI-AHASTUDIO-018` OpenAPI 3.0 specification served at `/api/openapi.json`
  - Acceptance: covers all existing endpoints; enables client generation
- [ ] `RMI-AHASTUDIO-019` `/metrics` Prometheus endpoint
  - Acceptance: query counts, latencies, cache hit rate
- [ ] `RMI-AHASTUDIO-020` WebSocket streaming for large result sets
  - Acceptance: deferred until a concrete consumer exists; re-scope before starting

## Phase 5 — Analytics Tools

**Theme:** Aggregated statistics without fetching all records to the client (legacy Phase 10c).
**Status:** Not started — 0 of 3 items completed

- [ ] `RMI-AHASTUDIO-021` `get_ideas_statistics` MCP tool
  - Acceptance: counts by status/category, vote statistics, top ideas per group; computed from the SQLite cache when synced, API aggregation as fallback
- [ ] `RMI-AHASTUDIO-022` `get_features_statistics` MCP tool
  - Acceptance: counts by release/status, requirements summary; cache-first
- [ ] `RMI-AHASTUDIO-023` `get_idea_voter_domains` MCP tool
  - Acceptance: unique voter domain analytics for an idea

## Phase 6 — DuckDB Evaluation

**Theme:** Columnar engine for analytics, benchmark-first (legacy Phase 13).
**Status:** Not started — 0 of 4 items completed

- [ ] `RMI-AHASTUDIO-024` Benchmark current SQLite performance on representative analytics queries
  - Depends on: `RMI-AHASTUDIO-021`, `RMI-AHASTUDIO-022`
- [ ] `RMI-AHASTUDIO-025` Feature-flagged DuckDB backend for entity data
  - Depends on: `RMI-AHASTUDIO-024`
  - Acceptance: SQLite retained for sync metadata
- [ ] `RMI-AHASTUDIO-026` Parquet export support
  - Depends on: `RMI-AHASTUDIO-025`
- [ ] `RMI-AHASTUDIO-027` Validate and flip default, or drop DuckDB if SQLite proves sufficient
  - Depends on: `RMI-AHASTUDIO-025`
  - Acceptance: decision recorded with benchmark evidence

## Phase 7 — External Integrations

**Theme:** Cross-system tools and AI workflows (legacy Phase 10d); lowest priority.
**Status:** Not started — 0 of 3 items completed

- [ ] `RMI-AHASTUDIO-028` Jira integration tools: `fetch_jira_issue`, `search_jira_issues`
- [ ] `RMI-AHASTUDIO-029` Confluence integration tools: `read_confluence_page`, `search_confluence`
- [ ] `RMI-AHASTUDIO-030` AI workflow tools: `review_idea`, `triage_new_ideas`, `groom_feature`
  - Acceptance: re-scope this phase after the OmniSignal adapter ships — some cross-system use cases may be better served at the OmniSignal layer
