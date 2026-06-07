package planner

import (
	"strings"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/schema"
)

// Planner converts AQL queries to execution plans.
type Planner struct {
	entity *schema.Entity
}

// New creates a new planner.
func New() *Planner {
	return &Planner{}
}

// Plan creates an execution plan from a parsed query.
func (p *Planner) Plan(query *ast.Query) *Plan {
	plan := &Plan{
		Entity:    query.From.Entity,
		APIParams: APIParams{},
	}

	p.entity = schema.GetEntity(query.From.Entity)

	// Process JOIN clauses
	for _, join := range query.Joins {
		plan.Joins = append(plan.Joins, &JoinPlan{
			Type:      join.Type,
			Entity:    join.Entity,
			Alias:     join.Alias,
			Condition: join.Condition,
		})
	}

	// Process WHERE clause
	if query.Where != nil {
		p.planFilters(query.Where.Expr, plan)
	}

	// Process SELECT clause
	if query.Select != nil {
		p.planSelectClause(query.Select, plan)
	}

	// Process GROUP BY clause
	if query.GroupBy != nil {
		plan.GroupBy = make([]string, len(query.GroupBy.Fields))
		for i, f := range query.GroupBy.Fields {
			plan.GroupBy[i] = f.String()
		}
		plan.RequiresPagination = true // Must fetch all for grouping
	}

	// Process HAVING clause
	if query.Having != nil {
		plan.Having = query.Having.Expr
	}

	// Process ORDER BY clause
	if query.OrderBy != nil {
		plan.OrderBy = &OrderBy{
			Field:     query.OrderBy.Field.String(),
			Direction: query.OrderBy.Dir,
		}
		// Check if we can push sorting to the API
		// Most Aha endpoints don't support arbitrary sorting, so we'll do it client-side
		plan.RequiresPagination = true
	}

	// Process LIMIT
	if query.Limit != nil {
		plan.Limit = query.Limit
	}

	// If we have client-side filters, aggregations, or joins, we need all results
	if len(plan.ClientFilters) > 0 || plan.HasAggregates || len(plan.Joins) > 0 {
		plan.RequiresPagination = true
	}

	// Check if custom fields are referenced
	plan.NeedsCustomFields = p.checkCustomFieldsNeeded(plan)
	if plan.NeedsCustomFields {
		plan.RequiresPagination = true // Need to fetch all records individually
	}

	return plan
}

// checkCustomFieldsNeeded returns true if any custom.* fields are referenced.
func (p *Planner) checkCustomFieldsNeeded(plan *Plan) bool {
	// Check select fields
	for _, f := range plan.SelectFields {
		if strings.HasPrefix(f, "custom.") {
			return true
		}
	}

	// Check order by
	if plan.OrderBy != nil && strings.HasPrefix(plan.OrderBy.Field, "custom.") {
		return true
	}

	// Check client filters
	for _, f := range plan.ClientFilters {
		if strings.HasPrefix(f.Field, "custom.") {
			return true
		}
	}

	// Check group by
	for _, f := range plan.GroupBy {
		if strings.HasPrefix(f, "custom.") {
			return true
		}
	}

	return false
}

// planSelectClause plans the SELECT clause.
func (p *Planner) planSelectClause(sel *ast.SelectClause, plan *Plan) {
	// Store original select items for complex processing
	plan.SelectItems = sel.Items

	for _, item := range sel.Items {
		if item.Star {
			// SELECT * - no field restriction
			plan.SelectFields = nil
			continue
		}

		if item.Aggregate != nil {
			plan.HasAggregates = true
			agg := Aggregation{
				Func:     item.Aggregate.Func,
				Distinct: item.Aggregate.Distinct,
				Alias:    item.Alias,
			}
			if item.Aggregate.Field != nil {
				agg.Field = item.Aggregate.Field.String()
			}
			if agg.Alias == "" {
				agg.Alias = item.Aggregate.String()
			}
			plan.Aggregations = append(plan.Aggregations, agg)
		}

		if item.Field != nil {
			plan.SelectFields = append(plan.SelectFields, item.Field.String())
		}
	}
}

// planFilters extracts filters from an expression and categorizes them
// as API-pushable or client-side.
func (p *Planner) planFilters(expr ast.Expr, plan *Plan) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		if e.Op == ast.OpAnd {
			// AND expressions can be split
			p.planFilters(e.Left, plan)
			p.planFilters(e.Right, plan)
		} else {
			// OR expressions can't be pushed to API, evaluate client-side
			plan.ClientFilters = append(plan.ClientFilters, p.exprToFilter(e))
			plan.RequiresPagination = true
		}

	case *ast.NotExpr:
		// NOT expressions are evaluated client-side
		plan.ClientFilters = append(plan.ClientFilters, p.exprToFilter(e))
		plan.RequiresPagination = true

	case *ast.ParenExpr:
		p.planFilters(e.Expr, plan)

	case *ast.CompareExpr:
		// Check for subquery in the value
		if e.Value != nil && e.Value.Type == ast.ValueTypeSubquery {
			subqueryPlan := p.planSubquery(e, plan)
			plan.Subqueries = append(plan.Subqueries, subqueryPlan)
			// Add a client-side filter that will be resolved after subquery execution
			plan.ClientFilters = append(plan.ClientFilters, Filter{
				Field:         e.Field.String(),
				Op:            e.Op,
				Value:         e.Value,
				SubqueryIndex: len(plan.Subqueries) - 1,
			})
			plan.RequiresPagination = true
		} else if p.canPushToAPI(e, plan) {
			p.pushToAPI(e, plan)
		} else {
			plan.ClientFilters = append(plan.ClientFilters, Filter{
				Field:         e.Field.String(),
				Op:            e.Op,
				Value:         e.Value,
				SubqueryIndex: -1, // no subquery
			})
		}
	}
}

// canPushToAPI checks if a comparison can be pushed to the Aha API.
func (p *Planner) canPushToAPI(expr *ast.CompareExpr, _ *Plan) bool {
	if p.entity == nil {
		return false
	}

	field := p.entity.Field(expr.Field.Name)
	if field == nil {
		return false
	}

	// Field must have an API parameter mapping
	if !field.IsPushable() {
		return false
	}

	// Only certain operators can be pushed
	switch expr.Op {
	case ast.OpEQ, ast.OpContains:
		return true
	case ast.OpGE:
		// >= can be pushed for date fields (updated_since, created_since)
		return field.Type == schema.FieldTypeDatetime
	default:
		return false
	}
}

// pushToAPI adds a filter to the API parameters.
func (p *Planner) pushToAPI(expr *ast.CompareExpr, plan *Plan) {
	field := p.entity.Field(expr.Field.Name)
	if field == nil {
		return
	}

	switch field.APIParam {
	case "q":
		if expr.Value != nil {
			plan.APIParams.Query = expr.Value.String
		}

	case "tag":
		if expr.Value != nil {
			plan.APIParams.Tag = expr.Value.String
		}

	case "assigned_to_user":
		if expr.Value != nil {
			plan.APIParams.AssignedToUser = expr.Value.String
		}

	case "workflow_status":
		if expr.Value != nil {
			plan.APIParams.WorkflowStatus = expr.Value.String
		}

	case "updated_since":
		t := p.resolveTimeValue(expr)
		if t != nil {
			plan.APIParams.UpdatedSince = t
		}

	case "created_since":
		t := p.resolveTimeValue(expr)
		if t != nil {
			plan.APIParams.CreatedSince = t
		}

	case "created_before":
		t := p.resolveTimeValue(expr)
		if t != nil {
			plan.APIParams.CreatedBefore = t
		}

	case "user_id":
		if expr.Value != nil {
			plan.APIParams.UserID = expr.Value.String
		}

	case "spam":
		if expr.Value != nil && expr.Value.Type == ast.ValueTypeBool {
			plan.APIParams.Spam = &expr.Value.Bool
		}
	}
}

// resolveTimeValue resolves a time value from an expression.
func (p *Planner) resolveTimeValue(expr *ast.CompareExpr) *time.Time {
	if expr.Value == nil {
		return nil
	}

	switch expr.Value.Type {
	case ast.ValueTypeTime:
		return &expr.Value.Time
	case ast.ValueTypeDuration:
		// For expressions like "updated_at >= now() - duration('30d')"
		// we need to handle this specially
		// For now, treat duration as relative to now
		t := time.Now().Add(-expr.Value.Duration)
		return &t
	case ast.ValueTypeString:
		// Try to parse as time
		t, err := time.Parse(time.RFC3339, expr.Value.String)
		if err != nil {
			// Try date-only format
			t, err = time.Parse("2006-01-02", expr.Value.String)
			if err != nil {
				return nil
			}
		}
		return &t
	}
	return nil
}

// exprToFilter converts a complex expression to a single client-side filter.
// This is a simplified conversion - complex expressions are represented
// as a filter that will be evaluated by the executor.
func (p *Planner) exprToFilter(_ ast.Expr) Filter {
	// For complex expressions, we store the full expression structure
	// The executor will handle evaluation
	return Filter{
		Field:         "", // empty indicates complex expression
		Op:            "", // not a simple comparison
		Value:         nil,
		SubqueryIndex: -1, // no subquery
	}
}

// planSubquery creates a SubqueryPlan for a subquery expression.
func (p *Planner) planSubquery(expr *ast.CompareExpr, parentPlan *Plan) *SubqueryPlan {
	subquery := expr.Value.Subquery
	if subquery == nil {
		return nil
	}

	// Determine if this is a scalar subquery
	isScalar := isScalarOperator(expr.Op)

	// Create a new planner for the subquery (fresh entity context)
	subPlanner := New()
	subPlan := subPlanner.Plan(subquery)

	return &SubqueryPlan{
		Query:    subquery,
		Plan:     subPlan,
		Index:    len(parentPlan.Subqueries),
		Field:    expr.Field.String(),
		Op:       expr.Op,
		IsScalar: isScalar,
	}
}

// isScalarOperator returns true if the operator expects a scalar value (not a list).
func isScalarOperator(op ast.CompareOp) bool {
	switch op {
	case ast.OpIN, ast.OpNotIn:
		return false
	default:
		return true
	}
}
