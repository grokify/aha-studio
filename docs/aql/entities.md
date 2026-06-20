# AQL Entities

AQL supports querying the following Aha.io entities.

## Core Entities

### features

Product features and their details.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Feature ID |
| `reference_num` | string | Yes | Reference number (e.g., FEAT-123) |
| `name` | string | Yes | Feature name |
| `description` | string | No | Feature description (HTML) |
| `status` | string | Yes | Workflow status name |
| `workflow_status_id` | string | Yes | Workflow status ID |
| `created_at` | datetime | Yes | Creation timestamp |
| `updated_at` | datetime | Yes | Last update timestamp |
| `start_date` | date | Yes | Start date |
| `due_date` | date | Yes | Due date |
| `release_id` | string | Yes | Release ID |
| `release_date` | date | Yes | Release target date (from associated release) |
| `position` | integer | Yes | Feature rank/position |
| `workspace` | string | Yes | Workspace/product name |
| `tag_list` | string | Yes | Comma-separated tags |
| `assigned_to_user` | string | Yes | Assigned user |
| `tags` | array | No | Tags list |

**Examples:**

```sql
-- Basic query
SELECT name, status, created_at FROM features WHERE status = 'In Progress' LIMIT 10

-- Query by release ID
FROM features WHERE release_id = 'PROD-R-123'

-- Query by release date
FROM features WHERE release_date = '2024-10-31'

-- Query by release date range
FROM features WHERE release_date >= '2024-01-01' AND release_date <= '2024-06-30'

-- Order by release date
SELECT name, release_id, release_date FROM features ORDER BY release_date DESC LIMIT 10

-- Order by rank (position)
SELECT reference_num, name, position, tag_list, workspace FROM features ORDER BY position ASC

-- Export to Excel ordered by rank
SELECT reference_num, name, position, tag_list, workspace FROM features ORDER BY position ASC
```

**Note:** Release-based queries (`release_id`, `release_date`), rank (`position`), tags (`tag_list`), and workspace require syncing with GraphQL support. Run `aha-studio sync --product PROD` to populate this data.

### ideas

Customer ideas and feedback.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Idea ID |
| `reference_num` | string | Yes | Reference number (e.g., IDEA-456) |
| `name` | string | Yes | Idea name |
| `description` | string | No | Idea description (HTML) |
| `status` | string | Yes | Workflow status name |
| `votes_count` | integer | Yes | Number of votes |
| `created_at` | datetime | Yes | Creation timestamp |
| `updated_at` | datetime | Yes | Last update timestamp |
| `tags` | array | No | Tags list |

**Example:**

```sql
SELECT name, votes_count, status FROM ideas ORDER BY votes_count DESC LIMIT 10
```

### releases

Product releases and milestones.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Release ID |
| `reference_num` | string | Yes | Reference number (e.g., REL-1) |
| `name` | string | Yes | Release name |
| `release_date` | date | Yes | Target release date |
| `released_on` | date | Yes | Actual release date |
| `created_at` | datetime | Yes | Creation timestamp |
| `product_id` | string | Yes | Parent product ID |

**Example:**

```sql
FROM releases WHERE release_date >= '2024-01-01' ORDER BY release_date ASC LIMIT 5
```

### epics

Feature epics for grouping related features.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Epic ID |
| `reference_num` | string | Yes | Reference number |
| `name` | string | Yes | Epic name |
| `description` | string | No | Epic description |
| `created_at` | datetime | Yes | Creation timestamp |
| `product_id` | string | Yes | Parent product ID |

**Example:**

```sql
SELECT name, created_at FROM epics ORDER BY created_at DESC LIMIT 10
```

### initiatives

Strategic initiatives.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Initiative ID |
| `reference_num` | string | Yes | Reference number |
| `name` | string | Yes | Initiative name |
| `description` | string | No | Initiative description |
| `created_at` | datetime | Yes | Creation timestamp |

**Example:**

```sql
FROM initiatives LIMIT 10
```

### goals

Product goals and OKRs.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Goal ID |
| `reference_num` | string | Yes | Reference number |
| `name` | string | Yes | Goal name |
| `description` | string | No | Goal description |
| `created_at` | datetime | Yes | Creation timestamp |

**Example:**

```sql
FROM goals ORDER BY created_at DESC LIMIT 10
```

## Supporting Entities

### users

Workspace users.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | User ID |
| `name` | string | Yes | Full name |
| `email` | string | Yes | Email address |
| `role` | string | Yes | User role |
| `created_at` | datetime | Yes | Creation timestamp |

**Example:**

```sql
SELECT name, email, role FROM users LIMIT 20
```

### products

Products/workspaces.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Product ID |
| `reference_prefix` | string | Yes | Reference prefix (e.g., PROD) |
| `name` | string | Yes | Product name |
| `created_at` | datetime | Yes | Creation timestamp |

**Example:**

```sql
SELECT name, reference_prefix FROM products
```

### comments

Comments on entities.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Comment ID |
| `body` | string | No | Comment body (HTML) |
| `user_id` | string | Yes | Author user ID |
| `created_at` | datetime | Yes | Creation timestamp |

### requirements

Feature requirements.

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Requirement ID |
| `name` | string | Yes | Requirement name |
| `description` | string | No | Description |
| `feature_id` | string | Yes | Parent feature ID |

### tags

Entity tags (derived entity).

| Field | Type | Sortable | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Tag name |
| `feature_count` | integer | Yes | Count on features |
| `epic_count` | integer | Yes | Count on epics |

**Example:**

```sql
SELECT name, feature_count FROM tags ORDER BY feature_count DESC LIMIT 10
```

## Custom Fields

Custom fields are accessible with the `custom.` prefix:

```sql
-- Query by custom field
FROM features WHERE custom.priority = 'High'

-- Select custom fields
SELECT name, custom.priority, custom.team FROM features LIMIT 10

-- Order by custom field
FROM features ORDER BY custom.priority DESC

-- Group by custom field
SELECT custom.team, COUNT(*) FROM features GROUP BY custom.team
```

**Notes:**

- Custom field filtering is applied **client-side** (not pushed to Aha API)
- Queries referencing custom fields fetch full entity details, which may be slower
- Supported on `features` and `goals` entities
