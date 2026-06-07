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

-- Custom field (prefixed with cf_)
FROM features WHERE cf_priority = 'High'
```

## Comments

```sql
-- Single line comment
FROM features LIMIT 10

/* Multi-line
   comment */
FROM ideas LIMIT 5
```
