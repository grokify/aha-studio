package planner

import (
	"testing"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/aql/parser"
)

func TestPlanSimpleFrom(t *testing.T) {
	tests := []struct {
		input  string
		entity ast.EntityType
	}{
		{"FROM features", ast.EntityFeatures},
		{"FROM ideas", ast.EntityIdeas},
		{"FROM releases", ast.EntityReleases},
		{"FROM initiatives", ast.EntityInitiatives},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser.New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			planner := New()
			plan := planner.Plan(query)

			if plan.Entity != tt.entity {
				t.Errorf("expected entity %s, got %s", tt.entity, plan.Entity)
			}
		})
	}
}

func TestPlanLimit(t *testing.T) {
	p := parser.New("FROM features LIMIT 10")
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if plan.Limit == nil {
		t.Fatal("expected Limit, got nil")
	}
	if *plan.Limit != 10 {
		t.Errorf("expected Limit 10, got %d", *plan.Limit)
	}
}

func TestPlanOrderBy(t *testing.T) {
	tests := []struct {
		input string
		field string
		dir   ast.SortDirection
	}{
		{"FROM features ORDER BY name", "name", ast.SortAsc},
		{"FROM features ORDER BY name ASC", "name", ast.SortAsc},
		{"FROM features ORDER BY updated_at DESC", "updated_at", ast.SortDesc},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser.New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			planner := New()
			plan := planner.Plan(query)

			if plan.OrderBy == nil {
				t.Fatal("expected OrderBy, got nil")
			}
			if plan.OrderBy.Field != tt.field {
				t.Errorf("expected field %s, got %s", tt.field, plan.OrderBy.Field)
			}
			if plan.OrderBy.Direction != tt.dir {
				t.Errorf("expected direction %s, got %s", tt.dir, plan.OrderBy.Direction)
			}
			// OrderBy requires pagination to fetch all for client-side sorting
			if !plan.RequiresPagination {
				t.Error("expected RequiresPagination=true with ORDER BY")
			}
		})
	}
}

func TestPlanPushableFilters(t *testing.T) {
	tests := []struct {
		input       string
		expectQuery string
		expectTag   string
	}{
		// name CONTAINS pushes to q parameter
		{`FROM features WHERE name CONTAINS "API"`, "API", ""},
		// tag = pushes to tag parameter
		{`FROM features WHERE tag = "urgent"`, "", "urgent"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser.New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			planner := New()
			plan := planner.Plan(query)

			if tt.expectQuery != "" && plan.APIParams.Query != tt.expectQuery {
				t.Errorf("expected API query %q, got %q", tt.expectQuery, plan.APIParams.Query)
			}
			if tt.expectTag != "" && plan.APIParams.Tag != tt.expectTag {
				t.Errorf("expected API tag %q, got %q", tt.expectTag, plan.APIParams.Tag)
			}
		})
	}
}

func TestPlanClientSideFilters(t *testing.T) {
	tests := []struct {
		input              string
		expectClientFilter bool
		filterField        string
		filterOp           ast.CompareOp
	}{
		// Non-pushable operators
		{`FROM features WHERE votes > 10`, true, "votes", ast.OpGT},
		{`FROM features WHERE votes < 100`, true, "votes", ast.OpLT},
		{`FROM features WHERE votes != 5`, true, "votes", ast.OpNE},

		// OR expressions are client-side
		{`FROM features WHERE status = "Done" OR status = "Closed"`, true, "", ""},

		// NOT expressions are client-side
		{`FROM features WHERE NOT status = "Done"`, true, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser.New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			planner := New()
			plan := planner.Plan(query)

			hasClientFilters := len(plan.ClientFilters) > 0
			if hasClientFilters != tt.expectClientFilter {
				t.Errorf("expected client filters: %v, got %d filters", tt.expectClientFilter, len(plan.ClientFilters))
			}

			if tt.expectClientFilter && !plan.RequiresPagination {
				t.Error("expected RequiresPagination=true with client-side filters")
			}

			if tt.filterField != "" && len(plan.ClientFilters) > 0 {
				if plan.ClientFilters[0].Field != tt.filterField {
					t.Errorf("expected filter field %s, got %s", tt.filterField, plan.ClientFilters[0].Field)
				}
			}
		})
	}
}

func TestPlanANDFilters(t *testing.T) {
	// AND filters should be split - pushable ones go to API, others to client
	input := `FROM features WHERE name CONTAINS "API" AND votes > 10`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	// name CONTAINS should be pushed to API query param
	if plan.APIParams.Query != "API" {
		t.Errorf("expected API query 'API', got %q", plan.APIParams.Query)
	}

	// votes > 10 should be client-side
	if len(plan.ClientFilters) == 0 {
		t.Error("expected client filters for votes > 10")
	}
}

func TestPlanJoins(t *testing.T) {
	// Note: Using simple JOIN condition since parser doesn't support field = field
	input := `FROM features f LEFT JOIN releases r ON release_id = "REL-1"`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if len(plan.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(plan.Joins))
	}

	join := plan.Joins[0]
	if join.Type != ast.JoinLeft {
		t.Errorf("expected LEFT join, got %s", join.Type)
	}
	if join.Entity != ast.EntityReleases {
		t.Errorf("expected releases, got %s", join.Entity)
	}
	if join.Alias != "r" {
		t.Errorf("expected alias r, got %s", join.Alias)
	}

	// Joins require pagination
	if !plan.RequiresPagination {
		t.Error("expected RequiresPagination=true with JOIN")
	}
}

func TestPlanSelectFields(t *testing.T) {
	tests := []struct {
		input  string
		fields []string
	}{
		{"SELECT * FROM features", nil},
		{"SELECT name FROM features", []string{"name"}},
		{"SELECT name, status FROM features", []string{"name", "status"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser.New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			planner := New()
			plan := planner.Plan(query)

			if tt.fields == nil {
				if len(plan.SelectFields) > 0 {
					t.Error("expected empty SelectFields for SELECT *")
				}
				return
			}

			if len(plan.SelectFields) != len(tt.fields) {
				t.Fatalf("expected %d fields, got %d", len(tt.fields), len(plan.SelectFields))
			}

			for i, f := range tt.fields {
				if plan.SelectFields[i] != f {
					t.Errorf("expected field %s, got %s", f, plan.SelectFields[i])
				}
			}
		})
	}
}

func TestPlanAggregates(t *testing.T) {
	tests := []struct {
		input      string
		aggCount   int
		firstFunc  ast.AggregateType
		firstField string
	}{
		{"SELECT COUNT(*) FROM features", 1, ast.AggCount, ""},
		{"SELECT COUNT(name) FROM features", 1, ast.AggCount, "name"},
		{"SELECT SUM(votes) FROM ideas", 1, ast.AggSum, "votes"},
		{"SELECT status, COUNT(*) FROM features GROUP BY status", 1, ast.AggCount, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser.New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			planner := New()
			plan := planner.Plan(query)

			if !plan.HasAggregates {
				t.Error("expected HasAggregates=true")
			}

			if len(plan.Aggregations) != tt.aggCount {
				t.Fatalf("expected %d aggregations, got %d", tt.aggCount, len(plan.Aggregations))
			}

			if plan.Aggregations[0].Func != tt.firstFunc {
				t.Errorf("expected %s, got %s", tt.firstFunc, plan.Aggregations[0].Func)
			}
			if plan.Aggregations[0].Field != tt.firstField {
				t.Errorf("expected field %q, got %q", tt.firstField, plan.Aggregations[0].Field)
			}
		})
	}
}

func TestPlanGroupBy(t *testing.T) {
	input := `SELECT status, COUNT(*) FROM features GROUP BY status`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if len(plan.GroupBy) != 1 {
		t.Fatalf("expected 1 group by field, got %d", len(plan.GroupBy))
	}
	if plan.GroupBy[0] != "status" {
		t.Errorf("expected status, got %s", plan.GroupBy[0])
	}

	// GROUP BY requires pagination to fetch all
	if !plan.RequiresPagination {
		t.Error("expected RequiresPagination=true with GROUP BY")
	}
}

func TestPlanHaving(t *testing.T) {
	input := `SELECT status, COUNT(*) AS cnt FROM features GROUP BY status HAVING cnt > 5`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if plan.Having == nil {
		t.Error("expected Having clause")
	}
}

func TestPlanUpdatedSince(t *testing.T) {
	// updated_at >= should push to updated_since API parameter
	input := `FROM features WHERE updated_at >= duration("30d")`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if plan.APIParams.UpdatedSince == nil {
		t.Error("expected UpdatedSince to be set")
	}
}

func TestPlanWorkflowStatus(t *testing.T) {
	// status = for ideas should push to workflow_status API parameter
	input := `FROM ideas WHERE status = "New"`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if plan.APIParams.WorkflowStatus != "New" {
		t.Errorf("expected WorkflowStatus 'New', got %q", plan.APIParams.WorkflowStatus)
	}
}

func TestPlanComplexQuery(t *testing.T) {
	// Complex query without JOIN
	// Note: Using "cnt" as alias since "count" is a reserved keyword
	input := `
		SELECT status, COUNT(*) AS cnt
		FROM features
		WHERE name CONTAINS "API"
		GROUP BY status
		ORDER BY cnt DESC
		LIMIT 10
	`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	// Verify all parts of the plan
	if plan.Entity != ast.EntityFeatures {
		t.Errorf("expected entity features, got %s", plan.Entity)
	}
	if plan.APIParams.Query != "API" {
		t.Errorf("expected API query 'API', got %q", plan.APIParams.Query)
	}
	if !plan.HasAggregates {
		t.Error("expected HasAggregates=true")
	}
	if len(plan.GroupBy) != 1 {
		t.Errorf("expected 1 group by field, got %d", len(plan.GroupBy))
	}
	if plan.OrderBy == nil {
		t.Error("expected OrderBy")
	}
	if plan.Limit == nil || *plan.Limit != 10 {
		t.Error("expected Limit 10")
	}
	if !plan.RequiresPagination {
		t.Error("expected RequiresPagination=true")
	}
}

func TestPlanProductID(t *testing.T) {
	input := `FROM features`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	// ProductID should be empty by default (set by caller)
	if plan.APIParams.ProductID != "" {
		t.Errorf("expected empty ProductID, got %q", plan.APIParams.ProductID)
	}

	// Can be set externally
	plan.APIParams.ProductID = "PLATFORM"
	if plan.APIParams.ProductID != "PLATFORM" {
		t.Error("ProductID should be settable")
	}
}

func TestPlanSubqueryScalar(t *testing.T) {
	// Test planning a scalar subquery: votes > (SELECT AVG(votes) FROM ideas)
	input := `FROM ideas WHERE votes > (SELECT AVG(votes) FROM ideas)`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	// Should have one subquery
	if len(plan.Subqueries) != 1 {
		t.Fatalf("expected 1 subquery, got %d", len(plan.Subqueries))
	}

	subquery := plan.Subqueries[0]

	// Subquery should be marked as scalar
	if !subquery.IsScalar {
		t.Error("expected IsScalar=true for > operator")
	}

	// Subquery should have a plan
	if subquery.Plan == nil {
		t.Fatal("expected subquery plan")
	}

	// Subquery plan should be for ideas entity
	if subquery.Plan.Entity != ast.EntityIdeas {
		t.Errorf("expected subquery entity ideas, got %s", subquery.Plan.Entity)
	}

	// Subquery plan should have aggregates
	if !subquery.Plan.HasAggregates {
		t.Error("expected subquery plan HasAggregates=true")
	}

	// Main plan should have client filter with subquery reference
	if len(plan.ClientFilters) != 1 {
		t.Fatalf("expected 1 client filter, got %d", len(plan.ClientFilters))
	}

	filter := plan.ClientFilters[0]
	if filter.SubqueryIndex != 0 {
		t.Errorf("expected SubqueryIndex 0, got %d", filter.SubqueryIndex)
	}

	// Main plan should require pagination (for client-side filtering)
	if !plan.RequiresPagination {
		t.Error("expected RequiresPagination=true with subquery")
	}
}

func TestPlanSubqueryIN(t *testing.T) {
	// Test planning an IN subquery: release_id IN (SELECT id FROM releases)
	input := `FROM features WHERE release_id IN (SELECT id FROM releases)`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	// Should have one subquery
	if len(plan.Subqueries) != 1 {
		t.Fatalf("expected 1 subquery, got %d", len(plan.Subqueries))
	}

	subquery := plan.Subqueries[0]

	// Subquery should NOT be marked as scalar (IN returns a list)
	if subquery.IsScalar {
		t.Error("expected IsScalar=false for IN operator")
	}

	// Subquery operator should be IN
	if subquery.Op != ast.OpIN {
		t.Errorf("expected Op IN, got %s", subquery.Op)
	}

	// Subquery field should be release_id
	if subquery.Field != "release_id" {
		t.Errorf("expected Field release_id, got %s", subquery.Field)
	}

	// Subquery plan should be for releases entity
	if subquery.Plan.Entity != ast.EntityReleases {
		t.Errorf("expected subquery entity releases, got %s", subquery.Plan.Entity)
	}
}

func TestPlanSubqueryNotIN(t *testing.T) {
	// Test planning a NOT IN subquery
	input := `FROM ideas WHERE status NOT IN (SELECT status FROM ideas WHERE votes > 100)`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	// Should have one subquery
	if len(plan.Subqueries) != 1 {
		t.Fatalf("expected 1 subquery, got %d", len(plan.Subqueries))
	}

	subquery := plan.Subqueries[0]

	// Subquery should NOT be marked as scalar
	if subquery.IsScalar {
		t.Error("expected IsScalar=false for NOT IN operator")
	}

	// Subquery operator should be NOT IN
	if subquery.Op != ast.OpNotIn {
		t.Errorf("expected Op NOT IN, got %s", subquery.Op)
	}

	// Subquery plan should have a client filter for votes > 100
	if len(subquery.Plan.ClientFilters) == 0 {
		t.Error("expected client filters in subquery plan")
	}
}

func TestPlanCustomFieldsInSelect(t *testing.T) {
	input := `SELECT name, custom.priority, custom.team FROM features`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if !plan.NeedsCustomFields {
		t.Error("expected NeedsCustomFields=true for SELECT with custom.* fields")
	}

	if !plan.RequiresPagination {
		t.Error("expected RequiresPagination=true when custom fields needed")
	}
}

func TestPlanCustomFieldsInWhere(t *testing.T) {
	input := `FROM features WHERE custom.priority = "High"`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if !plan.NeedsCustomFields {
		t.Error("expected NeedsCustomFields=true for WHERE with custom.* field")
	}

	// custom.priority should be a client-side filter
	if len(plan.ClientFilters) != 1 {
		t.Errorf("expected 1 client filter, got %d", len(plan.ClientFilters))
	}

	if plan.ClientFilters[0].Field != "custom.priority" {
		t.Errorf("expected filter field custom.priority, got %s", plan.ClientFilters[0].Field)
	}
}

func TestPlanCustomFieldsInOrderBy(t *testing.T) {
	input := `FROM features ORDER BY custom.priority DESC`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if !plan.NeedsCustomFields {
		t.Error("expected NeedsCustomFields=true for ORDER BY with custom.* field")
	}

	if plan.OrderBy == nil {
		t.Fatal("expected OrderBy")
	}
	if plan.OrderBy.Field != "custom.priority" {
		t.Errorf("expected OrderBy.Field custom.priority, got %s", plan.OrderBy.Field)
	}
}

func TestPlanCustomFieldsInGroupBy(t *testing.T) {
	input := `SELECT custom.team, COUNT(*) FROM features GROUP BY custom.team`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if !plan.NeedsCustomFields {
		t.Error("expected NeedsCustomFields=true for GROUP BY with custom.* field")
	}

	if len(plan.GroupBy) != 1 || plan.GroupBy[0] != "custom.team" {
		t.Errorf("expected GroupBy [custom.team], got %v", plan.GroupBy)
	}
}

func TestPlanNoCustomFields(t *testing.T) {
	input := `FROM features WHERE status = "Done" ORDER BY name`

	p := parser.New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	planner := New()
	plan := planner.Plan(query)

	if plan.NeedsCustomFields {
		t.Error("expected NeedsCustomFields=false when no custom.* fields referenced")
	}
}
