package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/grokify/aha-studio/studio"
)

// OpenAPISpec is the OpenAPI 3.0 specification for the Aha Studio HTTP API.
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers,omitempty"`
	Paths      map[string]OpenAPIPath `json:"paths"`
	Components *OpenAPIComponents     `json:"components,omitempty"`
}

// OpenAPIInfo contains API metadata.
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// OpenAPIServer describes a server.
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// OpenAPIPath contains operations for a path.
type OpenAPIPath map[string]OpenAPIOperation

// OpenAPIOperation describes an API operation.
type OpenAPIOperation struct {
	Summary     string                     `json:"summary"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []OpenAPIParameter         `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

// OpenAPIParameter describes a parameter.
type OpenAPIParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"`
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required,omitempty"`
	Schema      OpenAPISchema `json:"schema"`
}

// OpenAPIRequestBody describes a request body.
type OpenAPIRequestBody struct {
	Description string                      `json:"description,omitempty"`
	Required    bool                        `json:"required,omitempty"`
	Content     map[string]OpenAPIMediaType `json:"content"`
}

// OpenAPIMediaType describes media type content.
type OpenAPIMediaType struct {
	Schema OpenAPISchema `json:"schema"`
}

// OpenAPIResponse describes a response.
type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

// OpenAPISchema describes a schema.
type OpenAPISchema struct {
	Type        string                   `json:"type,omitempty"`
	Format      string                   `json:"format,omitempty"`
	Description string                   `json:"description,omitempty"`
	Items       *OpenAPISchema           `json:"items,omitempty"`
	Properties  map[string]OpenAPISchema `json:"properties,omitempty"`
	Required    []string                 `json:"required,omitempty"`
	Ref         string                   `json:"$ref,omitempty"`
	Enum        []string                 `json:"enum,omitempty"`
}

// OpenAPIComponents contains reusable components.
type OpenAPIComponents struct {
	Schemas         map[string]OpenAPISchema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
}

// OpenAPISecurityScheme describes a security scheme.
type OpenAPISecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
	Description  string `json:"description,omitempty"`
}

// buildOpenAPISpec generates the OpenAPI specification.
func buildOpenAPISpec(addr string) OpenAPISpec {
	return OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       "Aha Studio API",
			Description: "HTTP API for AQL queries against Aha! data with caching, sync, and graph support",
			Version:     Version,
		},
		Servers: []OpenAPIServer{
			{URL: "http://" + addr, Description: "Local server"},
		},
		Paths: map[string]OpenAPIPath{
			"/health": {
				"get": OpenAPIOperation{
					Summary:     "Health check",
					Description: "Returns server health status",
					OperationID: "getHealth",
					Tags:        []string{"System"},
					Responses: map[string]OpenAPIResponse{
						"200": {
							Description: "Server is healthy",
							Content: map[string]OpenAPIMediaType{
								"application/json": {
									Schema: OpenAPISchema{
										Type: "object",
										Properties: map[string]OpenAPISchema{
											"status": {Type: "string", Enum: []string{"ok"}},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/version": {
				"get": OpenAPIOperation{
					Summary:     "Get version information",
					Description: "Returns server and studio version information",
					OperationID: "getVersion",
					Tags:        []string{"System"},
					Responses: map[string]OpenAPIResponse{
						"200": {
							Description: "Version information",
							Content: map[string]OpenAPIMediaType{
								"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/VersionResponse"}},
							},
						},
					},
				},
			},
			"/api/query": {
				"get": OpenAPIOperation{
					Summary:     "Execute AQL query (GET)",
					Description: "Execute an AQL query against Aha data",
					OperationID: "queryGet",
					Tags:        []string{"Query"},
					Parameters: []OpenAPIParameter{
						{Name: "aql", In: "query", Description: "AQL query string", Required: true, Schema: OpenAPISchema{Type: "string"}},
						{Name: "product", In: "query", Description: "Product context for query", Schema: OpenAPISchema{Type: "string"}},
						{Name: "mode", In: "query", Description: "Query mode", Schema: OpenAPISchema{Type: "string", Enum: []string{"api", "offline", "prefer-cache"}}},
					},
					Security: []map[string][]string{{"ApiKeyAuth": {}}},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Query results", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/QueryResponse"}}}},
						"400": {Description: "Invalid query", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
						"401": {Description: "Unauthorized"},
					},
				},
				"post": OpenAPIOperation{
					Summary:     "Execute AQL query (POST)",
					Description: "Execute an AQL query against Aha data",
					OperationID: "queryPost",
					Tags:        []string{"Query"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					RequestBody: &OpenAPIRequestBody{
						Required: true,
						Content: map[string]OpenAPIMediaType{
							"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/QueryRequest"}},
						},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Query results", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/QueryResponse"}}}},
						"400": {Description: "Invalid query", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
						"401": {Description: "Unauthorized"},
					},
				},
			},
			"/api/entities": {
				"get": OpenAPIOperation{
					Summary:     "List available entities",
					Description: "Returns list of queryable entity types with their fields",
					OperationID: "getEntities",
					Tags:        []string{"Reference"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Entity list", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/EntitiesResponse"}}}},
					},
				},
			},
			"/api/syntax": {
				"get": OpenAPIOperation{
					Summary:     "Get AQL syntax reference",
					Description: "Returns AQL syntax documentation and examples",
					OperationID: "getSyntax",
					Tags:        []string{"Reference"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Syntax reference", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/SyntaxResponse"}}}},
					},
				},
			},
			"/api/products": {
				"get": OpenAPIOperation{
					Summary:     "List available products",
					Description: "Returns list of Aha products available for querying",
					OperationID: "getProducts",
					Tags:        []string{"Reference"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Product list", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ProductsResponse"}}}},
					},
				},
			},
			"/api/cache/status": {
				"get": OpenAPIOperation{
					Summary:     "Get cache status",
					Description: "Returns current cache configuration and status",
					OperationID: "getCacheStatus",
					Tags:        []string{"Cache"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Cache status", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/CacheStatusResponse"}}}},
					},
				},
			},
			"/api/sync": {
				"post": OpenAPIOperation{
					Summary:     "Trigger data sync",
					Description: "Sync data from Aha API to local cache",
					OperationID: "triggerSync",
					Tags:        []string{"Cache"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					RequestBody: &OpenAPIRequestBody{
						Required: true,
						Content: map[string]OpenAPIMediaType{
							"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/SyncRequest"}},
						},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Sync results", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/SyncResponse"}}}},
						"503": {Description: "Cache unavailable", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
					},
				},
			},
			"/api/sync/status": {
				"get": OpenAPIOperation{
					Summary:     "Get sync status",
					Description: "Returns last sync times and record counts per entity",
					OperationID: "getSyncStatus",
					Tags:        []string{"Cache"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "product", In: "query", Description: "Product to check sync status for", Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Sync status", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/SyncStatusResponse"}}}},
					},
				},
			},
			"/api/filters": {
				"get": OpenAPIOperation{
					Summary:     "List saved filters",
					Description: "Returns all saved AQL filters",
					OperationID: "listFilters",
					Tags:        []string{"Filters"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "product", In: "query", Description: "Filter by product", Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Filter list", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/FilterListResponse"}}}},
					},
				},
				"post": OpenAPIOperation{
					Summary:     "Create saved filter",
					Description: "Create a new saved AQL filter",
					OperationID: "createFilter",
					Tags:        []string{"Filters"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					RequestBody: &OpenAPIRequestBody{
						Required: true,
						Content: map[string]OpenAPIMediaType{
							"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/CreateFilterRequest"}},
						},
					},
					Responses: map[string]OpenAPIResponse{
						"201": {Description: "Filter created", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/FilterResponse"}}}},
						"400": {Description: "Invalid request", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
					},
				},
			},
			"/api/filters/{id}": {
				"get": OpenAPIOperation{
					Summary:     "Get saved filter",
					Description: "Get a saved filter by ID",
					OperationID: "getFilter",
					Tags:        []string{"Filters"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Filter ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Filter details", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/FilterResponse"}}}},
						"404": {Description: "Filter not found", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
					},
				},
				"put": OpenAPIOperation{
					Summary:     "Update saved filter",
					Description: "Update a saved filter",
					OperationID: "updateFilter",
					Tags:        []string{"Filters"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Filter ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					RequestBody: &OpenAPIRequestBody{
						Required: true,
						Content: map[string]OpenAPIMediaType{
							"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/UpdateFilterRequest"}},
						},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Filter updated", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/FilterResponse"}}}},
						"404": {Description: "Filter not found", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
					},
				},
				"delete": OpenAPIOperation{
					Summary:     "Delete saved filter",
					Description: "Delete a saved filter",
					OperationID: "deleteFilter",
					Tags:        []string{"Filters"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Filter ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Filter deleted"},
						"404": {Description: "Filter not found", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
					},
				},
			},
			"/api/search": {
				"get": OpenAPIOperation{
					Summary:     "Full-text search",
					Description: "Search across cached entities",
					OperationID: "search",
					Tags:        []string{"Search"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "q", In: "query", Description: "Search query", Required: true, Schema: OpenAPISchema{Type: "string"}},
						{Name: "entities", In: "query", Description: "Comma-separated entity types to search", Schema: OpenAPISchema{Type: "string"}},
						{Name: "limit", In: "query", Description: "Maximum results (default 100)", Schema: OpenAPISchema{Type: "integer"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Search results", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/SearchResponse"}}}},
					},
				},
			},
			"/api/graph/status": {
				"get": OpenAPIOperation{
					Summary:     "Get graph database status",
					Description: "Returns Neo4j connection status",
					OperationID: "getGraphStatus",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Graph status", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/GraphStatusResponse"}}}},
					},
				},
			},
			"/api/graph/features/{id}/connected": {
				"get": OpenAPIOperation{
					Summary:     "Get connected features",
					Description: "Find features connected to a given feature via dependencies",
					OperationID: "getConnectedFeatures",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Feature ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
						{Name: "depth", In: "query", Description: "Maximum depth (default 3)", Schema: OpenAPISchema{Type: "integer"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Connected features"},
					},
				},
			},
			"/api/graph/path": {
				"get": OpenAPIOperation{
					Summary:     "Find path between nodes",
					Description: "Find shortest path between two nodes in the graph",
					OperationID: "findPath",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "from", In: "query", Description: "Source node ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
						{Name: "to", In: "query", Description: "Target node ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Path between nodes"},
					},
				},
			},
			"/api/graph/releases/{id}/dependencies": {
				"get": OpenAPIOperation{
					Summary:     "Get release dependencies",
					Description: "Find all dependencies for features in a release",
					OperationID: "getReleaseDependencies",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Release ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Release dependencies"},
					},
				},
			},
			"/api/graph/initiatives/{id}/impact": {
				"get": OpenAPIOperation{
					Summary:     "Get initiative impact",
					Description: "Get features and releases impacted by an initiative",
					OperationID: "getInitiativeImpact",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Initiative ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Initiative impact"},
					},
				},
			},
			"/api/graph/products/{id}/overview": {
				"get": OpenAPIOperation{
					Summary:     "Get product overview",
					Description: "Get comprehensive overview of product entities and relationships",
					OperationID: "getProductOverview",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					Parameters: []OpenAPIParameter{
						{Name: "id", In: "path", Description: "Product ID", Required: true, Schema: OpenAPISchema{Type: "string"}},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Product overview"},
					},
				},
			},
			"/api/graph/cypher": {
				"post": OpenAPIOperation{
					Summary:     "Execute Cypher query",
					Description: "Execute a raw Cypher query against the graph database",
					OperationID: "executeCypher",
					Tags:        []string{"Graph"},
					Security:    []map[string][]string{{"ApiKeyAuth": {}}},
					RequestBody: &OpenAPIRequestBody{
						Required: true,
						Content: map[string]OpenAPIMediaType{
							"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/CypherRequest"}},
						},
					},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Query results"},
						"400": {Description: "Invalid query", Content: map[string]OpenAPIMediaType{"application/json": {Schema: OpenAPISchema{Ref: "#/components/schemas/ErrorResponse"}}}},
					},
				},
			},
			"/metrics": {
				"get": OpenAPIOperation{
					Summary:     "Prometheus metrics",
					Description: "Returns metrics in Prometheus exposition format",
					OperationID: "getMetrics",
					Tags:        []string{"System"},
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "Prometheus metrics", Content: map[string]OpenAPIMediaType{"text/plain": {Schema: OpenAPISchema{Type: "string"}}}},
					},
				},
			},
		},
		Components: &OpenAPIComponents{
			SecuritySchemes: map[string]OpenAPISecurityScheme{
				"ApiKeyAuth": {
					Type:        "apiKey",
					Name:        "Authorization",
					In:          "header",
					Description: "API key for authentication (format: Bearer <key>)",
				},
			},
			Schemas: map[string]OpenAPISchema{
				"VersionResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"server":        {Type: "string", Description: "HTTP server version"},
						"studio":        {Type: "string", Description: "Studio library version"},
						"cache_enabled": {Type: "boolean", Description: "Whether cache is enabled"},
						"default_mode":  {Type: "string", Description: "Default query mode"},
					},
				},
				"QueryRequest": {
					Type:     "object",
					Required: []string{"aql"},
					Properties: map[string]OpenAPISchema{
						"aql":     {Type: "string", Description: "AQL query string"},
						"product": {Type: "string", Description: "Product context"},
						"mode":    {Type: "string", Enum: []string{"api", "offline", "prefer-cache"}, Description: "Query execution mode"},
					},
				},
				"QueryResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"entity":  {Type: "string", Description: "Entity type queried"},
						"count":   {Type: "integer", Description: "Number of records returned"},
						"records": {Type: "array", Items: &OpenAPISchema{Type: "object"}, Description: "Query result records"},
						"source":  {Type: "string", Description: "Data source (api, cache, offline)"},
						"error":   {Type: "string", Description: "Error message if query failed"},
					},
				},
				"ErrorResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"error":   {Type: "string", Description: "Error message"},
						"code":    {Type: "string", Description: "Error code"},
						"details": {Type: "string", Description: "Additional details"},
					},
				},
				"EntitiesResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"entities": {Type: "array", Items: &OpenAPISchema{Ref: "#/components/schemas/EntityInfo"}},
					},
				},
				"EntityInfo": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"name":        {Type: "string"},
						"description": {Type: "string"},
						"fields":      {Type: "array", Items: &OpenAPISchema{Type: "string"}},
					},
				},
				"SyntaxResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"syntax":    {Type: "string"},
						"operators": {Type: "array", Items: &OpenAPISchema{Type: "string"}},
						"examples":  {Type: "array", Items: &OpenAPISchema{Type: "string"}},
					},
				},
				"ProductsResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"products": {Type: "array", Items: &OpenAPISchema{Ref: "#/components/schemas/ProductInfo"}},
					},
				},
				"ProductInfo": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"id":              {Type: "string"},
						"reference_num":   {Type: "string"},
						"name":            {Type: "string"},
						"product_line_id": {Type: "string"},
					},
				},
				"CacheStatusResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"enabled":       {Type: "boolean"},
						"database_path": {Type: "string"},
						"default_mode":  {Type: "string"},
						"offline_only":  {Type: "boolean"},
					},
				},
				"SyncRequest": {
					Type:     "object",
					Required: []string{"product"},
					Properties: map[string]OpenAPISchema{
						"product":     {Type: "string", Description: "Product to sync"},
						"incremental": {Type: "boolean", Description: "Incremental sync only"},
						"entities":    {Type: "array", Items: &OpenAPISchema{Type: "string"}, Description: "Entity types to sync"},
					},
				},
				"SyncResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"product": {Type: "string"},
						"results": {Type: "array", Items: &OpenAPISchema{Ref: "#/components/schemas/SyncEntityResult"}},
						"error":   {Type: "string"},
					},
				},
				"SyncEntityResult": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"entity":       {Type: "string"},
						"record_count": {Type: "integer"},
						"duration":     {Type: "string"},
						"error":        {Type: "string"},
					},
				},
				"SyncStatusResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"product":  {Type: "string"},
						"entities": {Type: "object"},
					},
				},
				"FilterListResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"filters": {Type: "array", Items: &OpenAPISchema{Ref: "#/components/schemas/FilterResponse"}},
					},
				},
				"FilterResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"id":          {Type: "string"},
						"name":        {Type: "string"},
						"aql":         {Type: "string"},
						"product":     {Type: "string"},
						"description": {Type: "string"},
						"is_favorite": {Type: "boolean"},
						"created_at":  {Type: "string", Format: "date-time"},
						"updated_at":  {Type: "string", Format: "date-time"},
					},
				},
				"CreateFilterRequest": {
					Type:     "object",
					Required: []string{"name", "aql"},
					Properties: map[string]OpenAPISchema{
						"name":        {Type: "string"},
						"aql":         {Type: "string"},
						"product":     {Type: "string"},
						"description": {Type: "string"},
						"is_favorite": {Type: "boolean"},
					},
				},
				"UpdateFilterRequest": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"name":        {Type: "string"},
						"aql":         {Type: "string"},
						"product":     {Type: "string"},
						"description": {Type: "string"},
						"is_favorite": {Type: "boolean"},
					},
				},
				"SearchResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"query":   {Type: "string"},
						"results": {Type: "array", Items: &OpenAPISchema{Ref: "#/components/schemas/SearchResult"}},
						"count":   {Type: "integer"},
					},
				},
				"SearchResult": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"entity":      {Type: "string"},
						"id":          {Type: "string"},
						"name":        {Type: "string"},
						"description": {Type: "string"},
						"score":       {Type: "number"},
					},
				},
				"GraphStatusResponse": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"connected": {Type: "boolean"},
						"uri":       {Type: "string"},
						"database":  {Type: "string"},
					},
				},
				"CypherRequest": {
					Type:     "object",
					Required: []string{"cypher"},
					Properties: map[string]OpenAPISchema{
						"cypher": {Type: "string", Description: "Cypher query"},
						"params": {Type: "object", Description: "Query parameters"},
					},
				},
			},
		},
	}
}

// handleOpenAPI handles GET /api/openapi.json.
func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	spec := buildOpenAPISpec(s.config.Addr)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(spec); err != nil {
		s.logger.Error("failed to encode OpenAPI spec", "error", err)
	}
}

// studioVersion is used to avoid import cycle - set from studio package
var _ = studio.Version
