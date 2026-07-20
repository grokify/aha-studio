// Package mcp provides MCP server integration for Aha Studio using OmniSkill.
package mcp

import (
	"context"

	"github.com/plexusone/omniskill/skill"
)

// AhaSkill implements the skill.Skill interface for Aha.io operations.
type AhaSkill struct {
	skill.BaseSkill
	handlers *ToolHandlers
}

// NewAhaSkill creates a new AhaSkill with the given configuration.
func NewAhaSkill(cfg *Config) *AhaSkill {
	return &AhaSkill{
		handlers: NewToolHandlers(cfg),
	}
}

// Name returns the skill identifier.
func (s *AhaSkill) Name() string {
	return "aha"
}

// Description returns a human-readable description of the skill.
func (s *AhaSkill) Description() string {
	return "Query and manage Aha.io data using AQL (Aha Query Language)"
}

// Init initializes the skill.
func (s *AhaSkill) Init(ctx context.Context) error {
	return s.handlers.Init(ctx)
}

// Close cleans up resources.
func (s *AhaSkill) Close() error {
	return nil
}

// Tools returns the tools provided by this skill.
func (s *AhaSkill) Tools() []skill.Tool {
	return []skill.Tool{
		// Query tool - execute AQL queries
		skill.NewTool("query", "Execute an AQL query against Aha.io",
			map[string]skill.Parameter{
				"query": {
					Type:        "string",
					Required:    true,
					Description: "AQL query string (e.g., 'FROM features WHERE status = \"In Progress\" LIMIT 10')",
				},
				"product": {
					Type:        "string",
					Required:    false,
					Description: "Product ID for context (required for releases)",
				},
			},
			s.handlers.Query),

		// Get feature by reference
		skill.NewTool("get_feature", "Get a feature by reference number",
			map[string]skill.Parameter{
				"reference": {
					Type:        "string",
					Required:    true,
					Description: "Feature reference number (e.g., FEAT-123)",
				},
			},
			s.handlers.GetFeature),

		// Get idea by reference
		skill.NewTool("get_idea", "Get an idea by reference number",
			map[string]skill.Parameter{
				"reference": {
					Type:        "string",
					Required:    true,
					Description: "Idea reference number (e.g., IDEA-456)",
				},
			},
			s.handlers.GetIdea),

		// Get release by reference
		skill.NewTool("get_release", "Get a release by reference number",
			map[string]skill.Parameter{
				"reference": {
					Type:        "string",
					Required:    true,
					Description: "Release reference number (e.g., REL-1)",
				},
			},
			s.handlers.GetRelease),

		// Get initiative by reference
		skill.NewTool("get_initiative", "Get an initiative by reference number",
			map[string]skill.Parameter{
				"reference": {
					Type:        "string",
					Required:    true,
					Description: "Initiative reference number (e.g., INIT-1)",
				},
			},
			s.handlers.GetInitiative),

		// Get epic by ID
		skill.NewTool("get_epic", "Get an epic by ID",
			map[string]skill.Parameter{
				"epic_id": {
					Type:        "string",
					Required:    true,
					Description: "Epic ID to retrieve",
				},
			},
			s.handlers.GetEpic),

		// Get goal by ID
		skill.NewTool("get_goal", "Get a goal by ID",
			map[string]skill.Parameter{
				"goal_id": {
					Type:        "string",
					Required:    true,
					Description: "Goal ID to retrieve",
				},
			},
			s.handlers.GetGoal),

		// Get comment by ID
		skill.NewTool("get_comment", "Get a comment by ID",
			map[string]skill.Parameter{
				"comment_id": {
					Type:        "string",
					Required:    true,
					Description: "Comment ID to retrieve",
				},
			},
			s.handlers.GetComment),

		// Get requirement by ID
		skill.NewTool("get_requirement", "Get a requirement by ID",
			map[string]skill.Parameter{
				"requirement_id": {
					Type:        "string",
					Required:    true,
					Description: "Requirement ID to retrieve",
				},
			},
			s.handlers.GetRequirement),

		// Get user by ID
		skill.NewTool("get_user", "Get a user by ID",
			map[string]skill.Parameter{
				"user_id": {
					Type:        "string",
					Required:    true,
					Description: "User ID to retrieve",
				},
			},
			s.handlers.GetUser),

		// Get key result by ID
		skill.NewTool("get_key_result", "Get a key result by ID",
			map[string]skill.Parameter{
				"key_result_id": {
					Type:        "string",
					Required:    true,
					Description: "Key result ID to retrieve",
				},
			},
			s.handlers.GetKeyResult),

		// Get persona by ID
		skill.NewTool("get_persona", "Get a persona by ID",
			map[string]skill.Parameter{
				"persona_id": {
					Type:        "string",
					Required:    true,
					Description: "Persona ID to retrieve",
				},
			},
			s.handlers.GetPersona),

		// Get team by ID
		skill.NewTool("get_team", "Get a team by ID",
			map[string]skill.Parameter{
				"team_id": {
					Type:        "string",
					Required:    true,
					Description: "Team ID to retrieve",
				},
			},
			s.handlers.GetTeam),

		// Get workflow by ID
		skill.NewTool("get_workflow", "Get a workflow by ID",
			map[string]skill.Parameter{
				"workflow_id": {
					Type:        "string",
					Required:    true,
					Description: "Workflow ID to retrieve",
				},
			},
			s.handlers.GetWorkflow),

		// List ideas with filtering
		skill.NewTool("list_ideas", "List ideas from Aha with optional filtering and pagination",
			map[string]skill.Parameter{
				"q": {
					Type:        "string",
					Required:    false,
					Description: "Search term to match against the idea name",
				},
				"spam": {
					Type:        "boolean",
					Required:    false,
					Description: "When true, shows ideas marked as spam",
				},
				"workflow_status": {
					Type:        "string",
					Required:    false,
					Description: "Filter by workflow status ID or name",
				},
				"sort": {
					Type:        "string",
					Required:    false,
					Description: "Sort by: recent, trending, or popular",
					Enum:        []any{"recent", "trending", "popular"},
				},
				"created_before": {
					Type:        "string",
					Required:    false,
					Description: "UTC timestamp (ISO8601). Only ideas created before this time",
				},
				"created_since": {
					Type:        "string",
					Required:    false,
					Description: "UTC timestamp (ISO8601). Only ideas created after this time",
				},
				"updated_since": {
					Type:        "string",
					Required:    false,
					Description: "UTC timestamp (ISO8601). Only ideas updated after this time",
				},
				"tag": {
					Type:        "string",
					Required:    false,
					Description: "Filter by tag value",
				},
				"page": {
					Type:        "integer",
					Required:    false,
					Description: "Page number",
				},
				"per_page": {
					Type:        "integer",
					Required:    false,
					Description: "Results per page",
				},
				"fields": {
					Type:        "array",
					Required:    false,
					Description: "Override the idea fields returned beyond id/name/reference_num/created_at/updated_at (Aha's list endpoint would otherwise omit votes/categories/score). Defaults to description, votes, categories, score, status_changed_at, workflow_status, feature, url, resource.",
				},
			},
			s.handlers.ListIdeas),

		// Search documents via GraphQL
		skill.NewTool("search_documents", "Search for Aha! documents using GraphQL",
			map[string]skill.Parameter{
				"query": {
					Type:        "string",
					Required:    true,
					Description: "Search query string",
				},
				"searchable_type": {
					Type:        "string",
					Required:    false,
					Description: "Type of document to search for (defaults to Page). Examples: Page, Feature, Epic, Release, etc.",
					Enum:        []any{"Page", "Feature", "Epic", "Release", "Initiative", "Idea", "Goal"},
				},
			},
			s.handlers.SearchDocuments),

		// List products
		skill.NewTool("list_products", "List all accessible Aha.io products",
			map[string]skill.Parameter{},
			s.handlers.ListProducts),

		// --- Phase 10a: Core Write Operations ---

		// List workflow statuses for a product
		skill.NewTool("list_workflow_statuses", "List all workflow statuses for a product (use to lookup status IDs by name)",
			map[string]skill.Parameter{
				"product_id": {
					Type:        "string",
					Required:    true,
					Description: "Product ID or reference prefix (e.g., 'PROD')",
				},
			},
			s.handlers.ListWorkflowStatuses),

		// List releases for a product
		skill.NewTool("list_releases", "List all releases for a product (use to lookup release IDs by name)",
			map[string]skill.Parameter{
				"product_id": {
					Type:        "string",
					Required:    true,
					Description: "Product ID or reference prefix (e.g., 'PROD')",
				},
			},
			s.handlers.ListReleases),

		// Create a new feature
		skill.NewTool("create_feature", "Create a new feature in Aha!",
			map[string]skill.Parameter{
				"release_id": {
					Type:        "string",
					Required:    true,
					Description: "Release ID or reference to create the feature in",
				},
				"name": {
					Type:        "string",
					Required:    true,
					Description: "Feature name",
				},
				"description": {
					Type:        "string",
					Required:    false,
					Description: "Feature description (HTML supported)",
				},
				"workflow_status": {
					Type:        "string",
					Required:    false,
					Description: "Initial workflow status ID or name",
				},
				"assigned_to_user": {
					Type:        "string",
					Required:    false,
					Description: "User ID or email to assign the feature to",
				},
			},
			s.handlers.CreateFeature),

		// Change feature status
		skill.NewTool("change_feature_status", "Change the workflow status of a feature",
			map[string]skill.Parameter{
				"feature_id": {
					Type:        "string",
					Required:    true,
					Description: "Feature ID or reference number (e.g., 'FEAT-123')",
				},
				"status": {
					Type:        "string",
					Required:    true,
					Description: "New workflow status ID or name",
				},
			},
			s.handlers.ChangeFeatureStatus),

		// Assign feature to release
		skill.NewTool("assign_feature_release", "Assign a feature to a different release",
			map[string]skill.Parameter{
				"feature_id": {
					Type:        "string",
					Required:    true,
					Description: "Feature ID or reference number (e.g., 'FEAT-123')",
				},
				"release_id": {
					Type:        "string",
					Required:    true,
					Description: "Release ID or reference to assign the feature to",
				},
			},
			s.handlers.AssignFeatureRelease),

		// Assign user to feature
		skill.NewTool("assign_user_to_feature", "Assign a user to a feature",
			map[string]skill.Parameter{
				"feature_id": {
					Type:        "string",
					Required:    true,
					Description: "Feature ID or reference number (e.g., 'FEAT-123')",
				},
				"user": {
					Type:        "string",
					Required:    true,
					Description: "User ID or email address to assign",
				},
			},
			s.handlers.AssignUserToFeature),

		// Add comment to feature
		skill.NewTool("add_feature_comment", "Add a comment to a feature",
			map[string]skill.Parameter{
				"feature_id": {
					Type:        "string",
					Required:    true,
					Description: "Feature ID or reference number (e.g., 'FEAT-123')",
				},
				"body": {
					Type:        "string",
					Required:    true,
					Description: "Comment body (HTML supported)",
				},
			},
			s.handlers.AddFeatureComment),

		// Add comment to idea
		skill.NewTool("add_idea_comment", "Add a comment to an idea",
			map[string]skill.Parameter{
				"idea_id": {
					Type:        "string",
					Required:    true,
					Description: "Idea ID or reference number (e.g., 'IDEA-456')",
				},
				"body": {
					Type:        "string",
					Required:    true,
					Description: "Comment body (HTML supported)",
				},
			},
			s.handlers.AddIdeaComment),

		// Describe AQL syntax
		skill.NewTool("describe_aql", "Get AQL syntax help and examples",
			map[string]skill.Parameter{
				"topic": {
					Type:        "string",
					Required:    false,
					Description: "Specific topic: 'entities', 'operators', 'aggregates', 'joins', or 'examples'",
					Enum:        []any{"entities", "operators", "aggregates", "joins", "examples"},
				},
			},
			s.handlers.DescribeAQL),

		// Browser automation tools
		skill.NewTool("list_predefined_templates", "List all predefined strategic model templates available for browser automation",
			map[string]skill.Parameter{},
			s.handlers.ListPredefinedTemplates),

		skill.NewTool("browser_create_template", "Create a strategic model template in Aha! using browser automation (requires AHA_EMAIL and AHA_PASSWORD)",
			map[string]skill.Parameter{
				"product_key": {
					Type:        "string",
					Required:    true,
					Description: "Aha product key (e.g., 'PROD')",
				},
				"template_type": {
					Type:        "string",
					Required:    true,
					Description: "Template type: capability-stack, maturity-model, opportunity-patton, feature-canvas, business-model, lean-canvas, value-proposition, opportunity-solution-tree",
					Enum:        []any{"capability-stack", "maturity-model", "opportunity-patton", "feature-canvas", "business-model", "lean-canvas", "value-proposition", "opportunity-solution-tree"},
				},
				"custom_name": {
					Type:        "string",
					Required:    false,
					Description: "Optional custom name for the template",
				},
				"headless": {
					Type:        "boolean",
					Required:    false,
					Description: "Run browser in headless mode (default: true)",
				},
			},
			s.handlers.BrowserCreateTemplate),

		// Graph (Neo4j) tools
		skill.NewTool("graph_sync", "Sync Aha data to Neo4j graph database for relationship queries (requires NEO4J_* env vars)",
			map[string]skill.Parameter{
				"product": {
					Type:        "string",
					Required:    false,
					Description: "Product ID to sync (uses default if not specified)",
				},
				"entities": {
					Type:        "array",
					Required:    false,
					Description: "Specific entities to sync: products, users, releases, features, epics, initiatives, goals, ideas",
				},
			},
			s.handlers.GraphSync),

		skill.NewTool("graph_query", "Execute a Cypher query against Neo4j graph database",
			map[string]skill.Parameter{
				"cypher": {
					Type:        "string",
					Required:    true,
					Description: "Cypher query to execute (e.g., 'MATCH (f:Feature)-[:IN_RELEASE]->(r:Release) RETURN f.name, r.name LIMIT 10')",
				},
				"params": {
					Type:        "object",
					Required:    false,
					Description: "Query parameters as key-value pairs",
				},
			},
			s.handlers.GraphQuery),

		skill.NewTool("graph_find_path", "Find the shortest path between two entities in the graph",
			map[string]skill.Parameter{
				"from_type": {
					Type:        "string",
					Required:    true,
					Description: "Source node type: Feature, Release, Epic, Initiative, Goal, Idea, Product, User",
					Enum:        []any{"Feature", "Release", "Epic", "Initiative", "Goal", "Idea", "Product", "User"},
				},
				"from_id": {
					Type:        "string",
					Required:    true,
					Description: "Source node ID",
				},
				"to_type": {
					Type:        "string",
					Required:    true,
					Description: "Target node type",
					Enum:        []any{"Feature", "Release", "Epic", "Initiative", "Goal", "Idea", "Product", "User"},
				},
				"to_id": {
					Type:        "string",
					Required:    true,
					Description: "Target node ID",
				},
			},
			s.handlers.GraphFindPath),

		skill.NewTool("graph_search", "Full-text search across graph entities",
			map[string]skill.Parameter{
				"query": {
					Type:        "string",
					Required:    true,
					Description: "Search query string",
				},
				"entity_types": {
					Type:        "array",
					Required:    false,
					Description: "Entity types to search: Feature, Idea, Initiative, Epic (searches all if not specified)",
				},
			},
			s.handlers.GraphSearch),

		skill.NewTool("graph_initiative_impact", "Get the impact analysis of an initiative (features, epics, goals, releases)",
			map[string]skill.Parameter{
				"initiative_id": {
					Type:        "string",
					Required:    true,
					Description: "Initiative ID to analyze",
				},
			},
			s.handlers.GraphInitiativeImpact),

		skill.NewTool("graph_release_deps", "Get feature dependencies for a release",
			map[string]skill.Parameter{
				"release_id": {
					Type:        "string",
					Required:    true,
					Description: "Release ID to analyze",
				},
			},
			s.handlers.GraphReleaseDeps),
	}
}
