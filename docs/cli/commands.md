# CLI Commands

Aha Studio provides two CLI tools: `aha-studio` for AQL queries and `aha-mcp-server` for MCP integration.

---

## aha-studio Commands

The `aha-studio` CLI provides AQL (Aha Query Language) for querying and exporting Aha.io data.

### query

Execute an AQL query.

```bash
aha-studio query "FROM features LIMIT 10"
aha-studio query "FROM ideas WHERE status = 'Shipped' LIMIT 5"
```

| Flag | Description |
|------|-------------|
| `-o, --output` | Output format: table, json, csv, markdown, yaml, html, xlsx |
| `-f, --file` | Output file path (required for xlsx format) |
| `-p, --product` | Product ID or reference prefix |
| `-v, --verbose` | Enable verbose output |
| `--stats` | Show query execution statistics |
| `--offline` | Use local SQLite cache instead of API |

### sync

Sync data from Aha.io to local SQLite cache.

```bash
aha-studio sync --product PROD
aha-studio sync --product PROD --since 2024-01-01
aha-studio sync --product PROD --since last

# Detailed sync: also fetch custom fields for features and initiatives
# (does an extra per-record API call, throttled via --rate-limit)
aha-studio sync --product PROD --entities features,initiatives --detailed
```

| Flag | Description |
|------|-------------|
| `-p, --product` | Product ID (required) |
| `--since` | Sync changes since date (YYYY-MM-DD) or "last" |
| `--entities` | Specific entities to sync (comma-separated) |
| `--detailed` | Fetch full per-record data including custom fields (features and initiatives only) |
| `--rate-limit` | Requests/sec for the sync client (default 10; Aha allows up to 20/sec, 300/min) |

!!! note
    `--detailed` mode makes an additional API call per record (list-then-get), so it's slower than a normal sync. Custom fields are stored in the cache's `data` JSON column, not yet queryable via AQL.

### shell

Start interactive AQL shell.

```bash
aha-studio shell
```

### Excel Export Examples

Export features to Excel with various filters:

```bash
# Export all features ordered by rank
aha-studio query -o xlsx -f features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features ORDER BY position ASC"

# Export features for a specific release
aha-studio query -o xlsx -f release-features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features WHERE release_id = 'PROD-R-123' ORDER BY position ASC"

# Export features by release name
aha-studio query -o xlsx -f release-features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features WHERE release = 'Q4 2024 Release' ORDER BY position ASC"

# Export features by release date range
aha-studio query -o xlsx -f q4-features.xlsx \
  "SELECT reference_num, name, position, release, release_date, tag_list, workspace
   FROM features
   WHERE release_date >= '2024-10-01' AND release_date <= '2024-12-31'
   ORDER BY release_date ASC, position ASC"

# Export with human-readable column headers
aha-studio query -o xlsx -f features.xlsx \
  "SELECT reference_num AS 'Feature Reference',
          name AS 'Feature Name',
          position AS 'Feature Rank',
          tag_list AS 'Feature Tags',
          workspace AS 'Workspace Name'
   FROM features ORDER BY position ASC"
```

**Note:** Release, rank, tags, and workspace data require syncing first:

```bash
aha-studio sync --product PROD
```

---

## aha-mcp-server Commands

The Aha! MCP server can also be used as a command-line tool for testing and scripting.

### tool

Invoke any registered MCP tool directly, without a running MCP server session.

This is useful for AI agent sessions that cannot register a new or updated MCP
tool mid-session — an agent's MCP servers are fixed at launch, so picking up a
tool that was added or changed normally requires a restart. `tool call` lets
an agent shell out to run any registered tool by name instead.

```bash
# List all available tools with descriptions
aha-mcp-server tool list

# Print the JSON Schema for a tool's parameters
aha-mcp-server tool schema list_initiative_features

# Call a tool with JSON parameters (as an argument)
aha-mcp-server tool call get_feature '{"reference":"FEAT-123"}'

# Call a tool with JSON parameters via stdin (omit the argument, or pass "-")
echo '{"initiative_id":"PROD-S-34"}' | aha-mcp-server tool call list_initiative_features
```

Credentials are resolved the same way as `serve`: `--subdomain`/`--api-key` flags,
falling back to `AHA_SUBDOMAIN`/`AHA_API_KEY` (or `AHA_DOMAIN`/`AHA_API_TOKEN`).

Results are printed to stdout as pretty-printed JSON; errors go to stderr with a
non-zero exit code, so `tool call` composes cleanly with `jq` and shell scripting:

```bash
aha-mcp-server tool call list_initiative_features '{"initiative_id":"PROD-S-34"}' \
  | jq -r '.features[].reference_num'
```

Sync (`sync_data`) requires `AHA_DB_PATH` (defaults to `~/.aha-studio/cache.db` if unset)
and is throttled via `AHA_SYNC_RPS` (default 10 req/s):

```bash
aha-mcp-server tool call sync_data '{"product":"PROD","entities":["initiatives"],"detailed":true}'
```

## Global Flags

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--subdomain` | `AHA_DOMAIN` | Aha! subdomain |
| `--api-key` | `AHA_API_TOKEN` | Aha! API key |
| `--vault` | `OMNITOKEN_VAULT_URI` | Vault URI for credentials |
| `--credentials-name` | `OMNITOKEN_CREDENTIALS_NAME` | Name of credentials in vault |
| `-o, --output` | - | Output format: json (default) or pretty |

## Server Commands

### serve

Start the MCP server (default when no command specified).

```bash
aha-mcp-server serve
aha-mcp-server  # Same as above
```

### version

Print version information.

```bash
aha-mcp-server version
```

## Aha! Commands

### search-documents

Search for documents.

```bash
aha-mcp-server search-documents "product roadmap"
aha-mcp-server search-documents "auth" --type Page --limit 5
```

| Flag | Description |
|------|-------------|
| `--type` | Document type (e.g., Page) |
| `--limit` | Maximum results |

### get-idea

Get an idea by ID.

```bash
aha-mcp-server get-idea IDEA-123
aha-mcp-server get-idea IDEA-123 --output pretty
```

### list-ideas

List ideas with filtering.

```bash
aha-mcp-server list-ideas
aha-mcp-server list-ideas --query "mobile" --workflow-status "Under consideration"
aha-mcp-server list-ideas --tag "priority" --sort recent --per-page 20
```

| Flag | Description |
|------|-------------|
| `-q, --query` | Search term |
| `--spam` | Show spam ideas |
| `--workflow-status` | Filter by status |
| `--sort` | Sort by: recent, trending, popular |
| `--created-before` | Created before date (ISO8601) |
| `--created-since` | Created after date (ISO8601) |
| `--updated-since` | Updated after date (ISO8601) |
| `--tag` | Filter by tag |
| `--user-id` | Filter by creator |
| `--page` | Page number |
| `--per-page` | Results per page |

### get-feature

Get a feature by ID.

```bash
aha-mcp-server get-feature FEAT-123
```

### get-epic

Get an epic by ID.

```bash
aha-mcp-server get-epic EPIC-456
```

### get-release

Get a release by ID.

```bash
aha-mcp-server get-release REL-789
```

### get-goal

Get a goal by ID.

```bash
aha-mcp-server get-goal GOAL-123
```

### get-initiative

Get an initiative by ID.

```bash
aha-mcp-server get-initiative INIT-456
```

### get-key-result

Get a key result by ID.

```bash
aha-mcp-server get-key-result KR-789
```

### get-persona

Get a persona by ID.

```bash
aha-mcp-server get-persona PERS-123
```

### get-requirement

Get a requirement by ID.

```bash
aha-mcp-server get-requirement REQ-456
```

### get-team

Get a team by ID.

```bash
aha-mcp-server get-team TEAM-789
```

### get-user

Get a user by ID.

```bash
aha-mcp-server get-user USER-123
```

### get-workflow

Get a workflow by ID.

```bash
aha-mcp-server get-workflow WF-456
```

### get-comment

Get a comment by ID.

```bash
aha-mcp-server get-comment COMMENT-789
```

## Examples

### Scripting with JSON

```bash
# Get feature and extract name with jq
aha-mcp-server get-feature FEAT-123 | jq '.feature.name'

# List ideas and count them
aha-mcp-server list-ideas --query "mobile" | jq '.ideas | length'

# Search and format results
aha-mcp-server search-documents "roadmap" | jq -r '.results[].title'
```

### Using with Vault

```bash
# 1Password
export OP_SERVICE_ACCOUNT_TOKEN="ops_..."
aha-mcp-server get-feature FEAT-123 --vault op://MyVault --credentials-name aha

# Bitwarden
export BW_ACCESS_TOKEN="..."
export BW_ORGANIZATION_ID="..."
aha-mcp-server list-ideas --vault bw://org-id --credentials-name aha
```
