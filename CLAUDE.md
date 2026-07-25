# CLAUDE.md — aha-studio

CLI tools for [Aha!](https://www.aha.io/) product management: AQL (Aha Query Language) with a lexer→parser→planner→executor pipeline, an MCP server built on `plexusone/omniskill`, local SQLite sync with FTS5, optional Neo4j graph analytics, and an HTTP API server. Specs live in `docs/specs/` (ARCHITECTURE, PRD, TRD, PLAN, ROADMAP); completed historical phases are in `docs/specs/ROADMAP_HISTORY.md`.

## PRISM Control

This repo's roadmap items are tracked in [prism-control](https://github.com/ProductBuildersHQ/prism-control). Use `prismctl work ready --repo github.com/grokify/aha-studio` to find claimable work, and carry the `Refs: RMI-AHASTUDIO-<NNN>` trailer on every commit.
