# Aha Studio — Roadmap

**Initiative:** `INIT-AHASTUDIO-001`
**Repository:** `github.com/grokify/aha-studio`
**Status:** In Progress — 5 of 8 phases completed

> RMI IDs are stable and permanent. Commits implementing an item carry the trailer `Refs: RMI-AHASTUDIO-NNN`. Phase status is derived from member RMIs — a phase is complete only when all its required RMIs are complete. Completed historical work (legacy phases 1–10a and most of 10b) is recorded in [ROADMAP_HISTORY.md](ROADMAP_HISTORY.md); this file covers remaining work only. See [PLAN.md](PLAN.md) for workstream rationale and sequencing.

## Phase 1 — Release Query Completion

**Theme:** Finish feature-release queries and clear the release blocker.
**Status:** Completed — 5 of 5 items completed

- [x] `RMI-AHASTUDIO-001` Feature→Release relationship rows during sync
  - Acceptance: sync inserts `BELONGS_TO` rows into the relationships table for every feature with a `release_id`
  - Delivered: `syncFeaturesGraphQL()` calls `UpsertRelationship("feature", f.Id, "BELONGS_TO_RELEASE", "release", releaseID, product)` in sync/sync.go:263
- [x] `RMI-AHASTUDIO-002` `GetFeaturesByReleaseDate` and `GetFeaturesByReleaseName` in `sync/db.go`
  - Depends on: `RMI-AHASTUDIO-001`
  - Acceptance: date-indexed and name-based lookups joining features to releases, with unit tests
  - Delivered: `GetFeaturesByReleaseDate`, `GetFeaturesByReleaseDateRange`, `GetFeaturesByReleaseName`, `GetFeaturesByReleaseID` in sync/db.go
- [x] `RMI-AHASTUDIO-003` AQL `release.date` and `release.name` qualifier support in planner and executor
  - Depends on: `RMI-AHASTUDIO-002`
  - Acceptance: `FROM features WHERE release.date = '2026-10-31'` and `release.name = 'Q4 2026'` work in offline and prefer-cache modes; extends the existing `custom.*` qualifier mechanism
  - Delivered: `Record.Get()` maps `release.*` prefix to release fields; `featureToRecord()` populates `release.date`, `release.name`, `release.id`; planner recognizes `release.*` in `checkCustomFieldsNeeded()`
- [x] `RMI-AHASTUDIO-004` `list_features_by_release_date` MCP tool
  - Depends on: `RMI-AHASTUDIO-002`
  - Acceptance: thin wrapper over the db methods, registered in the Aha skill
  - Delivered: `list_features_by_release_date` and `list_features_by_release_name` tools in mcp/skill.go; handlers wrap sync/db.go methods
- [x] `RMI-AHASTUDIO-005` Remove local `replace github.com/grokify/aha-go => ../aha-go` from `go.mod`
  - Acceptance: required `aha-go` changes released and tagged upstream; `go.mod` pins a published version; pre-push checklist passes
  - Delivered: go.mod has no aha-go replace directive; dependency pinned to published v0.8.0

## Phase 2 — Write Tool Gaps

**Theme:** Close out remaining legacy Phase 10b MCP write tools as one batch release.
**Status:** Completed — 6 of 6 items completed

- [x] `RMI-AHASTUDIO-006` `create_idea` MCP tool
  - Delivered: GraphQL CreateIdea mutation wrapper in mcp/handlers.go
- [x] `RMI-AHASTUDIO-007` `create_release` MCP tool
  - Acceptance: pairs with the existing `update_release`
  - Delivered: aha-go REST API CreateRelease; MCP tool `create_release` in mcp/handlers.go
- [x] `RMI-AHASTUDIO-008` `add_goal_to_feature` and `remove_goal_from_feature` MCP tools
  - Delivered: `add_goal_to_feature` via GraphQL CreateRecordLink
  - Note: `remove_goal_from_feature` blocked — no DeleteRecordLink mutation in aha-go
- [x] `RMI-AHASTUDIO-009` `get_feature_ideas` MCP tool (ideas promoted to a feature)
  - Delivered: aha-go REST API ListFeatureIdeas; MCP tool `get_feature_ideas` in mcp/handlers.go
- [x] `RMI-AHASTUDIO-010` `list_idea_categories` MCP tool with caching
  - Delivered: aha-go REST API ListProductIdeaCategories; MCP tool `list_idea_categories` in mcp/handlers.go
- [x] `RMI-AHASTUDIO-011` `delete_idea` MCP tool with confirmation semantics
  - Acceptance: destructive operation requires explicit confirmation parameter; documented in tool description
  - Delivered: aha-go REST API DeleteIdea; MCP tool `delete_idea` with `confirm` parameter in mcp/handlers.go

## Phase 3 — OmniSignal Adapter

**Theme:** Bridge Aha Ideas into the ProductContext signal layer.
**Status:** Completed — 6 of 6 items completed

- [x] `RMI-AHASTUDIO-012` IR schema coordination with signal-spec and omnisignal
  - Acceptance: cross-repo gate — `plexusone/signal-spec` defines the enhancement signal type (schema is owned there, not here); field mapping for Aha Idea agreed and recorded in TRD.md before implementation starts
  - Delivered: signal-spec v0.2.0 `TypeEnhancementRequest`; field mapping in omnisignal `docs/metadata-conventions.md`
- [x] `RMI-AHASTUDIO-013` Adapter package mapping Aha Idea to enhancement signal IR
  - Depends on: `RMI-AHASTUDIO-012`
  - Acceptance: raw facts only (votes, dates, counts); derived metrics such as frustration score are computed by OmniSignal, not here
  - Delivered: `omnisignal/provider.go` with `normalizeIdea()` function
- [x] `RMI-AHASTUDIO-014` Product and category normalization to canonical IDs
  - Depends on: `RMI-AHASTUDIO-013`
  - Delivered: `normalizeIdea()` extracts product ref and category refs as typed IDs
- [x] `RMI-AHASTUDIO-015` Metrics and lifecycle extraction: votes, subscribers, organizations, named customers, opportunities, estimated ARR; workflow status mapping and `ageDays`
  - Depends on: `RMI-AHASTUDIO-013`
  - Delivered: `normalizeIdea()` populates metadata with votes, workflow status, ageDays calculation
- [x] `RMI-AHASTUDIO-016` Batch export command (`aha-studio signals export`) and/or provider registration with `plexusone/omnisignal`
  - Depends on: `RMI-AHASTUDIO-014`, `RMI-AHASTUDIO-015`
  - Delivered: `init()` registers "aha" provider with omnisignal registry
- [x] `RMI-AHASTUDIO-017` Round-trip tests against recorded fixtures
  - Depends on: `RMI-AHASTUDIO-016`
  - Acceptance: no live API calls in unit tests
  - Delivered: `omnisignal/provider_test.go` with mock client

## Phase 4 — HTTP API Remainder

**Theme:** Complete the legacy Phase 11 HTTP surface.
**Status:** Completed — 2 of 2 required items completed (1 blocked/deferred)

- [x] `RMI-AHASTUDIO-018` OpenAPI 3.0 specification served at `/api/openapi.json`
  - Acceptance: covers all existing endpoints; enables client generation
  - Delivered: `httpserver/openapi.go` with full OpenAPI 3.0.3 spec at `/api/openapi.json`
- [x] `RMI-AHASTUDIO-019` `/metrics` Prometheus endpoint
  - Acceptance: query counts, latencies, cache hit rate
  - Delivered: `httpserver/metrics.go` with Prometheus exposition format at `/metrics`
- [~] `RMI-AHASTUDIO-020` WebSocket streaming for large result sets — **blocked/deferred**
  - Acceptance: deferred until a concrete consumer exists; re-scope before starting

## Phase 5 — Analytics Tools

**Theme:** Aggregated statistics without fetching all records to the client (legacy Phase 10c).
**Status:** Completed — 3 of 3 required items completed

- [x] `RMI-AHASTUDIO-021` `get_ideas_statistics` MCP tool
  - Acceptance: counts by status/category, vote statistics, top ideas per group; computed from the SQLite cache when synced, API aggregation as fallback
  - Delivered: sync/db.go GetIdeasStatistics with status counts, vote stats, recent/updated counts, top ideas by votes
- [x] `RMI-AHASTUDIO-022` `get_features_statistics` MCP tool
  - Acceptance: counts by release/status, requirements summary; cache-first
  - Delivered: sync/db.go GetFeaturesStatistics with status/release counts, with/without release, upcoming releases with feature counts
- [x] `RMI-AHASTUDIO-023` `get_voter_domain_histogram` MCP tool
  - Acceptance: unique voter domain analytics for an idea (or product-wide)
  - Un-blocked: prior investigation checked only aha-go's vendored openapi/aha.yaml and GraphQL schema, both of which expose just the aggregate votes/numEndorsements count. Live-tested against a real Aha account and confirmed `GET /ideas/:idea_id/endorsements` returns per-voter identity (name, email) via `endorsed_by_portal_user`/`endorsed_by_idea_user`; that endpoint was simply missing from the vendored spec.
  - Delivered: aha-go endorsement.go (ListIdeaEndorsements), aha-studio ent/schema/ideaendorsement.go, sync/sync.go syncIdeaEndorsements (opt-in, incremental), sync/db.go GetVoterEmailDomainStatistics, mcp get_voter_domain_histogram tool

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

## Phase 7 — Atlassian Integration & Cross-Tool Workflows

**Theme:** Compose mcp-atlassian skills into aha-studio and add cross-tool Aha↔Jira tools leveraging the Integration field.
**Status:** Not started — 0 of 3 items completed

- [ ] `RMI-AHASTUDIO-028` Compose mcp-atlassian Jira skill into aha-studio MCP server
  - Acceptance: when Atlassian credentials are configured, aha-studio exposes all Jira tools from `plexusone/mcp-atlassian/skills/jira`; skill composition uses omniskill interfaces
  - Note: skills remain independent in mcp-atlassian for standalone use
- [ ] `RMI-AHASTUDIO-029` Compose mcp-atlassian Confluence skill into aha-studio MCP server
  - Acceptance: when Atlassian credentials are configured, aha-studio exposes all Confluence tools from `plexusone/mcp-atlassian/skills/confluence`
- [ ] `RMI-AHASTUDIO-030` Cross-tool Aha↔Jira tools leveraging the Integration field
  - Depends on: `RMI-AHASTUDIO-028`
  - Tools:
    - `get_feature_with_jira` — fetch Aha Feature + linked Jira Epic/Issue in one call (Aha is product management source, Jira is engineering implementation source)
    - `sync_feature_status` — sync workflow status between Aha Feature and linked Jira Epic
    - `link_feature_to_epic` — set the Aha Feature Integration field to a Jira Epic URL/key
    - `unlink_feature_from_epic` — clear the Integration field
    - `list_features_with_jira_links` — find all features with Jira integrations
    - `get_jira_epic_with_feature` — fetch Jira Epic + linked Aha Feature (reverse lookup)
  - Acceptance: tools handle the Integration field parsing (Jira URL → issue key extraction); bidirectional lookups work

## Phase 8 — Voter Identity and Organization Sync

**Theme:** Sync idea_users and idea_organizations from Aha, and wire voter/customer-org tiering into a cache-backed omnisignal provider.
**Status:** In progress — 0 of 1 items completed

- [ ] `RMI-AHASTUDIO-031` Sync idea_users and idea_organizations; wire voter/customer tiering into a cache-backed omnisignal provider
  - Acceptance: aha-go `ListIdeaUsers`/`GetIdeaUser`/`ListIdeaOrganizations`/`GetIdeaOrganization` live-verified against a real Aha account; aha-studio `idea_users`/`idea_organizations` sync (`Detailed=true`) populates `email_domains`/`revenue`; a new `omnisignalcache` provider (registered as `"aha-studio"`, mirroring the existing `aha-go/omniroadmap` vs `aha-studio/omniroadmap` live/cached split) produces signals with `signal.MetaCustomers` populated from resolved organization refs — `idea_organizations.email_domains` is Aha's own authoritative domain→customer mapping, no manual config needed for the common case; `metrics.Compute(ctx, "reach", ...)` over those signals returns a non-zero distinct-customer count
