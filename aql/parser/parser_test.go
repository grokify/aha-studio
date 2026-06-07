package parser

import (
	"testing"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
)

func TestParseSimpleFrom(t *testing.T) {
	tests := []struct {
		input    string
		entity   ast.EntityType
		hasError bool
	}{
		{"FROM features", ast.EntityFeatures, false},
		{"FROM ideas", ast.EntityIdeas, false},
		{"FROM releases", ast.EntityReleases, false},
		{"FROM initiatives", ast.EntityInitiatives, false},
		{"FROM FEATURES", ast.EntityFeatures, false},
		{"from features", ast.EntityFeatures, false},
		{"FROM invalid", "", true},
		{"FROM", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if query.From.Entity != tt.entity {
				t.Errorf("expected entity %s, got %s", tt.entity, query.From.Entity)
			}
		})
	}
}

func TestParseFromWithAlias(t *testing.T) {
	tests := []struct {
		input  string
		entity ast.EntityType
		alias  string
	}{
		{"FROM features AS f", ast.EntityFeatures, "f"},
		{"FROM features f", ast.EntityFeatures, "f"},
		{"FROM ideas AS i", ast.EntityIdeas, "i"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if query.From.Entity != tt.entity {
				t.Errorf("expected entity %s, got %s", tt.entity, query.From.Entity)
			}
			if query.From.Alias != tt.alias {
				t.Errorf("expected alias %s, got %s", tt.alias, query.From.Alias)
			}
		})
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		input    string
		limit    int
		hasLimit bool
	}{
		{"FROM features LIMIT 10", 10, true},
		{"FROM features LIMIT 100", 100, true},
		{"FROM features LIMIT 1", 1, true},
		{"FROM features", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.hasLimit {
				if query.Limit == nil {
					t.Fatal("expected LIMIT, got nil")
				}
				if *query.Limit != tt.limit {
					t.Errorf("expected LIMIT %d, got %d", tt.limit, *query.Limit)
				}
			} else if query.Limit != nil {
				t.Error("expected no LIMIT")
			}
		})
	}
}

func TestParseOrderBy(t *testing.T) {
	tests := []struct {
		input string
		field string
		dir   ast.SortDirection
	}{
		{"FROM features ORDER BY name", "name", ast.SortAsc},
		{"FROM features ORDER BY name ASC", "name", ast.SortAsc},
		{"FROM features ORDER BY name DESC", "name", ast.SortDesc},
		{"FROM features ORDER BY updated_at DESC", "updated_at", ast.SortDesc},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if query.OrderBy == nil {
				t.Fatal("expected ORDER BY, got nil")
			}
			if query.OrderBy.Field.Name != tt.field {
				t.Errorf("expected field %s, got %s", tt.field, query.OrderBy.Field.Name)
			}
			if query.OrderBy.Dir != tt.dir {
				t.Errorf("expected direction %s, got %s", tt.dir, query.OrderBy.Dir)
			}
		})
	}
}

func TestParseWhereSimple(t *testing.T) {
	tests := []struct {
		input    string
		field    string
		op       ast.CompareOp
		valueStr string
	}{
		{`FROM features WHERE status = "Done"`, "status", ast.OpEQ, "Done"},
		{`FROM features WHERE name != "Test"`, "name", ast.OpNE, "Test"},
		{`FROM ideas WHERE votes > 5`, "votes", ast.OpGT, ""},
		{`FROM features WHERE votes >= 10`, "votes", ast.OpGE, ""},
		{`FROM features WHERE votes < 100`, "votes", ast.OpLT, ""},
		{`FROM features WHERE votes <= 50`, "votes", ast.OpLE, ""},
		{`FROM features WHERE name CONTAINS "API"`, "name", ast.OpContains, "API"},
		{`FROM features WHERE name LIKE "%test%"`, "name", ast.OpLike, "%test%"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if query.Where == nil {
				t.Fatal("expected WHERE, got nil")
			}

			cmp, ok := query.Where.Expr.(*ast.CompareExpr)
			if !ok {
				t.Fatal("expected CompareExpr")
			}
			if cmp.Field.Name != tt.field {
				t.Errorf("expected field %s, got %s", tt.field, cmp.Field.Name)
			}
			if cmp.Op != tt.op {
				t.Errorf("expected operator %s, got %s", tt.op, cmp.Op)
			}
		})
	}
}

func TestParseWhereIsNull(t *testing.T) {
	tests := []struct {
		input string
		field string
		op    ast.CompareOp
	}{
		{`FROM features WHERE description IS NULL`, "description", ast.OpIsNull},
		{`FROM features WHERE description IS NOT NULL`, "description", ast.OpIsNotNull},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmp, ok := query.Where.Expr.(*ast.CompareExpr)
			if !ok {
				t.Fatal("expected CompareExpr")
			}
			if cmp.Field.Name != tt.field {
				t.Errorf("expected field %s, got %s", tt.field, cmp.Field.Name)
			}
			if cmp.Op != tt.op {
				t.Errorf("expected operator %s, got %s", tt.op, cmp.Op)
			}
			if cmp.Value != nil {
				t.Error("expected nil value for IS NULL/IS NOT NULL")
			}
		})
	}
}

func TestParseWhereIN(t *testing.T) {
	input := `FROM features WHERE status IN ("New", "In Progress", "Ready")`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected CompareExpr")
	}
	if cmp.Op != ast.OpIN {
		t.Errorf("expected IN operator, got %s", cmp.Op)
	}
	if cmp.Value.Type != ast.ValueTypeStringList {
		t.Errorf("expected string list value, got %s", cmp.Value.Type)
	}
	if len(cmp.Value.Strings) != 3 {
		t.Errorf("expected 3 strings, got %d", len(cmp.Value.Strings))
	}
	expected := []string{"New", "In Progress", "Ready"}
	for i, s := range expected {
		if cmp.Value.Strings[i] != s {
			t.Errorf("expected %q, got %q", s, cmp.Value.Strings[i])
		}
	}
}

func TestParseWhereNotIn(t *testing.T) {
	input := `FROM features WHERE status NOT IN ("Closed", "Won't Fix")`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected CompareExpr")
	}
	if cmp.Op != ast.OpNotIn {
		t.Errorf("expected NOT IN operator, got %s", cmp.Op)
	}
}

func TestParseWhereAND(t *testing.T) {
	input := `FROM features WHERE status = "Done" AND votes > 10`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bin, ok := query.Where.Expr.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("expected BinaryExpr")
	}
	if bin.Op != ast.OpAnd {
		t.Errorf("expected AND operator, got %s", bin.Op)
	}

	// Check left side
	left, ok := bin.Left.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected left CompareExpr")
	}
	if left.Field.Name != "status" {
		t.Errorf("expected status, got %s", left.Field.Name)
	}

	// Check right side
	right, ok := bin.Right.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected right CompareExpr")
	}
	if right.Field.Name != "votes" {
		t.Errorf("expected votes, got %s", right.Field.Name)
	}
}

func TestParseWhereOR(t *testing.T) {
	input := `FROM features WHERE status = "Done" OR status = "Closed"`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bin, ok := query.Where.Expr.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("expected BinaryExpr")
	}
	if bin.Op != ast.OpOr {
		t.Errorf("expected OR operator, got %s", bin.Op)
	}
}

func TestParseWhereNOT(t *testing.T) {
	input := `FROM features WHERE NOT status = "Closed"`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	notExpr, ok := query.Where.Expr.(*ast.NotExpr)
	if !ok {
		t.Fatal("expected NotExpr")
	}

	_, ok = notExpr.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected inner CompareExpr")
	}
}

func TestParseWherePrecedence(t *testing.T) {
	// NOT has higher precedence than AND, which has higher precedence than OR
	input := `FROM features WHERE a = 1 OR b = 2 AND c = 3`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should parse as: a = 1 OR (b = 2 AND c = 3)
	or, ok := query.Where.Expr.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("expected outer OR BinaryExpr")
	}
	if or.Op != ast.OpOr {
		t.Errorf("expected OR, got %s", or.Op)
	}

	and, ok := or.Right.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("expected right side to be AND BinaryExpr")
	}
	if and.Op != ast.OpAnd {
		t.Errorf("expected AND, got %s", and.Op)
	}
}

func TestParseWhereParens(t *testing.T) {
	input := `FROM features WHERE (a = 1 OR b = 2) AND c = 3`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should parse as: (a = 1 OR b = 2) AND c = 3
	and, ok := query.Where.Expr.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("expected outer AND BinaryExpr")
	}
	if and.Op != ast.OpAnd {
		t.Errorf("expected AND, got %s", and.Op)
	}

	paren, ok := and.Left.(*ast.ParenExpr)
	if !ok {
		t.Fatal("expected left side to be ParenExpr")
	}

	or, ok := paren.Expr.(*ast.BinaryExpr)
	if !ok {
		t.Fatal("expected inner OR BinaryExpr")
	}
	if or.Op != ast.OpOr {
		t.Errorf("expected OR, got %s", or.Op)
	}
}

func TestParseSelectFields(t *testing.T) {
	tests := []struct {
		input  string
		fields []string
	}{
		{"SELECT * FROM features", nil},
		{"SELECT name FROM features", []string{"name"}},
		{"SELECT name, status FROM features", []string{"name", "status"}},
		{"SELECT name, status, votes FROM features", []string{"name", "status", "votes"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if query.Select == nil {
				t.Fatal("expected SELECT, got nil")
			}

			if tt.fields == nil {
				// SELECT *
				if len(query.Select.Items) != 1 || !query.Select.Items[0].Star {
					t.Error("expected SELECT *")
				}
				return
			}

			if len(query.Select.Items) != len(tt.fields) {
				t.Fatalf("expected %d fields, got %d", len(tt.fields), len(query.Select.Items))
			}

			for i, f := range tt.fields {
				if query.Select.Items[i].Field == nil {
					t.Errorf("expected field at position %d", i)
					continue
				}
				if query.Select.Items[i].Field.Name != f {
					t.Errorf("expected %s, got %s", f, query.Select.Items[i].Field.Name)
				}
			}
		})
	}
}

func TestParseSelectWithAlias(t *testing.T) {
	input := `SELECT name AS feature_name, votes AS vote_count FROM features`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		field string
		alias string
	}{
		{"name", "feature_name"},
		{"votes", "vote_count"},
	}

	for i, e := range expected {
		item := query.Select.Items[i]
		if item.Field.Name != e.field {
			t.Errorf("expected field %s, got %s", e.field, item.Field.Name)
		}
		if item.Alias != e.alias {
			t.Errorf("expected alias %s, got %s", e.alias, item.Alias)
		}
	}
}

func TestParseSelectAggregates(t *testing.T) {
	tests := []struct {
		input string
		agg   ast.AggregateType
		field string
	}{
		{"SELECT COUNT(*) FROM features", ast.AggCount, ""},
		{"SELECT COUNT(name) FROM features", ast.AggCount, "name"},
		{"SELECT SUM(votes) FROM ideas", ast.AggSum, "votes"},
		{"SELECT AVG(votes) FROM ideas", ast.AggAvg, "votes"},
		{"SELECT MIN(votes) FROM ideas", ast.AggMin, "votes"},
		{"SELECT MAX(votes) FROM ideas", ast.AggMax, "votes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if query.Select == nil || len(query.Select.Items) == 0 {
				t.Fatal("expected SELECT items")
			}

			item := query.Select.Items[0]
			if item.Aggregate == nil {
				t.Fatal("expected aggregate function")
			}
			if item.Aggregate.Func != tt.agg {
				t.Errorf("expected %s, got %s", tt.agg, item.Aggregate.Func)
			}
			if tt.field == "" {
				if item.Aggregate.Field != nil {
					t.Error("expected nil field for COUNT(*)")
				}
			} else {
				if item.Aggregate.Field == nil || item.Aggregate.Field.Name != tt.field {
					t.Errorf("expected field %s", tt.field)
				}
			}
		})
	}
}

func TestParseSelectDistinct(t *testing.T) {
	input := `SELECT DISTINCT status FROM features`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.Select == nil {
		t.Fatal("expected SELECT")
	}
	if !query.Select.Distinct {
		t.Error("expected DISTINCT")
	}
}

func TestParseSelectCountDistinct(t *testing.T) {
	input := `SELECT COUNT(DISTINCT status) FROM features`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := query.Select.Items[0]
	if item.Aggregate == nil {
		t.Fatal("expected aggregate")
	}
	if !item.Aggregate.Distinct {
		t.Error("expected DISTINCT in aggregate")
	}
}

func TestParseGroupBy(t *testing.T) {
	input := `SELECT status, COUNT(*) FROM features GROUP BY status`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.GroupBy == nil {
		t.Fatal("expected GROUP BY")
	}
	if len(query.GroupBy.Fields) != 1 {
		t.Fatalf("expected 1 group by field, got %d", len(query.GroupBy.Fields))
	}
	if query.GroupBy.Fields[0].Name != "status" {
		t.Errorf("expected status, got %s", query.GroupBy.Fields[0].Name)
	}
}

func TestParseGroupByMultiple(t *testing.T) {
	input := `SELECT status, product, COUNT(*) FROM features GROUP BY status, product`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.GroupBy == nil {
		t.Fatal("expected GROUP BY")
	}
	if len(query.GroupBy.Fields) != 2 {
		t.Fatalf("expected 2 group by fields, got %d", len(query.GroupBy.Fields))
	}
	if query.GroupBy.Fields[0].Name != "status" {
		t.Errorf("expected status, got %s", query.GroupBy.Fields[0].Name)
	}
	if query.GroupBy.Fields[1].Name != "product" {
		t.Errorf("expected product, got %s", query.GroupBy.Fields[1].Name)
	}
}

func TestParseHaving(t *testing.T) {
	input := `SELECT status, COUNT(*) AS cnt FROM features GROUP BY status HAVING cnt > 10`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query.Having == nil {
		t.Fatal("expected HAVING")
	}

	cmp, ok := query.Having.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected CompareExpr in HAVING")
	}
	if cmp.Field.Name != "cnt" {
		t.Errorf("expected cnt, got %s", cmp.Field.Name)
	}
	if cmp.Op != ast.OpGT {
		t.Errorf("expected >, got %s", cmp.Op)
	}
}

func TestParseJoin(t *testing.T) {
	// Note: Parser currently supports field = "value" in ON conditions,
	// not field = field. Using simple condition for test.
	input := `FROM features f JOIN releases r ON release_id = "REL-1"`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(query.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(query.Joins))
	}

	join := query.Joins[0]
	if join.Type != ast.JoinInner {
		t.Errorf("expected INNER JOIN, got %s", join.Type)
	}
	if join.Entity != ast.EntityReleases {
		t.Errorf("expected releases, got %s", join.Entity)
	}
	if join.Alias != "r" {
		t.Errorf("expected alias r, got %s", join.Alias)
	}
}

func TestParseLeftJoin(t *testing.T) {
	input := `FROM features f LEFT JOIN releases r ON release_id = "REL-1"`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	join := query.Joins[0]
	if join.Type != ast.JoinLeft {
		t.Errorf("expected LEFT JOIN, got %s", join.Type)
	}
}

func TestParseRightJoin(t *testing.T) {
	input := `FROM features f RIGHT JOIN releases r ON release_id = "REL-1"`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	join := query.Joins[0]
	if join.Type != ast.JoinRight {
		t.Errorf("expected RIGHT JOIN, got %s", join.Type)
	}
}

func TestParseFunctionNow(t *testing.T) {
	input := `FROM features WHERE created_at >= now()`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected CompareExpr")
	}
	if cmp.Value.Type != ast.ValueTypeTime {
		t.Errorf("expected time value, got %s", cmp.Value.Type)
	}
	// Value should be close to now
	if time.Since(cmp.Value.Time) > time.Second {
		t.Error("now() value too old")
	}
}

func TestParseFunctionDuration(t *testing.T) {
	input := `FROM features WHERE updated_at >= duration("30d")`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected CompareExpr")
	}
	if cmp.Value.Type != ast.ValueTypeDuration {
		t.Errorf("expected duration value, got %s", cmp.Value.Type)
	}

	expected := 30 * 24 * time.Hour
	if cmp.Value.Duration != expected {
		t.Errorf("expected %v, got %v", expected, cmp.Value.Duration)
	}
}

func TestParseQualifiedField(t *testing.T) {
	input := `FROM features WHERE custom.priority = "High"`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatal("expected CompareExpr")
	}
	if cmp.Field.Qualifier != "custom" {
		t.Errorf("expected qualifier custom, got %s", cmp.Field.Qualifier)
	}
	if cmp.Field.Name != "priority" {
		t.Errorf("expected name priority, got %s", cmp.Field.Name)
	}
}

func TestParseValueTypes(t *testing.T) {
	tests := []struct {
		input     string
		valueType ast.ValueType
	}{
		{`FROM features WHERE name = "test"`, ast.ValueTypeString},
		{`FROM features WHERE votes = 10`, ast.ValueTypeInt},
		{`FROM features WHERE score = 3.14`, ast.ValueTypeFloat},
		{`FROM features WHERE active = true`, ast.ValueTypeBool},
		{`FROM features WHERE active = false`, ast.ValueTypeBool},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := New(tt.input)
			query, err := p.Parse()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			cmp, ok := query.Where.Expr.(*ast.CompareExpr)
			if !ok {
				t.Fatal("expected CompareExpr")
			}
			if cmp.Value.Type != tt.valueType {
				t.Errorf("expected %s, got %s", tt.valueType, cmp.Value.Type)
			}
		})
	}
}

func TestParseComplexQuery(t *testing.T) {
	// Complex query without JOIN (JOIN parsing has limitations)
	// Note: Using "cnt" as alias since "count" is a reserved keyword (COUNT)
	input := `
		SELECT status, COUNT(*) AS cnt
		FROM features
		WHERE status IN ("New", "In Progress")
			AND updated_at >= duration("30d")
		GROUP BY status
		HAVING cnt > 5
		ORDER BY cnt DESC
		LIMIT 10
	`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all clauses are present
	if query.Select == nil {
		t.Error("expected SELECT")
	}
	if query.From == nil {
		t.Error("expected FROM")
	}
	if query.Where == nil {
		t.Error("expected WHERE")
	}
	if query.GroupBy == nil {
		t.Error("expected GROUP BY")
	}
	if query.Having == nil {
		t.Error("expected HAVING")
	}
	if query.OrderBy == nil {
		t.Error("expected ORDER BY")
	}
	if query.Limit == nil {
		t.Error("expected LIMIT")
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		input string
		desc  string
	}{
		{"SELECT name", "missing FROM"},
		{"FROM", "missing entity"},
		{"FROM features WHERE", "incomplete WHERE"},
		{"FROM features WHERE name =", "missing value"},
		{"FROM features WHERE name IN (", "unclosed list"},
		{"FROM features ORDER", "missing BY"},
		{"FROM features LIMIT", "missing limit value"},
		{"FROM features LIMIT abc", "non-integer limit"},
		{"FROM features GROUP", "missing BY"},
		{"FROM features HAVING count > 5", "HAVING without GROUP BY"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			p := New(tt.input)
			_, err := p.Parse()
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseSubqueryScalar(t *testing.T) {
	// Test scalar subquery: votes > (SELECT AVG(votes) FROM ideas)
	input := `FROM ideas WHERE votes > (SELECT AVG(votes) FROM ideas)`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if query.From.Entity != ast.EntityIdeas {
		t.Errorf("expected entity ideas, got %s", query.From.Entity)
	}

	if query.Where == nil {
		t.Fatal("expected WHERE clause")
	}

	// Check that the WHERE clause is a comparison with a subquery
	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatalf("expected CompareExpr, got %T", query.Where.Expr)
	}

	if cmp.Field.Name != "votes" {
		t.Errorf("expected field votes, got %s", cmp.Field.Name)
	}

	if cmp.Op != ast.OpGT {
		t.Errorf("expected operator >, got %s", cmp.Op)
	}

	if cmp.Value == nil {
		t.Fatal("expected value")
	}

	if cmp.Value.Type != ast.ValueTypeSubquery {
		t.Errorf("expected subquery value type, got %s", cmp.Value.Type)
	}

	if cmp.Value.Subquery == nil {
		t.Fatal("expected subquery")
	}

	// Check subquery structure
	sub := cmp.Value.Subquery
	if sub.From.Entity != ast.EntityIdeas {
		t.Errorf("expected subquery entity ideas, got %s", sub.From.Entity)
	}

	if sub.Select == nil || len(sub.Select.Items) != 1 {
		t.Fatal("expected 1 select item in subquery")
	}

	if sub.Select.Items[0].Aggregate == nil {
		t.Fatal("expected aggregate in subquery")
	}

	if sub.Select.Items[0].Aggregate.Func != ast.AggAvg {
		t.Errorf("expected AVG, got %s", sub.Select.Items[0].Aggregate.Func)
	}
}

func TestParseSubqueryIN(t *testing.T) {
	// Test IN subquery: release_id IN (SELECT id FROM releases WHERE released = false)
	input := `FROM features WHERE release_id IN (SELECT id FROM releases WHERE released = false)`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if query.From.Entity != ast.EntityFeatures {
		t.Errorf("expected entity features, got %s", query.From.Entity)
	}

	if query.Where == nil {
		t.Fatal("expected WHERE clause")
	}

	// Check that the WHERE clause is a comparison with a subquery
	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatalf("expected CompareExpr, got %T", query.Where.Expr)
	}

	if cmp.Field.Name != "release_id" {
		t.Errorf("expected field release_id, got %s", cmp.Field.Name)
	}

	if cmp.Op != ast.OpIN {
		t.Errorf("expected operator IN, got %s", cmp.Op)
	}

	if cmp.Value == nil || cmp.Value.Type != ast.ValueTypeSubquery {
		t.Fatal("expected subquery value")
	}

	// Check subquery structure
	sub := cmp.Value.Subquery
	if sub.From.Entity != ast.EntityReleases {
		t.Errorf("expected subquery entity releases, got %s", sub.From.Entity)
	}

	if sub.Select == nil || len(sub.Select.Items) != 1 {
		t.Fatal("expected 1 select item in subquery")
	}

	if sub.Select.Items[0].Field == nil || sub.Select.Items[0].Field.Name != "id" {
		t.Error("expected select id in subquery")
	}

	// Check subquery WHERE clause
	if sub.Where == nil {
		t.Fatal("expected WHERE in subquery")
	}
}

func TestParseSubqueryWithGroupBy(t *testing.T) {
	// Test subquery with GROUP BY and HAVING
	// Note: HAVING clause uses alias 'cnt' since parser requires field reference
	input := `FROM features WHERE release_id IN (SELECT release_id, COUNT(*) AS cnt FROM features GROUP BY release_id HAVING cnt > 5)`

	p := New(input)
	query, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Check that subquery parsed correctly
	cmp, ok := query.Where.Expr.(*ast.CompareExpr)
	if !ok {
		t.Fatalf("expected CompareExpr, got %T", query.Where.Expr)
	}

	sub := cmp.Value.Subquery
	if sub.GroupBy == nil || len(sub.GroupBy.Fields) != 1 {
		t.Fatal("expected GROUP BY in subquery")
	}

	if sub.Having == nil {
		t.Fatal("expected HAVING in subquery")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"60m", 60 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"500ms", 500 * time.Millisecond, false},
		{"", 0, true},
		{"d", 0, true},
		{"30x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, d)
			}
		})
	}
}

// Mutation tests

func TestParseInsertStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		entity   ast.EntityType
		columns  []string
		values   []string
		hasError bool
	}{
		{
			name:    "basic insert",
			input:   `INSERT INTO features (name) VALUES ('New Feature')`,
			entity:  ast.EntityFeatures,
			columns: []string{"name"},
			values:  []string{"New Feature"},
		},
		{
			name:    "insert multiple columns",
			input:   `INSERT INTO features (name, description) VALUES ('Feature', 'A description')`,
			entity:  ast.EntityFeatures,
			columns: []string{"name", "description"},
			values:  []string{"Feature", "A description"},
		},
		{
			name:    "insert with double quotes",
			input:   `INSERT INTO features (name, status) VALUES ("My Feature", "In Progress")`,
			entity:  ast.EntityFeatures,
			columns: []string{"name", "status"},
			values:  []string{"My Feature", "In Progress"},
		},
		{
			name:    "insert into ideas",
			input:   `INSERT INTO ideas (name, description) VALUES ('Idea', 'Description')`,
			entity:  ast.EntityIdeas,
			columns: []string{"name", "description"},
			values:  []string{"Idea", "Description"},
		},
		{
			name:     "missing VALUES",
			input:    `INSERT INTO features (name)`,
			hasError: true,
		},
		{
			name:     "missing columns",
			input:    `INSERT INTO features VALUES ('test')`,
			hasError: true,
		},
		{
			name:     "mismatched columns and values",
			input:    `INSERT INTO features (name, description) VALUES ('only one')`,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.input)
			stmt, err := p.ParseStatement()

			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			insert, ok := stmt.(*ast.InsertStatement)
			if !ok {
				t.Fatalf("expected InsertStatement, got %T", stmt)
			}
			if insert.Entity != tt.entity {
				t.Errorf("expected entity %s, got %s", tt.entity, insert.Entity)
			}
			if len(insert.Columns) != len(tt.columns) {
				t.Errorf("expected %d columns, got %d", len(tt.columns), len(insert.Columns))
			}
			for i, col := range tt.columns {
				if insert.Columns[i] != col {
					t.Errorf("column %d: expected %s, got %s", i, col, insert.Columns[i])
				}
			}
			for i, val := range tt.values {
				if insert.Values[i].String != val {
					t.Errorf("value %d: expected %s, got %s", i, val, insert.Values[i].String)
				}
			}
		})
	}
}

func TestParseUpdateStatement(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		entity      ast.EntityType
		assignments int
		hasWhere    bool
		hasError    bool
	}{
		{
			name:        "basic update",
			input:       `UPDATE features SET status = 'Done'`,
			entity:      ast.EntityFeatures,
			assignments: 1,
			hasWhere:    false,
		},
		{
			name:        "update with WHERE",
			input:       `UPDATE features SET status = 'Done' WHERE name = 'Test'`,
			entity:      ast.EntityFeatures,
			assignments: 1,
			hasWhere:    true,
		},
		{
			name:        "update multiple fields",
			input:       `UPDATE features SET status = 'Done', name = 'Updated' WHERE id = '123'`,
			entity:      ast.EntityFeatures,
			assignments: 2,
			hasWhere:    true,
		},
		{
			name:        "update ideas",
			input:       `UPDATE ideas SET status = 'Under Review' WHERE votes > 10`,
			entity:      ast.EntityIdeas,
			assignments: 1,
			hasWhere:    true,
		},
		{
			name:     "missing SET",
			input:    `UPDATE features status = 'Done'`,
			hasError: true,
		},
		{
			name:     "missing assignment",
			input:    `UPDATE features SET`,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.input)
			stmt, err := p.ParseStatement()

			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			update, ok := stmt.(*ast.UpdateStatement)
			if !ok {
				t.Fatalf("expected UpdateStatement, got %T", stmt)
			}
			if update.Entity != tt.entity {
				t.Errorf("expected entity %s, got %s", tt.entity, update.Entity)
			}
			if len(update.Assignments) != tt.assignments {
				t.Errorf("expected %d assignments, got %d", tt.assignments, len(update.Assignments))
			}
			if tt.hasWhere && update.Where == nil {
				t.Error("expected WHERE clause, got nil")
			}
			if !tt.hasWhere && update.Where != nil {
				t.Error("expected no WHERE clause")
			}
		})
	}
}

func TestParseDeleteStatement(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		entity   ast.EntityType
		hasWhere bool
		hasError bool
	}{
		{
			name:     "basic delete",
			input:    `DELETE FROM features WHERE id = '123'`,
			entity:   ast.EntityFeatures,
			hasWhere: true,
		},
		{
			name:     "delete with complex WHERE",
			input:    `DELETE FROM features WHERE status = 'Obsolete' AND updated_at < now() - duration('90d')`,
			entity:   ast.EntityFeatures,
			hasWhere: true,
		},
		{
			name:     "delete from ideas",
			input:    `DELETE FROM ideas WHERE votes = 0`,
			entity:   ast.EntityIdeas,
			hasWhere: true,
		},
		{
			name:     "delete without WHERE",
			input:    `DELETE FROM features`,
			hasError: true,
		},
		{
			name:     "missing FROM",
			input:    `DELETE features WHERE id = '123'`,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.input)
			stmt, err := p.ParseStatement()

			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			del, ok := stmt.(*ast.DeleteStatement)
			if !ok {
				t.Fatalf("expected DeleteStatement, got %T", stmt)
			}
			if del.Entity != tt.entity {
				t.Errorf("expected entity %s, got %s", tt.entity, del.Entity)
			}
			if tt.hasWhere && del.Where == nil {
				t.Error("expected WHERE clause, got nil")
			}
		})
	}
}

func TestParseStatementQuery(t *testing.T) {
	// ParseStatement should also handle regular queries
	input := "FROM features WHERE status = 'Done' LIMIT 10"
	p := New(input)
	stmt, err := p.ParseStatement()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	query, ok := stmt.(*ast.Query)
	if !ok {
		t.Fatalf("expected Query, got %T", stmt)
	}
	if query.From.Entity != ast.EntityFeatures {
		t.Errorf("expected features, got %s", query.From.Entity)
	}
	if query.Limit == nil || *query.Limit != 10 {
		t.Error("expected LIMIT 10")
	}
}
