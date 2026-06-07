# Tools Reference

The MCP server provides 34 tools for accessing, querying, and managing Aha.io data.

## Query Tools

### query

Execute an AQL query against Aha.io.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | AQL query string (e.g., 'FROM features WHERE status = "In Progress" LIMIT 10') |
| `product` | string | No | Product ID for context (required for releases) |

**Example:**

```json
{
  "tool": "query",
  "arguments": {
    "query": "FROM features WHERE status = 'In Progress' LIMIT 10"
  }
}
```

### describe_aql

Get AQL syntax help and examples.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `topic` | string | No | Specific topic: 'entities', 'operators', 'aggregates', 'joins', or 'examples' |

## Get Tools

All get tools retrieve a single object by ID or reference number.

### get_feature

Retrieve a feature by reference number.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `reference` | string | Yes | Feature reference number (e.g., FEAT-123) |

### get_idea

Retrieve an idea by reference number.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `reference` | string | Yes | Idea reference number (e.g., IDEA-456) |

### get_release

Retrieve a release by reference number.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `reference` | string | Yes | Release reference number (e.g., REL-1) |

### get_initiative

Retrieve an initiative by reference number.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `reference` | string | Yes | Initiative reference number (e.g., INIT-1) |

### get_epic

Retrieve an epic by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic_id` | string | Yes | Epic ID to retrieve |

### get_goal

Retrieve a goal by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `goal_id` | string | Yes | Goal ID to retrieve |

### get_comment

Retrieve a comment by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `comment_id` | string | Yes | Comment ID to retrieve |

### get_requirement

Retrieve a requirement by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `requirement_id` | string | Yes | Requirement ID to retrieve |

### get_user

Retrieve a user by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user_id` | string | Yes | User ID to retrieve |

### get_key_result

Retrieve a key result by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key_result_id` | string | Yes | Key result ID to retrieve |

### get_persona

Retrieve a persona by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `persona_id` | string | Yes | Persona ID to retrieve |

### get_team

Retrieve a team by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `team_id` | string | Yes | Team ID to retrieve |

### get_workflow

Retrieve a workflow by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `workflow_id` | string | Yes | Workflow ID to retrieve |

## List Tools

### list_ideas

List ideas from Aha with optional filtering and pagination.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | No | Search term to match against the idea name |
| `spam` | boolean | No | When true, shows ideas marked as spam |
| `workflow_status` | string | No | Filter by workflow status ID or name |
| `sort` | string | No | Sort by: recent, trending, or popular |
| `created_before` | string | No | UTC timestamp (ISO8601). Only ideas created before this time |
| `created_since` | string | No | UTC timestamp (ISO8601). Only ideas created after this time |
| `updated_since` | string | No | UTC timestamp (ISO8601). Only ideas updated after this time |
| `tag` | string | No | Filter by tag value |
| `page` | integer | No | Page number |
| `per_page` | integer | No | Results per page |

### list_products

List all accessible Aha.io products.

### list_workflow_statuses

List all workflow statuses for a product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix (e.g., 'PROD') |

### list_releases

List all releases for a product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix (e.g., 'PROD') |

### search_documents

Search for Aha! documents using GraphQL.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query string |
| `searchable_type` | string | No | Type of document to search for (defaults to Page) |

## Write Tools

### create_feature

Create a new feature in Aha!

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `release_id` | string | Yes | Release ID or reference to create the feature in |
| `name` | string | Yes | Feature name |
| `description` | string | No | Feature description (HTML supported) |
| `workflow_status` | string | No | Initial workflow status ID or name |
| `assigned_to_user` | string | No | User ID or email to assign the feature to |

### change_feature_status

Change the workflow status of a feature.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number (e.g., 'FEAT-123') |
| `status` | string | Yes | New workflow status ID or name |

### assign_feature_release

Assign a feature to a different release.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number (e.g., 'FEAT-123') |
| `release_id` | string | Yes | Release ID or reference to assign the feature to |

### assign_user_to_feature

Assign a user to a feature.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number (e.g., 'FEAT-123') |
| `user` | string | Yes | User ID or email address to assign |

### add_feature_comment

Add a comment to a feature.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number (e.g., 'FEAT-123') |
| `body` | string | Yes | Comment body (HTML supported) |

### add_idea_comment

Add a comment to an idea.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `idea_id` | string | Yes | Idea ID or reference number (e.g., 'IDEA-456') |
| `body` | string | Yes | Comment body (HTML supported) |

## Browser Tools

### list_predefined_templates

List all predefined strategic model templates available for browser automation.

### browser_create_template

Create a strategic model template in Aha! using browser automation.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_key` | string | Yes | Aha product key (e.g., 'PROD') |
| `template_type` | string | Yes | Template type: capability-stack, maturity-model, opportunity-patton, feature-canvas, business-model, lean-canvas, value-proposition, opportunity-solution-tree |
| `custom_name` | string | No | Optional custom name for the template |
| `headless` | boolean | No | Run browser in headless mode (default: true) |

!!! note
    Browser automation requires `AHA_EMAIL` and `AHA_PASSWORD` environment variables.

## Graph Tools (Neo4j)

Graph tools require Neo4j to be configured with `NEO4J_URI`, `NEO4J_USERNAME`, and `NEO4J_PASSWORD` environment variables.

### graph_sync

Sync Aha data to Neo4j graph database for relationship queries.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product` | string | No | Product ID to sync (uses default if not specified) |
| `entities` | array | No | Specific entities to sync: products, users, releases, features, epics, initiatives, goals, ideas |

### graph_query

Execute a Cypher query against Neo4j graph database.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `cypher` | string | Yes | Cypher query to execute |
| `params` | object | No | Query parameters as key-value pairs |

**Example:**

```json
{
  "tool": "graph_query",
  "arguments": {
    "cypher": "MATCH (f:Feature)-[:IN_RELEASE]->(r:Release) RETURN f.name, r.name LIMIT 10"
  }
}
```

### graph_find_path

Find the shortest path between two entities in the graph.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `from_type` | string | Yes | Source node type: Feature, Release, Epic, Initiative, Goal, Idea, Product, User |
| `from_id` | string | Yes | Source node ID |
| `to_type` | string | Yes | Target node type |
| `to_id` | string | Yes | Target node ID |

### graph_search

Full-text search across graph entities.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query string |
| `entity_types` | array | No | Entity types to search: Feature, Idea, Initiative, Epic (searches all if not specified) |

### graph_initiative_impact

Get the impact analysis of an initiative (features, epics, goals, releases).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `initiative_id` | string | Yes | Initiative ID to analyze |

### graph_release_deps

Get feature dependencies for a release.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `release_id` | string | Yes | Release ID to analyze |

## Response Format

All tools return JSON data including the requested object and any relevant metadata.

**Example Response:**

```json
{
  "id": "6789012345",
  "reference_num": "FEAT-123",
  "name": "User Authentication",
  "status": "In Development",
  "description": "Implement OAuth2 authentication..."
}
```
