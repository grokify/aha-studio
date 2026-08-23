package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	genql "github.com/Khan/genqlient/graphql"
	aha "github.com/grokify/aha-go"
	ahagql "github.com/grokify/aha-go/graphql"
	"github.com/grokify/aha-go/graphql/generated"
	"github.com/grokify/aha-studio/aql/parser"
	"github.com/grokify/aha-studio/aql/validator"
	"github.com/grokify/aha-studio/browser"
	"github.com/grokify/aha-studio/executor"
	"github.com/grokify/aha-studio/graph"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/sync"
)

// ToolHandlers provides handler functions for Aha MCP tools.
type ToolHandlers struct {
	config        *Config
	client        *aha.Client
	executor      *executor.Executor
	graphClient   *graph.Client
	graphqlClient genql.Client
	syncDB        *sync.DB
	syncer        *sync.Syncer
}

// NewToolHandlers creates a new ToolHandlers instance.
func NewToolHandlers(cfg *Config) *ToolHandlers {
	return &ToolHandlers{
		config: cfg,
	}
}

// Init initializes the handlers with an Aha client.
func (h *ToolHandlers) Init(ctx context.Context) error {
	client, err := h.config.NewClient()
	if err != nil {
		return fmt.Errorf("creating Aha client: %w", err)
	}
	h.client = client
	h.graphqlClient = ahagql.NewGenqlientClient(h.config.Subdomain, h.config.APIKey)
	h.executor = executor.New(client).WithGraphQL(h.graphqlClient)

	// Initialize graph client if Neo4j is configured
	graphCfg := graph.ConfigFromEnv()
	if graphCfg.Password != "" {
		graphClient, err := graph.NewClient(graphCfg)
		if err != nil {
			// Log warning but don't fail - graph is optional
			fmt.Printf("Warning: Neo4j connection failed: %v\n", err)
		} else {
			h.graphClient = graphClient
		}
	}

	// Initialize sync DB if configured
	if h.config.DBPath != "" {
		db, err := sync.Open(h.config.DBPath)
		if err != nil {
			// Log warning but don't fail - sync DB is optional
			fmt.Printf("Warning: Sync DB connection failed: %v\n", err)
		} else {
			h.syncDB = db

			// Sync uses its own throttled client, separate from h.client,
			// so bulk sync_data calls don't add latency to every other tool.
			syncClient, err := h.config.NewSyncClient()
			if err != nil {
				fmt.Printf("Warning: creating sync client failed: %v\n", err)
			} else {
				h.syncer = sync.NewSyncerWithGraphQL(db, syncClient, h.config.Subdomain, h.config.APIKey)
			}
		}
	}

	return nil
}

// Query executes an AQL query and returns results.
func (h *ToolHandlers) Query(ctx context.Context, params map[string]any) (any, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	productID, _ := params["product"].(string)
	if productID == "" {
		productID = h.config.DefaultProduct
	}

	// Parse the query
	p := parser.New(query)
	ast, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Validate
	v := validator.New()
	if err := v.Validate(ast); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Plan
	pl := planner.New()
	plan := pl.Plan(ast)

	// Set product context
	if productID != "" {
		plan.APIParams.ProductID = productID
	}

	// Execute
	result, err := h.executor.Execute(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("execution error: %w", err)
	}

	return map[string]any{
		"entity":  string(result.Entity),
		"count":   len(result.Records),
		"records": result.Records,
	}, nil
}

// GetFeature retrieves a feature by reference number.
func (h *ToolHandlers) GetFeature(ctx context.Context, params map[string]any) (any, error) {
	reference, ok := params["reference"].(string)
	if !ok || reference == "" {
		return nil, fmt.Errorf("reference parameter is required")
	}

	feature, err := h.client.GetFeature(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("getting feature %s: %w", reference, err)
	}

	return featureToMap(feature), nil
}

// GetIdea retrieves an idea by reference number.
func (h *ToolHandlers) GetIdea(ctx context.Context, params map[string]any) (any, error) {
	reference, ok := params["reference"].(string)
	if !ok || reference == "" {
		return nil, fmt.Errorf("reference parameter is required")
	}

	idea, err := h.client.GetIdea(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("getting idea %s: %w", reference, err)
	}

	return ideaToMap(idea), nil
}

// GetRelease retrieves a release by reference number.
func (h *ToolHandlers) GetRelease(ctx context.Context, params map[string]any) (any, error) {
	reference, ok := params["reference"].(string)
	if !ok || reference == "" {
		return nil, fmt.Errorf("reference parameter is required")
	}

	release, err := h.client.GetRelease(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("getting release %s: %w", reference, err)
	}

	return releaseToMap(release), nil
}

// GetInitiative retrieves an initiative by reference number.
func (h *ToolHandlers) GetInitiative(ctx context.Context, params map[string]any) (any, error) {
	reference, ok := params["reference"].(string)
	if !ok || reference == "" {
		return nil, fmt.Errorf("reference parameter is required")
	}

	initiative, err := h.client.GetInitiative(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("getting initiative %s: %w", reference, err)
	}

	return initiativeToMap(initiative), nil
}

// GetEpic retrieves an epic by ID.
func (h *ToolHandlers) GetEpic(ctx context.Context, params map[string]any) (any, error) {
	epicID, ok := params["epic_id"].(string)
	if !ok || epicID == "" {
		return nil, fmt.Errorf("epic_id parameter is required")
	}
	return h.getResourceByID(ctx, "epics", epicID, "epic")
}

// GetGoal retrieves a goal by ID.
func (h *ToolHandlers) GetGoal(ctx context.Context, params map[string]any) (any, error) {
	goalID, ok := params["goal_id"].(string)
	if !ok || goalID == "" {
		return nil, fmt.Errorf("goal_id parameter is required")
	}
	return h.getResourceByID(ctx, "goals", goalID, "goal")
}

// GetComment retrieves a comment by ID.
func (h *ToolHandlers) GetComment(ctx context.Context, params map[string]any) (any, error) {
	commentID, ok := params["comment_id"].(string)
	if !ok || commentID == "" {
		return nil, fmt.Errorf("comment_id parameter is required")
	}
	return h.getResourceByID(ctx, "comments", commentID, "comment")
}

// GetRequirement retrieves a requirement by ID.
func (h *ToolHandlers) GetRequirement(ctx context.Context, params map[string]any) (any, error) {
	requirementID, ok := params["requirement_id"].(string)
	if !ok || requirementID == "" {
		return nil, fmt.Errorf("requirement_id parameter is required")
	}
	return h.getResourceByID(ctx, "requirements", requirementID, "requirement")
}

// GetUser retrieves a user by ID.
func (h *ToolHandlers) GetUser(ctx context.Context, params map[string]any) (any, error) {
	userID, ok := params["user_id"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("user_id parameter is required")
	}
	return h.getResourceByID(ctx, "users", userID, "user")
}

// GetKeyResult retrieves a key result by ID.
func (h *ToolHandlers) GetKeyResult(ctx context.Context, params map[string]any) (any, error) {
	keyResultID, ok := params["key_result_id"].(string)
	if !ok || keyResultID == "" {
		return nil, fmt.Errorf("key_result_id parameter is required")
	}
	return h.getResourceByID(ctx, "key_results", keyResultID, "key_result")
}

// GetPersona retrieves a persona by ID.
func (h *ToolHandlers) GetPersona(ctx context.Context, params map[string]any) (any, error) {
	personaID, ok := params["persona_id"].(string)
	if !ok || personaID == "" {
		return nil, fmt.Errorf("persona_id parameter is required")
	}
	return h.getResourceByID(ctx, "personas", personaID, "persona")
}

// GetTeam retrieves a team by ID.
func (h *ToolHandlers) GetTeam(ctx context.Context, params map[string]any) (any, error) {
	teamID, ok := params["team_id"].(string)
	if !ok || teamID == "" {
		return nil, fmt.Errorf("team_id parameter is required")
	}
	return h.getResourceByID(ctx, "teams", teamID, "team")
}

// GetWorkflow retrieves a workflow by ID.
func (h *ToolHandlers) GetWorkflow(ctx context.Context, params map[string]any) (any, error) {
	workflowID, ok := params["workflow_id"].(string)
	if !ok || workflowID == "" {
		return nil, fmt.Errorf("workflow_id parameter is required")
	}
	return h.getResourceByID(ctx, "workflows", workflowID, "workflow")
}

// getResourceByID is a generic helper to fetch a resource by ID using the raw API.
// Note: DoRaw appends the path to BaseURL which already includes /api/v1,
// so we only need to provide the relative path (e.g., "/epics/AI-E-9").
func (h *ToolHandlers) getResourceByID(ctx context.Context, endpoint, id, resourceName string) (any, error) {
	resp, err := h.client.DoRaw(ctx, http.MethodGet, fmt.Sprintf("/%s/%s", endpoint, id), nil)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("error getting %s: %v", resourceName, err)}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("error reading response: %v", err)}, nil
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]any{
			resourceName:  string(body),
			"status_code": resp.StatusCode,
		}, nil
	}

	return map[string]any{
		resourceName:  result,
		"status_code": resp.StatusCode,
	}, nil
}

// ListIdeas lists ideas with optional filtering and pagination.
func (h *ToolHandlers) ListIdeas(ctx context.Context, params map[string]any) (any, error) {
	var opts []aha.ListIdeasOption

	if q, ok := params["q"].(string); ok && q != "" {
		opts = append(opts, aha.WithIdeaQuery(q))
	}
	if spam, ok := params["spam"].(bool); ok {
		opts = append(opts, aha.WithIdeaSpam(spam))
	}
	if ws, ok := params["workflow_status"].(string); ok && ws != "" {
		opts = append(opts, aha.WithIdeaStatus(ws))
	}
	if sort, ok := params["sort"].(string); ok && sort != "" {
		opts = append(opts, aha.WithIdeaSort(sort))
	}
	if cb, ok := params["created_before"].(string); ok && cb != "" {
		if t, err := time.Parse(time.RFC3339, cb); err == nil {
			opts = append(opts, aha.WithIdeaCreatedBefore(t))
		}
	}
	if cs, ok := params["created_since"].(string); ok && cs != "" {
		if t, err := time.Parse(time.RFC3339, cs); err == nil {
			opts = append(opts, aha.WithIdeaCreatedSince(t))
		}
	}
	if us, ok := params["updated_since"].(string); ok && us != "" {
		if t, err := time.Parse(time.RFC3339, us); err == nil {
			opts = append(opts, aha.WithIdeaUpdatedSince(t))
		}
	}
	if tag, ok := params["tag"].(string); ok && tag != "" {
		opts = append(opts, aha.WithIdeaTag(tag))
	}
	if page, ok := params["page"].(float64); ok {
		opts = append(opts, aha.WithIdeaPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok {
		opts = append(opts, aha.WithIdeaPerPage(int(perPage)))
	}
	if raw, ok := params["fields"].([]any); ok {
		var fields []string
		for _, v := range raw {
			if s, ok := v.(string); ok {
				fields = append(fields, s)
			}
		}
		if len(fields) > 0 {
			opts = append(opts, aha.WithIdeaFields(fields...))
		}
	}

	ideas, err := h.client.ListIdeas(ctx, opts...)
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Error listing ideas: %v", err),
		}, nil
	}

	return map[string]any{
		"ideas": ideas,
	}, nil
}

// SearchDocuments searches for Aha! documents using GraphQL.
func (h *ToolHandlers) SearchDocuments(ctx context.Context, params map[string]any) (any, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	searchableType := "Page"
	if st, ok := params["searchable_type"].(string); ok && st != "" {
		searchableType = st
	}

	resp, err := generated.SearchDocuments(ctx, h.graphqlClient, query, []string{searchableType})
	if err != nil {
		return map[string]any{"error": fmt.Sprintf("error making GraphQL request: %v", err)}, nil
	}

	results := make([]map[string]any, len(resp.SearchDocuments.Nodes))
	for i, node := range resp.SearchDocuments.Nodes {
		name := ""
		if node.Name != nil {
			name = *node.Name
		}
		searchableID := ""
		if node.SearchableId != nil {
			searchableID = *node.SearchableId
		}
		results[i] = map[string]any{
			"reference_num": searchableID,
			"name":          name,
			"type":          node.SearchableType,
			"url":           node.Url,
		}
	}

	return map[string]any{
		"results":       results,
		"total_results": resp.SearchDocuments.TotalCount,
		"current_page":  resp.SearchDocuments.CurrentPage,
		"total_pages":   resp.SearchDocuments.TotalPages,
		"is_last_page":  resp.SearchDocuments.IsLastPage,
	}, nil
}

// ListProducts lists all accessible Aha.io products.
func (h *ToolHandlers) ListProducts(ctx context.Context, params map[string]any) (any, error) {
	products, err := h.client.ListProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}

	var result []map[string]any
	for _, p := range products.Products {
		result = append(result, map[string]any{
			"id":           p.ID,
			"reference":    p.ReferencePrefix,
			"name":         p.Name,
			"product_line": p.ProductLine,
		})
	}

	return map[string]any{
		"count":    len(result),
		"products": result,
	}, nil
}

// DescribeAQL returns AQL syntax help.
func (h *ToolHandlers) DescribeAQL(ctx context.Context, params map[string]any) (any, error) {
	topic, _ := params["topic"].(string)

	switch topic {
	case "entities":
		return map[string]any{
			"topic":       "entities",
			"description": "AQL supports querying the following Aha.io entities",
			"entities": []map[string]string{
				{"name": "features", "description": "Product features"},
				{"name": "ideas", "description": "Customer ideas and feedback"},
				{"name": "releases", "description": "Product releases (requires product context)"},
				{"name": "initiatives", "description": "Strategic initiatives"},
			},
		}, nil

	case "operators":
		return map[string]any{
			"topic":       "operators",
			"description": "Comparison and logical operators",
			"comparison":  []string{"=", "!=", "<", "<=", ">", ">=", "IN", "NOT IN", "CONTAINS", "LIKE", "IS NULL", "IS NOT NULL"},
			"logical":     []string{"AND", "OR", "NOT"},
		}, nil

	case "aggregates":
		return map[string]any{
			"topic":       "aggregates",
			"description": "Aggregate functions for analytics",
			"functions": []map[string]string{
				{"name": "COUNT(*)", "description": "Count all records"},
				{"name": "COUNT(field)", "description": "Count non-null values"},
				{"name": "SUM(field)", "description": "Sum numeric values"},
				{"name": "AVG(field)", "description": "Average of numeric values"},
				{"name": "MIN(field)", "description": "Minimum value"},
				{"name": "MAX(field)", "description": "Maximum value"},
			},
			"example": "SELECT status, COUNT(*) FROM features GROUP BY status",
		}, nil

	case "joins":
		return map[string]any{
			"topic":       "joins",
			"description": "Join multiple entities",
			"types":       []string{"JOIN (INNER)", "LEFT JOIN", "RIGHT JOIN"},
			"example":     "FROM features f JOIN releases r ON f.release_id = r.id",
		}, nil

	case "examples":
		return map[string]any{
			"topic": "examples",
			"examples": []map[string]string{
				{"query": "FROM features LIMIT 10", "description": "Get first 10 features"},
				{"query": "FROM ideas WHERE votes > 5 ORDER BY votes DESC", "description": "Popular ideas"},
				{"query": "FROM features WHERE status = \"In Progress\"", "description": "Features in progress"},
				{"query": "SELECT status, COUNT(*) FROM features GROUP BY status", "description": "Feature count by status"},
				{"query": "FROM features WHERE updated_at >= now() - duration(\"7d\")", "description": "Recently updated features"},
			},
		}, nil

	default:
		return map[string]any{
			"description": "AQL (Aha Query Language) - SQL-like query language for Aha.io",
			"syntax":      "[SELECT fields] FROM entity [JOIN ...] [WHERE conditions] [GROUP BY fields] [HAVING condition] [ORDER BY field [ASC|DESC]] [LIMIT n]",
			"topics":      []string{"entities", "operators", "aggregates", "joins", "examples"},
			"hint":        "Use describe_aql with a topic parameter for detailed help",
		}, nil
	}
}

// Conversion helpers

func featureToMap(f *aha.Feature) map[string]any {
	m := map[string]any{
		"id":            f.ID,
		"reference_num": f.ReferenceNum,
		"name":          f.Name,
		"description":   f.Description,
		"url":           f.URL,
		"created_at":    f.CreatedAt,
	}
	if f.WorkflowStatus != nil {
		m["status"] = f.WorkflowStatus.Name
	}
	return m
}

func ideaToMap(i *aha.Idea) map[string]any {
	m := map[string]any{
		"id":            i.ID,
		"reference_num": i.ReferenceNum,
		"name":          i.Name,
		"description":   i.Description,
		"votes":         i.Votes,
		"created_at":    i.CreatedAt,
		"updated_at":    i.UpdatedAt,
	}
	if i.WorkflowStatus != nil {
		m["status"] = i.WorkflowStatus.Name
	}
	return m
}

func releaseToMap(r *aha.Release) map[string]any {
	m := map[string]any{
		"id":            r.ID,
		"reference_num": r.ReferenceNum,
		"name":          r.Name,
		"released":      r.Released,
		"parking_lot":   r.ParkingLot,
		"description":   r.Theme,
		"url":           r.URL,
	}
	if r.StartDate != nil {
		m["start_date"] = *r.StartDate
	}
	if r.ReleaseDate != nil {
		m["release_date"] = *r.ReleaseDate
	}
	return m
}

func initiativeToMap(i *aha.Initiative) map[string]any {
	m := map[string]any{
		"id":              i.ID,
		"reference_num":   i.ReferenceNum,
		"name":            i.Name,
		"description":     i.Description,
		"color":           i.Color,
		"position":        i.Position,
		"value":           i.Value,
		"effort":          i.Effort,
		"presented":       i.Presented,
		"progress":        i.Progress,
		"progress_source": i.ProgressSource,
		"url":             i.URL,
		"created_at":      i.CreatedAt,
	}
	if i.StartDate != nil {
		m["start_date"] = i.StartDate.Format("2006-01-02")
	}
	if i.EndDate != nil {
		m["end_date"] = i.EndDate.Format("2006-01-02")
	}
	if i.UpdatedAt != nil {
		m["updated_at"] = *i.UpdatedAt
	}
	if i.WorkflowStatus != nil {
		m["workflow_status"] = map[string]any{
			"id":   i.WorkflowStatus.ID,
			"name": i.WorkflowStatus.Name,
		}
	}
	if len(i.CustomFields) > 0 {
		fields := make([]map[string]any, len(i.CustomFields))
		for idx, f := range i.CustomFields {
			fields[idx] = map[string]any{
				"key":   f.Key,
				"name":  f.Name,
				"value": f.Value,
				"type":  f.Type,
			}
		}
		m["custom_fields"] = fields
	}
	if len(i.Features) > 0 {
		features := make([]map[string]any, len(i.Features))
		for idx, f := range i.Features {
			features[idx] = map[string]any{
				"id":            f.ID,
				"reference_num": f.ReferenceNum,
				"name":          f.Name,
				"url":           f.URL,
			}
		}
		m["features"] = features
	}
	return m
}

// Browser automation handlers

// ListPredefinedTemplates returns all available strategic model templates.
func (h *ToolHandlers) ListPredefinedTemplates(ctx context.Context, params map[string]any) (any, error) {
	templates := browser.ListPredefinedTemplates()

	var result []map[string]any
	for _, t := range templates {
		result = append(result, map[string]any{
			"type":        string(t.Type),
			"name":        t.Name,
			"description": t.Description,
			"row_count":   len(t.Rows),
		})
	}

	return map[string]any{
		"count":     len(result),
		"templates": result,
	}, nil
}

// BrowserCreateTemplate creates a strategic model template via browser automation.
func (h *ToolHandlers) BrowserCreateTemplate(ctx context.Context, params map[string]any) (any, error) {
	productKey, ok := params["product_key"].(string)
	if !ok || productKey == "" {
		return nil, fmt.Errorf("product_key parameter is required")
	}

	templateType, ok := params["template_type"].(string)
	if !ok || templateType == "" {
		return nil, fmt.Errorf("template_type parameter is required")
	}

	// Get predefined template
	templateConfig, ok := browser.GetPredefinedTemplate(browser.TemplateType(templateType))
	if !ok {
		return nil, fmt.Errorf("unknown template_type: %s. Use list_predefined_templates to see available templates", templateType)
	}

	// Override name if custom name provided
	if customName, ok := params["custom_name"].(string); ok && customName != "" {
		templateConfig.Name = customName
	}

	// Check browser credentials
	if h.config.BrowserEmail == "" || h.config.BrowserPassword == "" {
		return nil, fmt.Errorf("browser automation requires AHA_EMAIL and AHA_PASSWORD environment variables")
	}

	// Determine headless mode
	headless := true
	if h, ok := params["headless"].(bool); ok {
		headless = h
	}

	// Create browser client
	browserClient := browser.NewClient(browser.Config{
		Subdomain: h.config.Subdomain,
		Email:     h.config.BrowserEmail,
		Password:  h.config.BrowserPassword,
		Headless:  headless,
		Timeout:   30 * time.Second,
	})

	// Launch browser
	if err := browserClient.Launch(ctx); err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}
	defer func() { _ = browserClient.Close(ctx) }()

	// Create the template
	result, err := browserClient.CreateStrategicModelTemplate(ctx, productKey, templateConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	response := map[string]any{
		"success": result.Success,
		"url":     result.URL,
	}

	if result.Error != "" {
		response["error"] = result.Error
	}

	// Include screenshot as base64 if available
	if len(result.Screenshot) > 0 {
		response["screenshot_base64"] = base64.StdEncoding.EncodeToString(result.Screenshot)
	}

	return response, nil
}

// Graph handlers

// GraphSync syncs data to Neo4j graph database.
func (h *ToolHandlers) GraphSync(ctx context.Context, params map[string]any) (any, error) {
	if h.graphClient == nil {
		return nil, fmt.Errorf("Neo4j not configured. Set NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD environment variables")
	}

	productID, _ := params["product"].(string)
	if productID == "" {
		productID = h.config.DefaultProduct
	}
	if productID == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	// Get entities to sync
	var entities []string
	if e, ok := params["entities"].([]any); ok {
		for _, v := range e {
			if s, ok := v.(string); ok {
				entities = append(entities, s)
			}
		}
	}

	syncer := graph.NewSyncer(h.graphClient, h.client)
	results, err := syncer.SyncAll(ctx, graph.SyncOptions{
		Product:  productID,
		Entities: entities,
	})
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	var response []map[string]any
	for _, r := range results {
		rec := map[string]any{
			"entity":        r.Entity,
			"nodes_created": r.NodesCreated,
			"rels_created":  r.RelsCreated,
			"duration_ms":   r.Duration.Milliseconds(),
		}
		if r.Error != nil {
			rec["error"] = r.Error.Error()
		}
		response = append(response, rec)
	}

	return map[string]any{
		"product": productID,
		"results": response,
	}, nil
}

// SyncData syncs Aha data to the local SQLite cache used by the offline
// query tools (list_features_by_release_date, get_features_statistics, etc.).
// This blocks until the entire sync completes — for large or detailed
// syncs, that can take a while (detailed mode makes an additional per-record
// API call for features and initiatives).
func (h *ToolHandlers) SyncData(ctx context.Context, params map[string]any) (any, error) {
	if h.syncer == nil {
		return nil, fmt.Errorf("sync not configured: set AHA_DB_PATH (or use the default cache path)")
	}

	productID, _ := params["product"].(string)
	if productID == "" {
		productID = h.config.DefaultProduct
	}
	if productID == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	// Get entities to sync
	var entities []string
	if e, ok := params["entities"].([]any); ok {
		for _, v := range e {
			if s, ok := v.(string); ok {
				entities = append(entities, s)
			}
		}
	}

	detailed, _ := params["detailed"].(bool)
	incremental, _ := params["incremental"].(bool)

	results, err := h.syncer.SyncAll(ctx, sync.SyncOptions{
		Product:     productID,
		Entities:    entities,
		Detailed:    detailed,
		Incremental: incremental,
	})
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}

	var response []map[string]any
	totalRecords := 0
	for _, r := range results {
		rec := map[string]any{
			"entity":       r.Entity,
			"record_count": r.RecordCount,
			"duration_ms":  r.Duration.Milliseconds(),
		}
		if r.Error != nil {
			rec["error"] = r.Error.Error()
		} else {
			totalRecords += r.RecordCount
		}
		response = append(response, rec)
	}

	return map[string]any{
		"product":       productID,
		"detailed":      detailed,
		"total_records": totalRecords,
		"results":       response,
	}, nil
}

// GraphQuery executes a Cypher query against Neo4j.
func (h *ToolHandlers) GraphQuery(ctx context.Context, params map[string]any) (any, error) {
	if h.graphClient == nil {
		return nil, fmt.Errorf("Neo4j not configured. Set NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD environment variables")
	}

	cypher, ok := params["cypher"].(string)
	if !ok || cypher == "" {
		return nil, fmt.Errorf("cypher parameter is required")
	}

	// Extract query parameters
	queryParams := make(map[string]any)
	if p, ok := params["params"].(map[string]any); ok {
		queryParams = p
	}

	result, err := h.graphClient.RunCypher(ctx, cypher, queryParams)
	if err != nil {
		return nil, fmt.Errorf("cypher query failed: %w", err)
	}

	return map[string]any{
		"count":   len(result.Records),
		"records": result.Records,
	}, nil
}

// GraphFindPath finds the shortest path between two entities.
func (h *ToolHandlers) GraphFindPath(ctx context.Context, params map[string]any) (any, error) {
	if h.graphClient == nil {
		return nil, fmt.Errorf("Neo4j not configured. Set NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD environment variables")
	}

	fromType, _ := params["from_type"].(string)
	fromID, _ := params["from_id"].(string)
	toType, _ := params["to_type"].(string)
	toID, _ := params["to_id"].(string)

	if fromType == "" || fromID == "" || toType == "" || toID == "" {
		return nil, fmt.Errorf("from_type, from_id, to_type, and to_id parameters are required")
	}

	result, err := h.graphClient.FindPath(ctx,
		graph.NodeLabel(fromType), fromID,
		graph.NodeLabel(toType), toID)
	if err != nil {
		return nil, fmt.Errorf("find path failed: %w", err)
	}

	return map[string]any{
		"paths": result.Paths,
	}, nil
}

// GraphSearch performs full-text search across graph entities.
func (h *ToolHandlers) GraphSearch(ctx context.Context, params map[string]any) (any, error) {
	if h.graphClient == nil {
		return nil, fmt.Errorf("Neo4j not configured. Set NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD environment variables")
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	var entityTypes []graph.NodeLabel
	if types, ok := params["entity_types"].([]any); ok {
		for _, t := range types {
			if s, ok := t.(string); ok {
				entityTypes = append(entityTypes, graph.NodeLabel(s))
			}
		}
	}

	result, err := h.graphClient.FullTextSearch(ctx, query, entityTypes)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return map[string]any{
		"count":   len(result.Records),
		"records": result.Records,
	}, nil
}

// GraphInitiativeImpact returns the impact analysis of an initiative.
func (h *ToolHandlers) GraphInitiativeImpact(ctx context.Context, params map[string]any) (any, error) {
	if h.graphClient == nil {
		return nil, fmt.Errorf("Neo4j not configured. Set NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD environment variables")
	}

	initiativeID, ok := params["initiative_id"].(string)
	if !ok || initiativeID == "" {
		return nil, fmt.Errorf("initiative_id parameter is required")
	}

	result, err := h.graphClient.GetInitiativeImpact(ctx, initiativeID)
	if err != nil {
		return nil, fmt.Errorf("initiative impact query failed: %w", err)
	}

	if len(result.Records) > 0 {
		return result.Records[0], nil
	}
	return map[string]any{"initiative": nil}, nil
}

// GraphReleaseDeps returns feature dependencies for a release.
func (h *ToolHandlers) GraphReleaseDeps(ctx context.Context, params map[string]any) (any, error) {
	if h.graphClient == nil {
		return nil, fmt.Errorf("Neo4j not configured. Set NEO4J_URI, NEO4J_USERNAME, and NEO4J_PASSWORD environment variables")
	}

	releaseID, ok := params["release_id"].(string)
	if !ok || releaseID == "" {
		return nil, fmt.Errorf("release_id parameter is required")
	}

	result, err := h.graphClient.GetReleaseDependencies(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("release dependencies query failed: %w", err)
	}

	return map[string]any{
		"release_id": releaseID,
		"features":   result.Records,
	}, nil
}

// --- Phase 10a: Core Write Operations ---

// ListWorkflowStatuses lists all workflow statuses for a product.
func (h *ToolHandlers) ListWorkflowStatuses(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	workflowList, err := h.client.ListProductWorkflows(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}

	var workflows []map[string]any
	for _, w := range workflowList.Workflows {
		statuses := make([]map[string]any, len(w.Statuses))
		for i, s := range w.Statuses {
			statuses[i] = map[string]any{
				"id":       s.ID,
				"name":     s.Name,
				"position": s.Position,
				"complete": s.Complete,
				"color":    s.Color,
			}
		}
		workflows = append(workflows, map[string]any{
			"id":       w.ID,
			"name":     w.Name,
			"statuses": statuses,
		})
	}

	return map[string]any{
		"product_id": productID,
		"workflows":  workflows,
	}, nil
}

// ListReleases lists all releases for a product.
func (h *ToolHandlers) ListReleases(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	releaseList, err := h.client.ListProductReleases(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("listing releases: %w", err)
	}

	var releases []map[string]any
	for _, r := range releaseList.Releases {
		release := map[string]any{
			"id":            r.ID,
			"reference_num": r.ReferenceNum,
			"name":          r.Name,
			"released":      r.Released,
			"parking_lot":   r.ParkingLot,
		}
		if r.StartDate != nil {
			release["start_date"] = r.StartDate.Format(time.RFC3339)
		}
		if r.ReleaseDate != nil {
			release["release_date"] = r.ReleaseDate.Format(time.RFC3339)
		}
		releases = append(releases, release)
	}

	return map[string]any{
		"product_id": productID,
		"count":      len(releases),
		"releases":   releases,
	}, nil
}

// CreateFeature creates a new feature.
func (h *ToolHandlers) CreateFeature(ctx context.Context, params map[string]any) (any, error) {
	releaseID, ok := params["release_id"].(string)
	if !ok || releaseID == "" {
		return nil, fmt.Errorf("release_id parameter is required")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	var opts []aha.CreateFeatureOption
	opts = append(opts, aha.WithFeatureName(name))

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithFeatureDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithFeatureStatus(status))
	}

	if user, ok := params["assigned_to_user"].(string); ok && user != "" {
		opts = append(opts, aha.WithFeatureAssignedTo(user))
	}

	feature, err := h.client.CreateFeature(ctx, releaseID, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating feature: %w", err)
	}

	return map[string]any{
		"id":            feature.ID,
		"reference_num": feature.ReferenceNum,
		"name":          feature.Name,
		"url":           feature.URL,
		"message":       fmt.Sprintf("Feature %s created successfully", feature.ReferenceNum),
	}, nil
}

// ChangeFeatureStatus changes the workflow status of a feature.
//
//nolint:dupl // Similar pattern but updates different feature fields
func (h *ToolHandlers) ChangeFeatureStatus(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	status, ok := params["status"].(string)
	if !ok || status == "" {
		return nil, fmt.Errorf("status parameter is required")
	}

	feature, err := h.client.UpdateFeature(ctx, featureID, aha.WithUpdateFeatureStatus(status))
	if err != nil {
		return nil, fmt.Errorf("updating feature status: %w", err)
	}

	var statusName string
	if feature.WorkflowStatus != nil {
		statusName = feature.WorkflowStatus.Name
	}

	return map[string]any{
		"id":            feature.ID,
		"reference_num": feature.ReferenceNum,
		"name":          feature.Name,
		"status":        statusName,
		"message":       fmt.Sprintf("Feature %s status changed to '%s'", feature.ReferenceNum, statusName),
	}, nil
}

// AssignFeatureRelease assigns a feature to a release.
//
//nolint:dupl // Similar pattern but updates different feature fields
func (h *ToolHandlers) AssignFeatureRelease(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	releaseID, ok := params["release_id"].(string)
	if !ok || releaseID == "" {
		return nil, fmt.Errorf("release_id parameter is required")
	}

	feature, err := h.client.UpdateFeature(ctx, featureID, aha.WithUpdateFeatureRelease(releaseID))
	if err != nil {
		return nil, fmt.Errorf("assigning feature to release: %w", err)
	}

	var releaseName string
	if feature.Release != nil {
		releaseName = feature.Release.Name
	}

	return map[string]any{
		"id":            feature.ID,
		"reference_num": feature.ReferenceNum,
		"name":          feature.Name,
		"release":       releaseName,
		"message":       fmt.Sprintf("Feature %s assigned to release '%s'", feature.ReferenceNum, releaseName),
	}, nil
}

// AssignUserToFeature assigns a user to a feature.
func (h *ToolHandlers) AssignUserToFeature(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	user, ok := params["user"].(string)
	if !ok || user == "" {
		return nil, fmt.Errorf("user parameter is required")
	}

	feature, err := h.client.UpdateFeature(ctx, featureID, aha.WithUpdateFeatureAssignedToUser(user))
	if err != nil {
		return nil, fmt.Errorf("assigning user to feature: %w", err)
	}

	var assignedTo string
	if feature.AssignedTo != nil {
		assignedTo = feature.AssignedTo.Name()
	}

	return map[string]any{
		"id":            feature.ID,
		"reference_num": feature.ReferenceNum,
		"name":          feature.Name,
		"assigned_to":   assignedTo,
		"message":       fmt.Sprintf("Feature %s assigned to '%s'", feature.ReferenceNum, assignedTo),
	}, nil
}

// AddFeatureComment adds a comment to a feature.
func (h *ToolHandlers) AddFeatureComment(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	body, ok := params["body"].(string)
	if !ok || body == "" {
		return nil, fmt.Errorf("body parameter is required")
	}

	comment, err := h.client.CreateFeatureComment(ctx, featureID, aha.WithCommentBody(body))
	if err != nil {
		return nil, fmt.Errorf("adding feature comment: %w", err)
	}

	return map[string]any{
		"id":         comment.ID,
		"feature_id": featureID,
		"body":       comment.Body,
		"created_at": comment.CreatedAt.Format(time.RFC3339),
		"message":    fmt.Sprintf("Comment added to feature %s", featureID),
	}, nil
}

// AddIdeaComment adds a comment to an idea.
func (h *ToolHandlers) AddIdeaComment(ctx context.Context, params map[string]any) (any, error) {
	ideaID, ok := params["idea_id"].(string)
	if !ok || ideaID == "" {
		return nil, fmt.Errorf("idea_id parameter is required")
	}

	body, ok := params["body"].(string)
	if !ok || body == "" {
		return nil, fmt.Errorf("body parameter is required")
	}

	comment, err := h.client.CreateIdeaComment(ctx, ideaID, aha.WithCommentBody(body))
	if err != nil {
		return nil, fmt.Errorf("adding idea comment: %w", err)
	}

	return map[string]any{
		"id":         comment.ID,
		"idea_id":    ideaID,
		"body":       comment.Body,
		"created_at": comment.CreatedAt.Format(time.RFC3339),
		"message":    fmt.Sprintf("Comment added to idea %s", ideaID),
	}, nil
}

// UpdateInitiative updates an initiative's fields.
func (h *ToolHandlers) UpdateInitiative(ctx context.Context, params map[string]any) (any, error) {
	initiativeID, ok := params["initiative_id"].(string)
	if !ok || initiativeID == "" {
		return nil, fmt.Errorf("initiative_id parameter is required")
	}

	var opts []aha.UpdateInitiativeOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateInitiativeName(name))
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateInitiativeDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithUpdateInitiativeStatus(status))
	}

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithUpdateInitiativeStartDate(t))
	}

	if endDate, ok := params["end_date"].(string); ok && endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithUpdateInitiativeEndDate(t))
	}

	if value, ok := params["value"].(float64); ok {
		opts = append(opts, aha.WithUpdateInitiativeValue(value))
	}

	if effort, ok := params["effort"].(float64); ok {
		opts = append(opts, aha.WithUpdateInitiativeEffort(effort))
	}

	if color, ok := params["color"].(string); ok && color != "" {
		opts = append(opts, aha.WithUpdateInitiativeColor(color))
	}

	if presented, ok := params["presented"].(bool); ok {
		opts = append(opts, aha.WithUpdateInitiativePresented(presented))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	initiative, err := h.client.UpdateInitiative(ctx, initiativeID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating initiative: %w", err)
	}

	return map[string]any{
		"id":            initiative.ID,
		"reference_num": initiative.ReferenceNum,
		"name":          initiative.Name,
		"description":   initiative.Description,
		"url":           initiative.URL,
		"message":       fmt.Sprintf("Initiative %s updated successfully", initiative.ReferenceNum),
	}, nil
}

// ListCustomFieldDefinitions lists all custom field definitions, optionally filtered by product.
func (h *ToolHandlers) ListCustomFieldDefinitions(ctx context.Context, params map[string]any) (any, error) {
	productID, _ := params["product_id"].(string)
	entityType, _ := params["entity_type"].(string)

	var defs []aha.CustomFieldDefinition
	var err error

	if productID != "" {
		defs, err = h.client.ListProductCustomFieldDefinitions(ctx, productID)
	} else {
		defs, err = h.client.ListCustomFieldDefinitions(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("listing custom field definitions: %w", err)
	}

	// Filter by entity type if specified
	if entityType != "" {
		var filtered []aha.CustomFieldDefinition
		for _, d := range defs {
			if d.CustomFieldableType == entityType {
				filtered = append(filtered, d)
			}
		}
		defs = filtered
	}

	// Group by entity type for better readability
	byType := make(map[string][]map[string]any)
	for _, d := range defs {
		field := map[string]any{
			"id":   d.ID,
			"name": d.Name,
			"key":  d.Key,
			"type": d.Type,
		}
		if d.AllowsOtherOption {
			field["allows_other"] = true
		}
		byType[d.CustomFieldableType] = append(byType[d.CustomFieldableType], field)
	}

	return map[string]any{
		"count":            len(defs),
		"fields_by_entity": byType,
		"message":          fmt.Sprintf("Found %d custom field definitions", len(defs)),
	}, nil
}

// ListCustomFieldOptions lists options for a select custom field.
func (h *ToolHandlers) ListCustomFieldOptions(ctx context.Context, params map[string]any) (any, error) {
	fieldID, ok := params["field_id"].(string)
	if !ok || fieldID == "" {
		return nil, fmt.Errorf("field_id parameter is required")
	}

	opts, err := h.client.ListCustomFieldOptions(ctx, fieldID)
	if err != nil {
		return nil, fmt.Errorf("listing custom field options: %w", err)
	}

	var options []map[string]any
	for _, o := range opts {
		opt := map[string]any{
			"id":    o.ID,
			"value": o.Value,
		}
		if o.Position > 0 {
			opt["position"] = o.Position
		}
		if o.Color != "" {
			opt["color"] = o.Color
		}
		options = append(options, opt)
	}

	return map[string]any{
		"count":   len(options),
		"options": options,
		"message": fmt.Sprintf("Found %d options for field %s", len(options), fieldID),
	}, nil
}

// UpdateEpic updates an epic's fields.
//
//nolint:dupl // Similar pattern for different entity types
func (h *ToolHandlers) UpdateEpic(ctx context.Context, params map[string]any) (any, error) {
	epicID, ok := params["epic_id"].(string)
	if !ok || epicID == "" {
		return nil, fmt.Errorf("epic_id parameter is required")
	}

	var opts []aha.UpdateEpicOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateEpicName(name))
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateEpicDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithUpdateEpicStatus(status))
	}

	if progress, ok := params["progress"].(float64); ok {
		opts = append(opts, aha.WithUpdateEpicProgress(progress))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	epic, err := h.client.UpdateEpic(ctx, epicID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating epic: %w", err)
	}

	return map[string]any{
		"id":            epic.ID,
		"reference_num": epic.ReferenceNum,
		"name":          epic.Name,
		"url":           epic.URL,
		"message":       fmt.Sprintf("Epic %s updated successfully", epic.ReferenceNum),
	}, nil
}

// UpdateGoal updates a goal's fields.
//
//nolint:dupl // Similar pattern for different entity types
func (h *ToolHandlers) UpdateGoal(ctx context.Context, params map[string]any) (any, error) {
	goalID, ok := params["goal_id"].(string)
	if !ok || goalID == "" {
		return nil, fmt.Errorf("goal_id parameter is required")
	}

	var opts []aha.UpdateGoalOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateGoalName(name))
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateGoalDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithUpdateGoalStatus(status))
	}

	if progress, ok := params["progress"].(float64); ok {
		opts = append(opts, aha.WithUpdateGoalProgress(progress))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	goal, err := h.client.UpdateGoal(ctx, goalID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating goal: %w", err)
	}

	return map[string]any{
		"id":            goal.ID,
		"reference_num": goal.ReferenceNum,
		"name":          goal.Name,
		"url":           goal.URL,
		"message":       fmt.Sprintf("Goal %s updated successfully", goal.ReferenceNum),
	}, nil
}

// UpdateFeature updates a feature's fields (general update tool).
func (h *ToolHandlers) UpdateFeature(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	var opts []aha.UpdateFeatureOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateFeatureName(name))
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateFeatureDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithUpdateFeatureStatus(status))
	}

	if user, ok := params["assigned_to_user"].(string); ok && user != "" {
		opts = append(opts, aha.WithUpdateFeatureAssignedToUser(user))
	}

	if release, ok := params["release"].(string); ok && release != "" {
		opts = append(opts, aha.WithUpdateFeatureRelease(release))
	}

	if initiative, ok := params["initiative"].(string); ok && initiative != "" {
		opts = append(opts, aha.WithUpdateFeatureInitiative(initiative))
	}

	if tags, ok := params["tags"].(string); ok && tags != "" {
		opts = append(opts, aha.WithUpdateFeatureTags(tags))
	}

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithUpdateFeatureStartDate(t))
	}

	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		t, err := time.Parse("2006-01-02", dueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithUpdateFeatureDueDate(t))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	feature, err := h.client.UpdateFeature(ctx, featureID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating feature: %w", err)
	}

	var statusName string
	if feature.WorkflowStatus != nil {
		statusName = feature.WorkflowStatus.Name
	}

	return map[string]any{
		"id":            feature.ID,
		"reference_num": feature.ReferenceNum,
		"name":          feature.Name,
		"status":        statusName,
		"url":           feature.URL,
		"message":       fmt.Sprintf("Feature %s updated successfully", feature.ReferenceNum),
	}, nil
}

// customFieldableTypeFromString maps an MCP entity_type parameter
// (case-insensitive) to the GraphQL CustomFieldableTypeEnum value
// SetCustomFieldValues expects. Scoped to Feature, Initiative, and
// Release - the entities this tool was built for, even though the
// underlying enum has more values (Epic, Goal, Idea, etc.).
func customFieldableTypeFromString(entityType string) (generated.CustomFieldableTypeEnum, error) {
	switch strings.ToLower(entityType) {
	case "feature":
		return generated.CustomFieldableTypeEnumFeature, nil
	case "initiative":
		return generated.CustomFieldableTypeEnumInitiative, nil
	case "release":
		return generated.CustomFieldableTypeEnumRelease, nil
	default:
		return "", fmt.Errorf("entity_type must be one of: feature, initiative, release (got %q)", entityType)
	}
}

// SetCustomFieldValues sets one or more custom field values on a Feature,
// Initiative, or Release record. Use list_custom_fields to discover valid
// keys for a given entity_type, and list_custom_field_options for the
// valid values of a select/choice field.
func (h *ToolHandlers) SetCustomFieldValues(ctx context.Context, params map[string]any) (any, error) {
	entityType, _ := params["entity_type"].(string)
	recordType, err := customFieldableTypeFromString(entityType)
	if err != nil {
		return nil, err
	}

	recordID, _ := params["record_id"].(string)
	if recordID == "" {
		return nil, fmt.Errorf("record_id parameter is required")
	}

	customFields, ok := params["custom_fields"].(map[string]any)
	if !ok || len(customFields) == 0 {
		return nil, fmt.Errorf("custom_fields parameter is required (a non-empty object mapping custom field key to value)")
	}

	values, err := ahagql.SetCustomFieldValues(ctx, h.graphqlClient, recordID, recordType, customFields)
	if err != nil {
		return nil, fmt.Errorf("setting custom field values: %w", err)
	}

	results := make([]map[string]any, 0, len(values))
	for _, v := range values {
		results = append(results, map[string]any{
			"key":         v.Key,
			"value":       v.Value,
			"human_value": v.HumanValue,
		})
	}

	return map[string]any{
		"record_id":     recordID,
		"entity_type":   entityType,
		"custom_fields": results,
		"message":       fmt.Sprintf("Updated %d custom field(s) on %s %s", len(results), entityType, recordID),
	}, nil
}

// UpdateRequirement updates a requirement's fields.
//
//nolint:dupl // Similar pattern for different entity types
func (h *ToolHandlers) UpdateRequirement(ctx context.Context, params map[string]any) (any, error) {
	requirementID, ok := params["requirement_id"].(string)
	if !ok || requirementID == "" {
		return nil, fmt.Errorf("requirement_id parameter is required")
	}

	var opts []aha.UpdateRequirementOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateRequirementName(name))
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateRequirementDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithUpdateRequirementStatus(status))
	}

	if workDone, ok := params["work_done"].(float64); ok {
		opts = append(opts, aha.WithUpdateRequirementWorkDone(workDone))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	req, err := h.client.UpdateRequirement(ctx, requirementID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating requirement: %w", err)
	}

	return map[string]any{
		"id":            req.ID,
		"reference_num": req.ReferenceNum,
		"name":          req.Name,
		"url":           req.URL,
		"message":       fmt.Sprintf("Requirement %s updated successfully", req.ReferenceNum),
	}, nil
}

// UpdateRelease updates a release's fields.
//
//nolint:dupl // Similar pattern for different entity types
func (h *ToolHandlers) UpdateRelease(ctx context.Context, params map[string]any) (any, error) {
	releaseID, ok := params["release_id"].(string)
	if !ok || releaseID == "" {
		return nil, fmt.Errorf("release_id parameter is required")
	}

	var opts []aha.UpdateReleaseOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithReleaseName(name))
	}

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithReleaseStartDate(t))
	}

	if releaseDate, ok := params["release_date"].(string); ok && releaseDate != "" {
		t, err := time.Parse("2006-01-02", releaseDate)
		if err != nil {
			return nil, fmt.Errorf("invalid release_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithReleaseDate(t))
	}

	if parkingLot, ok := params["parking_lot"].(bool); ok {
		opts = append(opts, aha.WithReleaseParkingLot(parkingLot))
	}

	if description, ok := params["description"].(string); ok && description != "" {
		opts = append(opts, aha.WithReleaseTheme(description))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	release, err := h.client.UpdateRelease(ctx, releaseID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating release: %w", err)
	}

	return map[string]any{
		"id":            release.ID,
		"reference_num": release.ReferenceNum,
		"name":          release.Name,
		"parking_lot":   release.ParkingLot,
		"description":   release.Theme,
		"url":           release.URL,
		"message":       fmt.Sprintf("Release %s updated successfully", release.ReferenceNum),
	}, nil
}

// CreateEpic creates a new epic in a release.
func (h *ToolHandlers) CreateEpic(ctx context.Context, params map[string]any) (any, error) {
	releaseID, ok := params["release_id"].(string)
	if !ok || releaseID == "" {
		return nil, fmt.Errorf("release_id parameter is required")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	var opts []aha.CreateEpicOption
	opts = append(opts, aha.WithEpicName(name))

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithEpicDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithEpicStatus(status))
	}

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithEpicStartDate(t))
	}

	if dueDate, ok := params["due_date"].(string); ok && dueDate != "" {
		t, err := time.Parse("2006-01-02", dueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithEpicDueDate(t))
	}

	if color, ok := params["color"].(string); ok && color != "" {
		opts = append(opts, aha.WithEpicColor(color))
	}

	if initiative, ok := params["initiative"].(string); ok && initiative != "" {
		opts = append(opts, aha.WithEpicInitiative(initiative))
	}

	epic, err := h.client.CreateEpic(ctx, releaseID, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating epic: %w", err)
	}

	return map[string]any{
		"id":            epic.ID,
		"reference_num": epic.ReferenceNum,
		"name":          epic.Name,
		"url":           epic.URL,
		"message":       fmt.Sprintf("Epic %s created successfully", epic.ReferenceNum),
	}, nil
}

// CreateGoal creates a new goal in a product.
func (h *ToolHandlers) CreateGoal(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	var opts []aha.CreateGoalOption
	opts = append(opts, aha.WithGoalName(name))

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithGoalDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithGoalStatus(status))
	}

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithGoalStartDate(t))
	}

	if endDate, ok := params["end_date"].(string); ok && endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithGoalEndDate(t))
	}

	goal, err := h.client.CreateGoal(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating goal: %w", err)
	}

	return map[string]any{
		"id":            goal.ID,
		"reference_num": goal.ReferenceNum,
		"name":          goal.Name,
		"url":           goal.URL,
		"message":       fmt.Sprintf("Goal %s created successfully", goal.ReferenceNum),
	}, nil
}

// CreateInitiative creates a new initiative in a product.
func (h *ToolHandlers) CreateInitiative(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	var opts []aha.CreateInitiativeOption
	opts = append(opts, aha.WithInitiativeName(name))

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithInitiativeDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithInitiativeStatus(status))
	}

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithInitiativeStartDate(t))
	}

	if endDate, ok := params["end_date"].(string); ok && endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format (expected YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithInitiativeEndDate(t))
	}

	if value, ok := params["value"].(float64); ok {
		opts = append(opts, aha.WithInitiativeValue(value))
	}

	if effort, ok := params["effort"].(float64); ok {
		opts = append(opts, aha.WithInitiativeEffort(effort))
	}

	if color, ok := params["color"].(string); ok && color != "" {
		opts = append(opts, aha.WithInitiativeColor(color))
	}

	initiative, err := h.client.CreateInitiative(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating initiative: %w", err)
	}

	return map[string]any{
		"id":            initiative.ID,
		"reference_num": initiative.ReferenceNum,
		"name":          initiative.Name,
		"url":           initiative.URL,
		"message":       fmt.Sprintf("Initiative %s created successfully", initiative.ReferenceNum),
	}, nil
}

// CreateRequirement creates a new requirement for a feature.
func (h *ToolHandlers) CreateRequirement(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	var opts []aha.CreateRequirementOption
	opts = append(opts, aha.WithRequirementName(name))

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithRequirementDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithRequirementStatus(status))
	}

	if user, ok := params["assigned_to_user"].(string); ok && user != "" {
		opts = append(opts, aha.WithRequirementAssignedTo(user))
	}

	if estimate, ok := params["original_estimate"].(float64); ok {
		opts = append(opts, aha.WithRequirementEstimate(estimate))
	}

	requirement, err := h.client.CreateRequirement(ctx, featureID, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating requirement: %w", err)
	}

	return map[string]any{
		"id":            requirement.ID,
		"reference_num": requirement.ReferenceNum,
		"name":          requirement.Name,
		"url":           requirement.URL,
		"message":       fmt.Sprintf("Requirement %s created successfully", requirement.ReferenceNum),
	}, nil
}

// UpdateIdea updates an idea's fields.
//
//nolint:dupl // Similar pattern for different entity types
func (h *ToolHandlers) UpdateIdea(ctx context.Context, params map[string]any) (any, error) {
	ideaID, ok := params["idea_id"].(string)
	if !ok || ideaID == "" {
		return nil, fmt.Errorf("idea_id parameter is required")
	}

	var opts []aha.UpdateIdeaOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateIdeaName(name))
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateIdeaDescription(desc))
	}

	if status, ok := params["workflow_status"].(string); ok && status != "" {
		opts = append(opts, aha.WithUpdateIdeaStatus(status))
	}

	if visibility, ok := params["visibility"].(string); ok && visibility != "" {
		opts = append(opts, aha.WithUpdateIdeaVisibility(visibility))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	idea, err := h.client.UpdateIdea(ctx, ideaID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating idea: %w", err)
	}

	var statusName string
	if idea.WorkflowStatus != nil {
		statusName = idea.WorkflowStatus.Name
	}

	return map[string]any{
		"id":            idea.ID,
		"reference_num": idea.ReferenceNum,
		"name":          idea.Name,
		"status":        statusName,
		"message":       fmt.Sprintf("Idea %s updated successfully", idea.ReferenceNum),
	}, nil
}

// =============================================================================
// List Tools
// =============================================================================

// ListFeatures lists features with optional filtering.
func (h *ToolHandlers) ListFeatures(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	var opts []aha.ListFeaturesOption

	if q, ok := params["q"].(string); ok && q != "" {
		opts = append(opts, aha.WithFeatureQuery(q))
	}
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithFeaturePage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithFeaturePerPage(int(perPage)))
	}

	list, err := h.client.ListFeatures(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing features: %w", err)
	}

	features := make([]map[string]any, len(list.Features))
	for i, f := range list.Features {
		features[i] = map[string]any{
			"id":            f.ID,
			"reference_num": f.ReferenceNum,
			"name":          f.Name,
			"url":           f.URL,
		}
	}

	return map[string]any{
		"features":   features,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListReleaseFeatures lists features for a specific release.
func (h *ToolHandlers) ListReleaseFeatures(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	releaseID, ok := params["release_id"].(string)
	if !ok || releaseID == "" {
		return nil, fmt.Errorf("release_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListReleaseFeatures(ctx, releaseID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing release features: %w", err)
	}

	features := make([]map[string]any, len(list.Features))
	for i, f := range list.Features {
		features[i] = map[string]any{
			"id":            f.ID,
			"reference_num": f.ReferenceNum,
			"name":          f.Name,
			"url":           f.URL,
		}
	}

	return map[string]any{
		"features":   features,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListEpics lists epics with optional filtering.
func (h *ToolHandlers) ListEpics(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	var opts []aha.ListEpicsOption

	if q, ok := params["q"].(string); ok && q != "" {
		opts = append(opts, aha.WithEpicQuery(q))
	}
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithEpicPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithEpicPerPage(int(perPage)))
	}

	list, err := h.client.ListEpics(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing epics: %w", err)
	}

	epics := make([]map[string]any, len(list.Epics))
	for i, e := range list.Epics {
		epics[i] = map[string]any{
			"id":            e.ID,
			"reference_num": e.ReferenceNum,
			"name":          e.Name,
			"url":           e.URL,
		}
	}

	return map[string]any{
		"epics":      epics,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListProductEpics lists epics for a specific product.
func (h *ToolHandlers) ListProductEpics(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListProductEpics(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing product epics: %w", err)
	}

	epics := make([]map[string]any, len(list.Epics))
	for i, e := range list.Epics {
		epics[i] = map[string]any{
			"id":            e.ID,
			"reference_num": e.ReferenceNum,
			"name":          e.Name,
			"url":           e.URL,
		}
	}

	return map[string]any{
		"epics":      epics,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListGoals lists goals with optional filtering.
func (h *ToolHandlers) ListGoals(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	var opts []aha.ListGoalsOption

	if q, ok := params["q"].(string); ok && q != "" {
		opts = append(opts, aha.WithGoalQuery(q))
	}
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithGoalPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithGoalPerPage(int(perPage)))
	}

	list, err := h.client.ListGoals(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing goals: %w", err)
	}

	goals := make([]map[string]any, len(list.Goals))
	for i, g := range list.Goals {
		goals[i] = map[string]any{
			"id":            g.ID,
			"reference_num": g.ReferenceNum,
			"name":          g.Name,
			"url":           g.URL,
		}
	}

	return map[string]any{
		"goals":      goals,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListProductGoals lists goals for a specific product.
func (h *ToolHandlers) ListProductGoals(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListProductGoals(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing product goals: %w", err)
	}

	goals := make([]map[string]any, len(list.Goals))
	for i, g := range list.Goals {
		goals[i] = map[string]any{
			"id":            g.ID,
			"reference_num": g.ReferenceNum,
			"name":          g.Name,
			"url":           g.URL,
		}
	}

	return map[string]any{
		"goals":      goals,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListInitiatives lists initiatives with optional filtering.
func (h *ToolHandlers) ListInitiatives(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	var opts []aha.ListInitiativesOption

	if q, ok := params["q"].(string); ok && q != "" {
		opts = append(opts, aha.WithInitiativeQuery(q))
	}
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithInitiativePage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithInitiativePerPage(int(perPage)))
	}

	list, err := h.client.ListInitiatives(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing initiatives: %w", err)
	}

	initiatives := make([]map[string]any, len(list.Initiatives))
	for i, init := range list.Initiatives {
		initiatives[i] = map[string]any{
			"id":            init.ID,
			"reference_num": init.ReferenceNum,
			"name":          init.Name,
			"url":           init.URL,
		}
	}

	return map[string]any{
		"initiatives": initiatives,
		"pagination":  paginationToMap(list.Pagination),
	}, nil
}

// ListProductInitiatives lists initiatives for a specific product.
func (h *ToolHandlers) ListProductInitiatives(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListProductInitiatives(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing product initiatives: %w", err)
	}

	initiatives := make([]map[string]any, len(list.Initiatives))
	for i, init := range list.Initiatives {
		initiatives[i] = map[string]any{
			"id":            init.ID,
			"reference_num": init.ReferenceNum,
			"name":          init.Name,
			"url":           init.URL,
		}
	}

	return map[string]any{
		"initiatives": initiatives,
		"pagination":  paginationToMap(list.Pagination),
	}, nil
}

// ListFeatureRequirements lists requirements for a specific feature.
func (h *ToolHandlers) ListFeatureRequirements(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across list handlers
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListFeatureRequirements(ctx, featureID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing feature requirements: %w", err)
	}

	requirements := make([]map[string]any, len(list.Requirements))
	for i, r := range list.Requirements {
		requirements[i] = map[string]any{
			"id":            r.ID,
			"reference_num": r.ReferenceNum,
			"name":          r.Name,
			"url":           r.URL,
		}
	}

	return map[string]any{
		"requirements": requirements,
		"pagination":   paginationToMap(list.Pagination),
	}, nil
}

// ListUsers lists workspace users.
func (h *ToolHandlers) ListUsers(ctx context.Context, params map[string]any) (any, error) {
	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListUsers(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	users := make([]map[string]any, len(list.Users))
	for i, u := range list.Users {
		users[i] = map[string]any{
			"id":    u.ID,
			"name":  u.Name,
			"email": u.Email,
		}
	}

	return map[string]any{
		"users":      users,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// =============================================================================
// Delete Tools
// =============================================================================

// DeleteRequirement deletes a requirement.
func (h *ToolHandlers) DeleteRequirement(ctx context.Context, params map[string]any) (any, error) {
	requirementID, ok := params["requirement_id"].(string)
	if !ok || requirementID == "" {
		return nil, fmt.Errorf("requirement_id parameter is required")
	}

	err := h.client.DeleteRequirement(ctx, requirementID)
	if err != nil {
		return nil, fmt.Errorf("deleting requirement: %w", err)
	}

	return map[string]any{
		"message": fmt.Sprintf("Requirement %s deleted successfully", requirementID),
	}, nil
}

// DeleteComment deletes a comment.
func (h *ToolHandlers) DeleteComment(ctx context.Context, params map[string]any) (any, error) {
	commentID, ok := params["comment_id"].(string)
	if !ok || commentID == "" {
		return nil, fmt.Errorf("comment_id parameter is required")
	}

	err := h.client.DeleteComment(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("deleting comment: %w", err)
	}

	return map[string]any{
		"message": fmt.Sprintf("Comment %s deleted successfully", commentID),
	}, nil
}

// =============================================================================
// Comment Tools
// =============================================================================

// UpdateComment updates an existing comment.
func (h *ToolHandlers) UpdateComment(ctx context.Context, params map[string]any) (any, error) {
	commentID, ok := params["comment_id"].(string)
	if !ok || commentID == "" {
		return nil, fmt.Errorf("comment_id parameter is required")
	}

	var opts []aha.UpdateCommentOption

	if body, ok := params["body"].(string); ok && body != "" {
		opts = append(opts, aha.WithUpdateCommentBody(body))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("body parameter is required")
	}

	comment, err := h.client.UpdateComment(ctx, commentID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating comment: %w", err)
	}

	return map[string]any{
		"id":      comment.ID,
		"body":    comment.Body,
		"message": "Comment updated successfully",
	}, nil
}

// ListFeatureComments lists comments for a feature.
func (h *ToolHandlers) ListFeatureComments(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across comment list handlers
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListFeatureComments(ctx, featureID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing feature comments: %w", err)
	}

	comments := make([]map[string]any, len(list.Comments))
	for i, c := range list.Comments {
		var userName string
		if c.User != nil {
			userName = c.User.Name()
		}
		comments[i] = map[string]any{
			"id":         c.ID,
			"body":       c.Body,
			"user":       userName,
			"created_at": c.CreatedAt.Format(time.RFC3339),
		}
	}

	return map[string]any{
		"comments":   comments,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListIdeaComments lists comments for an idea.
func (h *ToolHandlers) ListIdeaComments(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across comment list handlers
	ideaID, ok := params["idea_id"].(string)
	if !ok || ideaID == "" {
		return nil, fmt.Errorf("idea_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListIdeaComments(ctx, ideaID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing idea comments: %w", err)
	}

	comments := make([]map[string]any, len(list.Comments))
	for i, c := range list.Comments {
		var userName string
		if c.User != nil {
			userName = c.User.Name()
		}
		comments[i] = map[string]any{
			"id":         c.ID,
			"body":       c.Body,
			"user":       userName,
			"created_at": c.CreatedAt.Format(time.RFC3339),
		}
	}

	return map[string]any{
		"comments":   comments,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// ListEpicComments lists comments for an epic.
func (h *ToolHandlers) ListEpicComments(ctx context.Context, params map[string]any) (any, error) { //nolint:dupl // Similar pagination pattern across comment list handlers
	epicID, ok := params["epic_id"].(string)
	if !ok || epicID == "" {
		return nil, fmt.Errorf("epic_id parameter is required")
	}

	var opts []aha.ListOption
	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithPage(int(page)))
	}
	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithPerPage(int(perPage)))
	}

	list, err := h.client.ListEpicComments(ctx, epicID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing epic comments: %w", err)
	}

	comments := make([]map[string]any, len(list.Comments))
	for i, c := range list.Comments {
		var userName string
		if c.User != nil {
			userName = c.User.Name()
		}
		comments[i] = map[string]any{
			"id":         c.ID,
			"body":       c.Body,
			"user":       userName,
			"created_at": c.CreatedAt.Format(time.RFC3339),
		}
	}

	return map[string]any{
		"comments":   comments,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// =============================================================================
// User Tools
// =============================================================================

// GetCurrentUser returns the authenticated user.
func (h *ToolHandlers) GetCurrentUser(ctx context.Context, params map[string]any) (any, error) {
	user, err := h.client.GetCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting current user: %w", err)
	}

	return map[string]any{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	}, nil
}

// =============================================================================
// Product Tools
// =============================================================================

// CreateProduct creates a new product.
func (h *ToolHandlers) CreateProduct(ctx context.Context, params map[string]any) (any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	referencePrefix, ok := params["reference_prefix"].(string)
	if !ok || referencePrefix == "" {
		return nil, fmt.Errorf("reference_prefix parameter is required")
	}

	var opts []aha.CreateProductOption

	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithProductDescription(desc))
	}
	if parentID, ok := params["parent_id"].(string); ok && parentID != "" {
		opts = append(opts, aha.WithProductParentID(parentID))
	}

	product, err := h.client.CreateProduct(ctx, name, referencePrefix, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating product: %w", err)
	}

	return map[string]any{
		"id":               product.ID,
		"reference_prefix": product.ReferencePrefix,
		"name":             product.Name,
		"message":          fmt.Sprintf("Product %s created successfully", product.ReferencePrefix),
	}, nil
}

// UpdateProduct updates an existing product.
func (h *ToolHandlers) UpdateProduct(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	var opts []aha.UpdateProductOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateProductName(name))
	}
	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateProductDescription(desc))
	}
	if prefix, ok := params["reference_prefix"].(string); ok && prefix != "" {
		opts = append(opts, aha.WithUpdateProductReferencePrefix(prefix))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	product, err := h.client.UpdateProduct(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating product: %w", err)
	}

	return map[string]any{
		"id":               product.ID,
		"reference_prefix": product.ReferencePrefix,
		"name":             product.Name,
		"message":          fmt.Sprintf("Product %s updated successfully", product.ReferencePrefix),
	}, nil
}

// =============================================================================
// Strategic Model Tools
// =============================================================================

// ListStrategicModels lists strategic models.
func (h *ToolHandlers) ListStrategicModels(ctx context.Context, params map[string]any) (any, error) {
	var opts []aha.ListStrategicModelsOption

	if kind, ok := params["kind"].(string); ok && kind != "" {
		opts = append(opts, aha.WithStrategicModelKind(kind))
	}

	list, err := h.client.ListStrategicModels(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing strategic models: %w", err)
	}

	models := make([]map[string]any, len(list.StrategicModels))
	for i, m := range list.StrategicModels {
		models[i] = map[string]any{
			"id":   m.ID,
			"name": m.Name,
			"kind": m.Kind,
			"url":  m.URL,
		}
	}

	return map[string]any{
		"strategic_models": models,
		"pagination":       paginationToMap(list.Pagination),
	}, nil
}

// ListProductStrategicModels lists strategic models for a product.
func (h *ToolHandlers) ListProductStrategicModels(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	var opts []aha.ListStrategicModelsOption

	if kind, ok := params["kind"].(string); ok && kind != "" {
		opts = append(opts, aha.WithStrategicModelKind(kind))
	}

	list, err := h.client.ListProductStrategicModels(ctx, productID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing product strategic models: %w", err)
	}

	models := make([]map[string]any, len(list.StrategicModels))
	for i, m := range list.StrategicModels {
		models[i] = map[string]any{
			"id":   m.ID,
			"name": m.Name,
			"kind": m.Kind,
			"url":  m.URL,
		}
	}

	return map[string]any{
		"strategic_models": models,
		"pagination":       paginationToMap(list.Pagination),
	}, nil
}

// GetStrategicModel retrieves a strategic model by ID.
func (h *ToolHandlers) GetStrategicModel(ctx context.Context, params map[string]any) (any, error) {
	modelID, ok := params["model_id"].(string)
	if !ok || modelID == "" {
		return nil, fmt.Errorf("model_id parameter is required")
	}

	model, err := h.client.GetStrategicModel(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("getting strategic model: %w", err)
	}

	components := make([]map[string]any, len(model.Components))
	for i, c := range model.Components {
		components[i] = map[string]any{
			"id":          c.ID,
			"name":        c.Name,
			"description": c.Description,
			"position":    c.Position,
		}
	}

	return map[string]any{
		"id":         model.ID,
		"name":       model.Name,
		"kind":       model.Kind,
		"url":        model.URL,
		"components": components,
	}, nil
}

// CreateStrategicModel creates a new strategic model.
func (h *ToolHandlers) CreateStrategicModel(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}

	kind, ok := params["kind"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("kind parameter is required")
	}

	var opts []aha.CreateStrategicModelOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithStrategicModelName(name))
	}

	model, err := h.client.CreateStrategicModel(ctx, productID, kind, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating strategic model: %w", err)
	}

	return map[string]any{
		"id":      model.ID,
		"name":    model.Name,
		"kind":    model.Kind,
		"url":     model.URL,
		"message": fmt.Sprintf("Strategic model created successfully"),
	}, nil
}

// UpdateStrategicModel updates a strategic model.
func (h *ToolHandlers) UpdateStrategicModel(ctx context.Context, params map[string]any) (any, error) {
	modelID, ok := params["model_id"].(string)
	if !ok || modelID == "" {
		return nil, fmt.Errorf("model_id parameter is required")
	}

	var opts []aha.UpdateStrategicModelOption

	if name, ok := params["name"].(string); ok && name != "" {
		opts = append(opts, aha.WithUpdateStrategicModelName(name))
	}
	if desc, ok := params["description"].(string); ok && desc != "" {
		opts = append(opts, aha.WithUpdateStrategicModelDescription(desc))
	}

	if len(opts) == 0 {
		return nil, fmt.Errorf("at least one field to update is required")
	}

	model, err := h.client.UpdateStrategicModel(ctx, modelID, opts...)
	if err != nil {
		return nil, fmt.Errorf("updating strategic model: %w", err)
	}

	return map[string]any{
		"id":      model.ID,
		"name":    model.Name,
		"kind":    model.Kind,
		"url":     model.URL,
		"message": "Strategic model updated successfully",
	}, nil
}

// =============================================================================
// Create Tools (GraphQL mutations)
// =============================================================================

// CreateIdea creates a new idea using the GraphQL API.
func (h *ToolHandlers) CreateIdea(ctx context.Context, params map[string]any) (any, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name parameter is required")
	}

	productID, _ := params["product_id"].(string)
	if productID == "" {
		productID = h.config.DefaultProduct
	}
	if productID == "" {
		return nil, fmt.Errorf("product_id parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	// Optional: skip required fields validation
	var skipValidation *bool
	if skip, ok := params["skip_required_fields_validation"].(bool); ok {
		skipValidation = &skip
	}

	// Use GraphQL mutation
	resp, err := generated.CreateIdea(ctx, h.graphqlClient, name, productID, skipValidation)
	if err != nil {
		return nil, fmt.Errorf("creating idea: %w", err)
	}

	if resp.CreateIdea == nil {
		return nil, fmt.Errorf("creating idea: no response")
	}

	// Check for errors
	if len(resp.CreateIdea.Errors.Attributes) > 0 {
		var errMsgs []string
		for _, attr := range resp.CreateIdea.Errors.Attributes {
			for _, msg := range attr.FullMessages {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", attr.Name, msg))
			}
		}
		return nil, fmt.Errorf("creating idea: %s", errMsgs)
	}

	if resp.CreateIdea.Idea == nil {
		return nil, fmt.Errorf("creating idea: no idea returned")
	}

	idea := resp.CreateIdea.Idea
	return map[string]any{
		"id":              idea.Id,
		"reference_num":   idea.ReferenceNum,
		"name":            idea.Name,
		"workflow_status": idea.WorkflowStatus.Name,
		"created_at":      idea.CreatedAt.Format(time.RFC3339),
		"message":         "Idea created successfully",
	}, nil
}

// AddGoalToFeature links a goal to a feature using the GraphQL CreateRecordLink mutation.
func (h *ToolHandlers) AddGoalToFeature(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id parameter is required")
	}

	goalID, ok := params["goal_id"].(string)
	if !ok || goalID == "" {
		return nil, fmt.Errorf("goal_id parameter is required")
	}

	// Default link type is RELATES_TO
	linkType := generated.RecordLinkTypeEnumRelatesTo
	if lt, ok := params["link_type"].(string); ok && lt != "" {
		switch lt {
		case "RELATES_TO":
			linkType = generated.RecordLinkTypeEnumRelatesTo
		case "IMPACTS":
			linkType = generated.RecordLinkTypeEnumImpacts
		case "DEPENDS_ON":
			linkType = generated.RecordLinkTypeEnumDependsOn
		default:
			return nil, fmt.Errorf("invalid link_type: %s (valid: RELATES_TO, IMPACTS, DEPENDS_ON)", lt)
		}
	}

	resp, err := generated.CreateRecordLink(ctx, h.graphqlClient,
		generated.LinkableRecordTypesEnumGoal,
		goalID,
		generated.LinkableRecordTypesEnumFeature,
		featureID,
		linkType,
	)
	if err != nil {
		return nil, fmt.Errorf("creating record link: %w", err)
	}

	if resp.CreateRecordLink == nil {
		return nil, fmt.Errorf("creating record link: no response")
	}

	// Check for errors
	if len(resp.CreateRecordLink.Errors.Attributes) > 0 {
		var errMsgs []string
		for _, attr := range resp.CreateRecordLink.Errors.Attributes {
			for _, msg := range attr.FullMessages {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", attr.Name, msg))
			}
		}
		return nil, fmt.Errorf("creating record link: %s", errMsgs)
	}

	if resp.CreateRecordLink.RecordLink == nil {
		return nil, fmt.Errorf("creating record link: no link returned")
	}

	link := resp.CreateRecordLink.RecordLink
	return map[string]any{
		"id":        link.Id,
		"link_type": string(link.LinkType),
		"message":   fmt.Sprintf("Goal %s linked to feature %s", goalID, featureID),
	}, nil
}

// =============================================================================
// Offline/Cache-Based Tools (require synced SQLite database)
// =============================================================================

// ListFeaturesByReleaseDate lists features by release date from the local cache.
// This tool queries the synced SQLite database, not the live Aha API.
func (h *ToolHandlers) ListFeaturesByReleaseDate(ctx context.Context, params map[string]any) (any, error) {
	if h.syncDB == nil {
		return nil, fmt.Errorf("sync database not configured (set AHA_DB_PATH environment variable)")
	}

	product, _ := params["product"].(string)
	if product == "" {
		product = h.config.DefaultProduct
	}
	if product == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	releaseDate, _ := params["release_date"].(string)
	startDate, _ := params["start_date"].(string)
	endDate, _ := params["end_date"].(string)

	var features []map[string]any
	var err error

	if releaseDate != "" {
		// Exact date match
		features, err = h.syncDB.GetFeaturesByReleaseDate(ctx, product, releaseDate)
	} else if startDate != "" || endDate != "" {
		// Date range
		features, err = h.syncDB.GetFeaturesByReleaseDateRange(ctx, product, startDate, endDate)
	} else {
		return nil, fmt.Errorf("one of release_date, start_date, or end_date is required")
	}

	if err != nil {
		return nil, fmt.Errorf("querying features by release date: %w", err)
	}

	return map[string]any{
		"product":  product,
		"count":    len(features),
		"features": features,
	}, nil
}

// GetIdeasStatistics returns aggregated statistics for ideas from the local cache.
func (h *ToolHandlers) GetIdeasStatistics(ctx context.Context, params map[string]any) (any, error) {
	if h.syncDB == nil {
		return nil, fmt.Errorf("sync database not configured (set AHA_DB_PATH environment variable)")
	}

	product, _ := params["product"].(string)
	if product == "" {
		product = h.config.DefaultProduct
	}
	if product == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	limit := 10
	if l, ok := params["top_count"].(float64); ok && l > 0 {
		limit = int(l)
	}

	stats, err := h.syncDB.GetIdeasStatistics(ctx, product, limit)
	if err != nil {
		return nil, fmt.Errorf("getting ideas statistics: %w", err)
	}

	return map[string]any{
		"product":    product,
		"statistics": stats,
	}, nil
}

// GetFeaturesStatistics returns aggregated statistics for features from the local cache.
func (h *ToolHandlers) GetFeaturesStatistics(ctx context.Context, params map[string]any) (any, error) {
	if h.syncDB == nil {
		return nil, fmt.Errorf("sync database not configured (set AHA_DB_PATH environment variable)")
	}

	product, _ := params["product"].(string)
	if product == "" {
		product = h.config.DefaultProduct
	}
	if product == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	limit := 5
	if l, ok := params["upcoming_releases_count"].(float64); ok && l > 0 {
		limit = int(l)
	}

	stats, err := h.syncDB.GetFeaturesStatistics(ctx, product, limit)
	if err != nil {
		return nil, fmt.Errorf("getting features statistics: %w", err)
	}

	return map[string]any{
		"product":    product,
		"statistics": stats,
	}, nil
}

// GetVoterDomainHistogram returns an email-domain histogram of idea voters from the local cache.
func (h *ToolHandlers) GetVoterDomainHistogram(ctx context.Context, params map[string]any) (any, error) {
	if h.syncDB == nil {
		return nil, fmt.Errorf("sync database not configured (set AHA_DB_PATH environment variable)")
	}

	product, _ := params["product"].(string)
	if product == "" {
		product = h.config.DefaultProduct
	}
	if product == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	ideaID, _ := params["idea_id"].(string)

	limit := 10
	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	stats, err := h.syncDB.GetVoterEmailDomainStatistics(ctx, product, ideaID, limit)
	if err != nil {
		return nil, fmt.Errorf("getting voter domain histogram: %w", err)
	}

	return map[string]any{
		"product":    product,
		"idea_id":    ideaID,
		"statistics": stats,
	}, nil
}

// ListFeaturesByReleaseName lists features by release name from the local cache.
// This tool queries the synced SQLite database, not the live Aha API.
func (h *ToolHandlers) ListFeaturesByReleaseName(ctx context.Context, params map[string]any) (any, error) {
	if h.syncDB == nil {
		return nil, fmt.Errorf("sync database not configured (set AHA_DB_PATH environment variable)")
	}

	product, _ := params["product"].(string)
	if product == "" {
		product = h.config.DefaultProduct
	}
	if product == "" {
		return nil, fmt.Errorf("product parameter is required (or set AHA_DEFAULT_PRODUCT)")
	}

	releaseName, ok := params["release_name"].(string)
	if !ok || releaseName == "" {
		return nil, fmt.Errorf("release_name parameter is required")
	}

	features, err := h.syncDB.GetFeaturesByReleaseName(ctx, product, releaseName)
	if err != nil {
		return nil, fmt.Errorf("querying features by release name: %w", err)
	}

	return map[string]any{
		"product":      product,
		"release_name": releaseName,
		"count":        len(features),
		"features":     features,
	}, nil
}

// =============================================================================
// Phase 2 Write Tool Gaps
// =============================================================================

// CreateRelease creates a new release for a product.
func (h *ToolHandlers) CreateRelease(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id is required")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	var opts []aha.CreateReleaseOption

	if startDate, ok := params["start_date"].(string); ok && startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format (use YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithCreateReleaseStartDate(t))
	}

	if releaseDate, ok := params["release_date"].(string); ok && releaseDate != "" {
		t, err := time.Parse("2006-01-02", releaseDate)
		if err != nil {
			return nil, fmt.Errorf("invalid release_date format (use YYYY-MM-DD): %w", err)
		}
		opts = append(opts, aha.WithCreateReleaseDate(t))
	}

	if parkingLot, ok := params["parking_lot"].(bool); ok {
		opts = append(opts, aha.WithCreateReleaseParkingLot(parkingLot))
	}

	if theme, ok := params["theme"].(string); ok && theme != "" {
		opts = append(opts, aha.WithCreateReleaseTheme(theme))
	}

	release, err := h.client.CreateRelease(ctx, productID, name, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating release: %w", err)
	}

	return map[string]any{
		"id":            release.ID,
		"reference_num": release.ReferenceNum,
		"name":          release.Name,
		"start_date":    formatTimePtr(release.StartDate),
		"release_date":  formatTimePtr(release.ReleaseDate),
		"parking_lot":   release.ParkingLot,
		"theme":         release.Theme,
		"url":           release.URL,
	}, nil
}

// DeleteIdea deletes an idea by ID.
func (h *ToolHandlers) DeleteIdea(ctx context.Context, params map[string]any) (any, error) {
	ideaID, ok := params["idea_id"].(string)
	if !ok || ideaID == "" {
		return nil, fmt.Errorf("idea_id is required")
	}

	confirm, ok := params["confirm"].(bool)
	if !ok || !confirm {
		return nil, fmt.Errorf("confirm must be true to delete an idea (destructive operation)")
	}

	err := h.client.DeleteIdea(ctx, ideaID)
	if err != nil {
		return nil, fmt.Errorf("deleting idea: %w", err)
	}

	return map[string]any{
		"status":  "deleted",
		"idea_id": ideaID,
	}, nil
}

// ListIdeaCategories lists idea categories for a product.
func (h *ToolHandlers) ListIdeaCategories(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id is required")
	}

	list, err := h.client.ListProductIdeaCategories(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("listing idea categories: %w", err)
	}

	categories := make([]map[string]any, len(list.Categories))
	for i, cat := range list.Categories {
		categories[i] = map[string]any{
			"id":         cat.ID,
			"name":       cat.Name,
			"parent_id":  cat.ParentID,
			"project_id": cat.ProjectID,
			"created_at": cat.CreatedAt.Format(time.RFC3339),
		}
	}

	return map[string]any{
		"categories": categories,
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// GetFeatureIdeas lists ideas linked to a feature.
func (h *ToolHandlers) GetFeatureIdeas(ctx context.Context, params map[string]any) (any, error) {
	featureID, ok := params["feature_id"].(string)
	if !ok || featureID == "" {
		return nil, fmt.Errorf("feature_id is required")
	}

	var opts []aha.ListFeatureIdeasOption

	if page, ok := params["page"].(float64); ok && page > 0 {
		opts = append(opts, aha.WithFeatureIdeasPage(int(page)))
	}

	if perPage, ok := params["per_page"].(float64); ok && perPage > 0 {
		opts = append(opts, aha.WithFeatureIdeasPerPage(int(perPage)))
	}

	list, err := h.client.ListFeatureIdeas(ctx, featureID, opts...)
	if err != nil {
		return nil, fmt.Errorf("listing feature ideas: %w", err)
	}

	ideas := make([]map[string]any, len(list.Ideas))
	for i, idea := range list.Ideas {
		ideas[i] = map[string]any{
			"id":            idea.ID,
			"reference_num": idea.ReferenceNum,
			"name":          idea.Name,
			"description":   idea.Description,
			"votes":         idea.Votes,
			"score":         idea.Score,
			"created_at":    idea.CreatedAt.Format(time.RFC3339),
			"updated_at":    idea.UpdatedAt.Format(time.RFC3339),
		}
		if idea.WorkflowStatus != nil {
			ideas[i]["workflow_status"] = map[string]any{
				"id":   idea.WorkflowStatus.ID,
				"name": idea.WorkflowStatus.Name,
			}
		}
	}

	return map[string]any{
		"feature_id": featureID,
		"ideas":      ideas,
		"count":      len(ideas),
		"pagination": paginationToMap(list.Pagination),
	}, nil
}

// =============================================================================
// Initiative-Feature Relationship Tools
// =============================================================================

// ListInitiativeFeatures lists features linked to a specific initiative.
func (h *ToolHandlers) ListInitiativeFeatures(ctx context.Context, params map[string]any) (any, error) {
	initiativeID, ok := params["initiative_id"].(string)
	if !ok || initiativeID == "" {
		return nil, fmt.Errorf("initiative_id parameter is required")
	}

	initiative, err := h.client.GetInitiative(ctx, initiativeID)
	if err != nil {
		return nil, fmt.Errorf("getting initiative %s: %w", initiativeID, err)
	}

	features := make([]map[string]any, len(initiative.Features))
	for i, f := range initiative.Features {
		features[i] = map[string]any{
			"id":            f.ID,
			"reference_num": f.ReferenceNum,
			"name":          f.Name,
			"url":           f.URL,
		}
	}

	return map[string]any{
		"initiative_id":   initiative.ID,
		"initiative_name": initiative.Name,
		"features":        features,
		"total_features":  len(features),
	}, nil
}

// GetInitiativeWithFeatures retrieves an initiative with all its linked features.
func (h *ToolHandlers) GetInitiativeWithFeatures(ctx context.Context, params map[string]any) (any, error) {
	initiativeID, ok := params["initiative_id"].(string)
	if !ok || initiativeID == "" {
		return nil, fmt.Errorf("initiative_id parameter is required")
	}

	initiative, err := h.client.GetInitiative(ctx, initiativeID)
	if err != nil {
		return nil, fmt.Errorf("getting initiative %s: %w", initiativeID, err)
	}

	return initiativeToMap(initiative), nil
}

// ListInitiativesByTag lists initiatives filtered by a tag value from custom fields.
func (h *ToolHandlers) ListInitiativesByTag(ctx context.Context, params map[string]any) (any, error) {
	productID, ok := params["product_id"].(string)
	if !ok || productID == "" {
		return nil, fmt.Errorf("product_id parameter is required")
	}
	tag, ok := params["tag"].(string)
	if !ok || tag == "" {
		return nil, fmt.Errorf("tag parameter is required")
	}

	var allInitiatives []aha.InitiativeMeta
	page := 1
	for {
		list, err := h.client.ListProductInitiatives(ctx, productID, aha.WithPage(page), aha.WithPerPage(200))
		if err != nil {
			return nil, fmt.Errorf("listing product initiatives: %w", err)
		}
		allInitiatives = append(allInitiatives, list.Initiatives...)
		if list.Pagination.CurrentPage >= list.Pagination.TotalPages || len(list.Initiatives) == 0 {
			break
		}
		page++
	}

	var matched []map[string]any
	for _, meta := range allInitiatives {
		initiative, err := h.client.GetInitiative(ctx, meta.ID)
		if err != nil {
			continue
		}
		if initiativeHasTag(initiative, tag) {
			matched = append(matched, initiativeToMap(initiative))
		}
	}

	return map[string]any{
		"initiatives": matched,
		"total_count": len(matched),
		"filter_tag":  tag,
	}, nil
}

// initiativeHasTag checks if an initiative has a specific tag in its custom fields.
func initiativeHasTag(init *aha.Initiative, tag string) bool {
	for _, cf := range init.CustomFields {
		if !strings.Contains(strings.ToLower(cf.Key), "tag") && !strings.EqualFold(cf.Name, "Tags") {
			continue
		}
		raw, ok := cf.Value.([]byte)
		if !ok {
			continue
		}
		var tags []string
		if err := json.Unmarshal(raw, &tags); err != nil {
			continue
		}
		for _, t := range tags {
			if strings.EqualFold(t, tag) {
				return true
			}
		}
	}
	return false
}

// =============================================================================
// Helper Functions
// =============================================================================

// paginationToMap converts pagination to a map.
func paginationToMap(p aha.Pagination) map[string]any {
	return map[string]any{
		"total_records": p.TotalRecords,
		"total_pages":   p.TotalPages,
		"current_page":  p.CurrentPage,
	}
}

// formatTimePtr formats a time pointer to ISO8601 string or empty if nil.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
