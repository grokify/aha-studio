# Tools Reference

The MCP server provides 71 tools for accessing, querying, and managing Aha.io data.

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

### get_current_user

Retrieve the authenticated user.

**Parameters:** None

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

### get_strategic_model

Retrieve a strategic model by ID.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `model_id` | string | Yes | Strategic model ID |

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
| `fields` | array | No | Override the idea fields returned beyond id/name/reference_num/created_at/updated_at. Defaults to description, votes, categories, score, status_changed_at, workflow_status, feature, url, resource |

### list_products

List all accessible Aha.io products.

### list_features

List features with optional filtering.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | No | Search query to filter features |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_release_features

List features for a specific release.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `release_id` | string | Yes | Release ID or reference number |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_epics

List epics with optional filtering.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | No | Search query to filter epics |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_product_epics

List epics for a specific product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_goals

List goals with optional filtering.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | No | Search query to filter goals |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_product_goals

List goals for a specific product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_initiatives

List initiatives with optional filtering.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `q` | string | No | Search query to filter initiatives |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_product_initiatives

List initiatives for a specific product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_feature_requirements

List requirements for a specific feature.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_users

List workspace users.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

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

### list_custom_fields

List custom field definitions for a product or all products.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | No | Product ID or reference prefix to filter by (lists all if not specified) |
| `entity_type` | string | No | Filter by entity type: Feature, Initiative, Epic, Idea, Goal, Release, Requirement |

### list_custom_field_options

List options for a select/choice custom field.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `field_id` | string | Yes | Custom field definition ID |

### list_strategic_models

List strategic models.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_product_strategic_models

List strategic models for a specific product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### search_documents

Search for Aha! documents using GraphQL.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query string |
| `searchable_type` | string | No | Type of document to search for (defaults to Page) |

## Create Tools

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

### create_epic

Create a new epic in a release.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `release_id` | string | Yes | Release ID or reference to create the epic in |
| `name` | string | Yes | Epic name |
| `description` | string | No | Epic description (HTML supported) |
| `workflow_status` | string | No | Initial workflow status ID or name |
| `start_date` | string | No | Start date (YYYY-MM-DD format) |
| `due_date` | string | No | Due date (YYYY-MM-DD format) |
| `color` | string | No | Color hex code (e.g., '#ff0000') |
| `initiative` | string | No | Initiative ID or reference to link |

### create_goal

Create a new goal in a product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `name` | string | Yes | Goal name |
| `description` | string | No | Goal description (HTML supported) |
| `workflow_status` | string | No | Initial workflow status ID or name |
| `start_date` | string | No | Start date (YYYY-MM-DD format) |
| `end_date` | string | No | End date (YYYY-MM-DD format) |

### create_initiative

Create a new initiative in a product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `name` | string | Yes | Initiative name |
| `description` | string | No | Initiative description (HTML supported) |
| `workflow_status` | string | No | Initial workflow status ID or name |
| `start_date` | string | No | Start date (YYYY-MM-DD format) |
| `end_date` | string | No | End date (YYYY-MM-DD format) |
| `value` | number | No | Value score |
| `effort` | number | No | Effort score |
| `color` | string | No | Color hex code (e.g., '#ff0000') |

### create_requirement

Create a new requirement for a feature.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number to add the requirement to |
| `name` | string | Yes | Requirement name |
| `description` | string | No | Requirement description (HTML supported) |
| `workflow_status` | string | No | Initial workflow status ID or name |
| `assigned_to_user` | string | No | User ID or email to assign |
| `original_estimate` | number | No | Original time estimate |

### create_product

Create a new product (workspace).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Product name |
| `reference_prefix` | string | Yes | Reference prefix (e.g., 'PROD') |
| `description` | string | No | Product description |
| `parent_id` | string | No | Parent product ID for nested products |

### create_strategic_model

Create a new strategic model.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `kind` | string | Yes | Model type (e.g., 'canvas', 'scorecard') |
| `name` | string | No | Model name |

## Update Tools

### update_feature

Update a feature's fields (general update tool).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number (e.g., 'FEAT-123') |
| `name` | string | No | New name for the feature |
| `description` | string | No | New description (HTML supported) |
| `workflow_status` | string | No | New workflow status ID or name |
| `assigned_to_user` | string | No | User ID or email to assign |
| `release` | string | No | Release ID or reference to assign |
| `initiative` | string | No | Initiative ID or reference to link |
| `tags` | string | No | Comma-separated tags |
| `start_date` | string | No | Start date (YYYY-MM-DD format) |
| `due_date` | string | No | Due date (YYYY-MM-DD format) |

### update_epic

Update an epic's fields (name, description, status, progress).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic_id` | string | Yes | Epic ID or reference number |
| `name` | string | No | New name for the epic |
| `description` | string | No | New description (HTML supported) |
| `workflow_status` | string | No | New workflow status ID or name |
| `progress` | number | No | Progress percentage (0-100) |

### update_goal

Update a goal's fields (name, description, status, progress).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `goal_id` | string | Yes | Goal ID or reference number |
| `name` | string | No | New name for the goal |
| `description` | string | No | New description (HTML supported) |
| `workflow_status` | string | No | New workflow status ID or name |
| `progress` | number | No | Progress percentage (0-100) |

### update_initiative

Update an initiative's fields (name, description, status, dates, etc.).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `initiative_id` | string | Yes | Initiative ID or reference number (e.g., 'INIT-1') |
| `name` | string | No | New name for the initiative |
| `description` | string | No | New description (HTML supported) |
| `workflow_status` | string | No | New workflow status ID or name |
| `start_date` | string | No | Start date (YYYY-MM-DD format) |
| `end_date` | string | No | End date (YYYY-MM-DD format) |
| `value` | number | No | Value score |
| `effort` | number | No | Effort score |
| `color` | string | No | Color hex code (e.g., '#ff0000') |
| `presented` | boolean | No | Whether the initiative is presented |

### update_requirement

Update a requirement's fields (name, description, status, work_done).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `requirement_id` | string | Yes | Requirement ID or reference number |
| `name` | string | No | New name for the requirement |
| `description` | string | No | New description (HTML supported) |
| `workflow_status` | string | No | New workflow status ID or name |
| `work_done` | number | No | Work done percentage (0-100) |

### update_release

Update a release's fields (name, dates, parking_lot).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `release_id` | string | Yes | Release ID or reference number |
| `name` | string | No | New name for the release |
| `start_date` | string | No | Start date (YYYY-MM-DD format) |
| `release_date` | string | No | Release date (YYYY-MM-DD format) |
| `parking_lot` | boolean | No | Whether this is a parking lot release |
| `description` | string | No | New description for the release (HTML supported). Aha calls this field 'theme' internally. |

### update_idea

Update an idea's fields (name, description, status, visibility).

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `idea_id` | string | Yes | Idea ID or reference number (e.g., 'IDEA-123') |
| `name` | string | No | New name for the idea |
| `description` | string | No | New description (HTML supported) |
| `workflow_status` | string | No | New workflow status ID or name |
| `visibility` | string | No | Visibility (e.g., public, private) |

### update_product

Update an existing product.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product ID or reference prefix |
| `name` | string | No | New product name |
| `description` | string | No | New product description |
| `reference_prefix` | string | No | New reference prefix |

### update_strategic_model

Update a strategic model.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `model_id` | string | Yes | Strategic model ID |
| `name` | string | No | New model name |

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

## Comment Tools

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

### update_comment

Update an existing comment.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `comment_id` | string | Yes | Comment ID |
| `body` | string | Yes | New comment body (HTML supported) |

### list_feature_comments

List comments for a feature.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `feature_id` | string | Yes | Feature ID or reference number |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_idea_comments

List comments for an idea.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `idea_id` | string | Yes | Idea ID or reference number |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

### list_epic_comments

List comments for an epic.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `epic_id` | string | Yes | Epic ID or reference number |
| `page` | number | No | Page number for pagination |
| `per_page` | number | No | Results per page (max 200) |

## Delete Tools

### delete_requirement

Delete a requirement.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `requirement_id` | string | Yes | Requirement ID or reference number |

### delete_comment

Delete a comment.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `comment_id` | string | Yes | Comment ID |

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
