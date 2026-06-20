# AQL Syntax

AQL (Aha Query Language) provides a SQL-like interface for querying Aha.io data.

## Basic Syntax

```sql
[SELECT fields] FROM entity [WHERE conditions] [ORDER BY field [ASC|DESC]] [LIMIT n]
```

## Query Examples

### Simple Queries

```sql
-- Get all features (limited)
FROM features LIMIT 10

-- Get all ideas
FROM ideas LIMIT 20

-- Get all releases
FROM releases
```

### SELECT Clause

```sql
-- Select specific fields
SELECT name, status, created_at FROM features LIMIT 10

-- Select with aliases
SELECT name, status as current_status FROM features LIMIT 10
```

### WHERE Clause

```sql
-- Filter by status
FROM features WHERE status = 'In Progress'

-- Filter by tag
FROM ideas WHERE tag = 'customer-request'

-- Multiple conditions
FROM features WHERE status = 'In Progress' AND tag = 'v2'

-- String contains
FROM features WHERE name CONTAINS 'authentication'

-- Date comparison
FROM ideas WHERE created_at >= '2024-01-01'

-- Filter by release ID
FROM features WHERE release_id = 'PROD-R-123'

-- Filter by release date
FROM features WHERE release_date >= '2024-01-01' AND release_date <= '2024-06-30'
```

### ORDER BY Clause

```sql
-- Order ascending
FROM features ORDER BY created_at ASC LIMIT 10

-- Order descending
FROM releases ORDER BY release_date DESC LIMIT 5
```

### LIMIT Clause

```sql
-- Limit results
FROM features LIMIT 10

-- Get single result
FROM features WHERE reference_num = 'FEAT-123' LIMIT 1
```

## Aggregate Functions

```sql
-- Count records
SELECT COUNT(*) FROM features

-- Count by group
SELECT status, COUNT(*) as count FROM features GROUP BY status

-- Group by field
SELECT product_id, status, COUNT(*) FROM features GROUP BY product_id, status
```

## Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equals | `status = 'Done'` |
| `!=` | Not equals | `status != 'Closed'` |
| `>` | Greater than | `created_at > '2024-01-01'` |
| `>=` | Greater or equal | `score >= 100` |
| `<` | Less than | `priority < 3` |
| `<=` | Less or equal | `votes <= 10` |
| `CONTAINS` | String contains | `name CONTAINS 'auth'` |
| `IN` | In list | `status IN ('Open', 'In Progress')` |
| `AND` | Logical AND | `status = 'Open' AND tag = 'v2'` |
| `OR` | Logical OR | `status = 'Open' OR status = 'Closed'` |

## Data Types

| Type | Example |
|------|---------|
| String | `'In Progress'`, `"Done"` |
| Number | `100`, `3.14` |
| Boolean | `true`, `false` |
| Date | `'2024-01-01'`, `'2024-01-01T12:00:00Z'` |
| Null | `null` |

## Field References

Fields can be referenced directly by name:

```sql
-- Simple field reference
FROM features WHERE status = 'Done'

-- Custom field (prefixed with custom.)
FROM features WHERE custom.priority = 'High'
```

## Comments

```sql
-- Single line comment
FROM features LIMIT 10

/* Multi-line
   comment */
FROM ideas LIMIT 5
```

## Excel Export

AQL supports exporting query results to Excel (XLSX) format using the `-o xlsx` flag with `-f <filename>`.

### Export Features by Rank

```bash
# Export all features ordered by rank
aha-studio query -o xlsx -f features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features
   ORDER BY position ASC"
```

### Export Features by Release

```bash
# Export features for a specific release ID
aha-studio query -o xlsx -f release-features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features
   WHERE release_id = 'PROD-R-123'
   ORDER BY position ASC"

# Export features by release name
aha-studio query -o xlsx -f release-features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace
   FROM features
   WHERE release = 'Q4 2024 Release'
   ORDER BY position ASC"

# Export features by release date
aha-studio query -o xlsx -f release-features.xlsx \
  "SELECT reference_num, name, position, release, tag_list, workspace
   FROM features
   WHERE release_date = '2024-12-31'
   ORDER BY position ASC"

# Export features by release date range
aha-studio query -o xlsx -f q4-features.xlsx \
  "SELECT reference_num, name, position, release, release_date, tag_list, workspace
   FROM features
   WHERE release_date >= '2024-10-01' AND release_date <= '2024-12-31'
   ORDER BY release_date ASC, position ASC"
```

### Export with Column Aliases

```bash
# Use aliases for human-readable column headers
aha-studio query -o xlsx -f features.xlsx \
  "SELECT reference_num AS 'Feature Reference',
          name AS 'Feature Name',
          position AS 'Feature Rank',
          tag_list AS 'Feature Tags',
          workspace AS 'Workspace Name'
   FROM features
   ORDER BY position ASC"
```

**Note:** Release, rank, tags, and workspace data require syncing first and using offline mode:

```bash
# 1. Sync data (uses GraphQL to capture release info, rank, tags, workspace)
aha-studio sync --product PROD

# 2. Query using offline mode (queries SQLite cache)
aha-studio query --offline -o xlsx -f features.xlsx \
  "SELECT reference_num, name, position, tag_list, workspace FROM features ORDER BY position ASC"
```

The `release_id`, `position`, `tag_list`, and `workspace` fields are only available via GraphQL sync - the REST API does not include them. Use `--offline` flag to query the synced SQLite cache.
