# Aha Studio Implementation Plan

## Scope

This plan covers remaining and upcoming work only. Completed phases (1 through 10a, and most of 10b) are recorded in [ROADMAP.md](ROADMAP.md), which remains the authoritative per-phase status tracker. Update ROADMAP.md status tables as items below land.

## Workstream 1: Phase 12 — Feature-Release Query Support (In Progress)

Enable querying features by release date or name from the local cache. Sync-side capture of `release_id` is complete; the query layer remains.

| Item | Package | Notes |
|------|---------|-------|
| Feature→Release relationship rows | `sync/` | Insert `BELONGS_TO` rows into the relationships table during sync |
| `GetFeaturesByReleaseDate` | `sync/db.go` | Date-indexed lookup joining features to releases |
| `GetFeaturesByReleaseName` | `sync/db.go` | Name-based lookup |
| AQL `release.date` support | `planner/`, `executor/` | Qualifier syntax already exists for `custom.*`; extend to `release.*` |
| AQL `release.name` support | `planner/`, `executor/` | Same mechanism |
| `list_features_by_release_date` MCP tool | `mcp/` | Thin wrapper over the db methods |

Depends on `aha-go` returning release details with features (in place as of v0.7.0 usage).

## Workstream 2: Phase 11 — HTTP API Remainder (In Progress)

Core server, saved filters, search, and graph endpoints are complete.

| Item | Priority | Notes |
|------|----------|-------|
| OpenAPI 3.0 specification | Medium | Generate or hand-author; serve at `/api/openapi.json`; enables client generation |
| `/metrics` Prometheus endpoint | Low | Query counts, latencies, cache hit rate |
| WebSocket streaming | Low | Stream large result sets; defer until a concrete consumer exists |

## Workstream 3: Phase 10b Gaps — Remaining Write Tools

Small, independent items; good batch for a single release.

| Tool | Notes |
|------|-------|
| `create_idea` | `aha-go` `CreateIdea()` exists per ROADMAP dependency table |
| `create_release` | Pair with existing `update_release` |
| `add_goal_to_feature` / `remove_goal_from_feature` | Goal-feature linking |
| `get_feature_ideas` | Ideas promoted to a feature |
| `list_idea_categories` | Category lookup with caching |
| `delete_idea` | Low priority; destructive, require confirmation semantics |

## Workstream 4: Phase 10c — Analytics and Statistics Tools

Aggregated metrics without fetching all records to the client.

| Item | Notes |
|------|-------|
| `get_ideas_statistics` | Counts by status/category, vote stats, top ideas per group |
| `get_features_statistics` | Counts by release/status, requirements summary |
| `get_voter_domain_histogram` | Unique voter domain analytics (per-idea or product-wide) |

Prefer computing from the SQLite cache when synced (fast, no API quota); fall back to API aggregation.

## Workstream 5: OmniSignal Adapter (New)

Bridges Aha Studio into the ProductContext ecosystem. See TRD.md for the IR contract.

| Step | Deliverable |
|------|-------------|
| 1 | Adapter package (e.g., `omnisignal/` in this repo) mapping Aha Idea → enhancement signal IR |
| 2 | Product/category normalization: map Aha products and idea categories to canonical IDs |
| 3 | Metrics extraction: votes, subscribers, organizations, named customers, opportunities, estimated ARR |
| 4 | Lifecycle mapping: Aha workflow status → normalized status, plus `ageDays` from created date |
| 5 | Batch export command (`aha-studio signals export`) and/or provider registration with `plexusone/omnisignal` |
| 6 | Round-trip tests against recorded fixtures; no live API in unit tests |

Coordinate the IR schema with `plexusone/omnisignal` (runtime) and `plexusone/signal-spec` (canonical model) before implementing step 1 — the signal schema is owned there, not here.

## Workstream 6: Phase 13 — DuckDB Migration (Planned, Last)

Columnar engine for analytical queries. Deliberately sequenced after Phases 10c and 12 so real query patterns inform the design.

| Step | Notes |
|------|-------|
| 1 | Benchmark current SQLite performance on representative analytics queries |
| 2 | Feature-flagged DuckDB backend for entity data; SQLite retained for sync metadata |
| 3 | Parquet export support |
| 4 | Validate and flip default, or drop if SQLite proves sufficient |

## Workstream 7: Phase 10d — External Integrations (Planned)

Jira fetch/search, Confluence read/search, and AI workflow tools (`review_idea`, `triage_new_ideas`, `groom_feature`). Lowest priority; revisit scope after the OmniSignal adapter ships, since some cross-system use cases may be better served at the OmniSignal layer than inside Aha Studio.

## Sequencing Summary

1. Phase 12 completion (small, unblocks release-based reporting)
2. Phase 10b gap tools (small batch release)
3. OmniSignal adapter (strategic; needs schema coordination first)
4. Phase 11 remainder (OpenAPI spec first)
5. Phase 10c analytics tools
6. Phase 13 DuckDB evaluation
7. Phase 10d integrations

## Release Discipline

Each workstream should ship as its own tagged release following the pre-push checklist: no local `replace` directives (one currently exists in `go.mod` for `aha-go` and must be resolved before pushing), tests and lint green, CI passing before tagging, conventional commits throughout.
