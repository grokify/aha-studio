// Package planner converts AQL ASTs to execution plans.
package planner

import (
	"time"

	"github.com/grokify/aha-studio/aql/ast"
)

// Plan represents an execution plan for an AQL query.
type Plan struct {
	// Entity is the entity type to query.
	Entity ast.EntityType

	// APIParams are parameters that can be pushed to the Aha API.
	APIParams APIParams

	// ClientFilters are filters that must be applied client-side.
	ClientFilters []Filter

	// SelectFields are the fields to return. Nil means all fields.
	SelectFields []string

	// SelectItems contains the original select items (for aggregate queries).
	SelectItems []ast.SelectItem

	// Aggregations defines aggregate functions to apply.
	Aggregations []Aggregation

	// GroupBy fields for grouping results.
	GroupBy []string

	// Having filter applied after aggregation.
	Having ast.Expr

	// Joins for multi-entity queries.
	Joins []*JoinPlan

	// OrderBy specifies the sort field and direction.
	OrderBy *OrderBy

	// Limit is the maximum number of results to return.
	Limit *int

	// RequiresPagination indicates if we need to fetch all pages
	// (true when client-side filtering or sorting is required).
	RequiresPagination bool

	// HasAggregates indicates if the query contains aggregate functions.
	HasAggregates bool

	// NeedsCustomFields indicates if the query references custom.* fields.
	// When true, the executor needs to fetch full entity details (not just meta).
	NeedsCustomFields bool

	// Subqueries contains any subqueries that need to be executed first.
	// Each subquery is identified by an index and will be resolved before
	// the main query is executed.
	Subqueries []*SubqueryPlan
}

// SubqueryPlan represents a subquery that needs to be executed.
type SubqueryPlan struct {
	// Query is the AST for the subquery.
	Query *ast.Query

	// Plan is the execution plan for the subquery (populated by planner).
	Plan *Plan

	// Index identifies this subquery for result substitution.
	Index int

	// Field is the main query field this subquery is compared against.
	Field string

	// Op is the comparison operator used.
	Op ast.CompareOp

	// IsScalar indicates if this subquery should return a single value
	// (true for comparisons like >, <, =; false for IN).
	IsScalar bool
}

// Aggregation represents an aggregate function to apply.
type Aggregation struct {
	Func     ast.AggregateType
	Field    string // empty for COUNT(*)
	Distinct bool
	Alias    string
}

// JoinPlan represents a planned join operation.
type JoinPlan struct {
	Type      ast.JoinType
	Entity    ast.EntityType
	Alias     string
	Condition ast.Expr
}

// APIParams represents parameters that can be pushed to the Aha API.
type APIParams struct {
	// Common parameters
	Query        string     // q parameter for name search
	Page         int        // page number
	PerPage      int        // results per page
	UpdatedSince *time.Time // updated_since filter
	Tag          string     // tag filter

	// Feature-specific parameters
	AssignedToUser string // assigned_to_user filter
	WorkflowStatus string // workflow_status filter (features and ideas)

	// Idea-specific parameters
	CreatedSince  *time.Time // created_since filter
	CreatedBefore *time.Time // created_before filter
	UserID        string     // user_id filter
	Spam          *bool      // spam filter
	Sort          string     // sort order (recent, trending, popular)

	// Context parameters (required for some endpoints)
	ProductID string // required for product-scoped queries
	ReleaseID string // required for release-scoped queries
}

// Filter represents a client-side filter.
type Filter struct {
	Field string
	Op    ast.CompareOp
	Value *ast.Value

	// SubqueryIndex is the index into Plan.Subqueries if this filter
	// uses a subquery. -1 indicates no subquery.
	SubqueryIndex int

	// SubqueryResult holds the resolved subquery result after execution.
	// For scalar subqueries: a single value (int64, float64, string)
	// For list subqueries: a []any of values
	SubqueryResult any
}

// OrderBy represents the sort order.
type OrderBy struct {
	Field     string
	Direction ast.SortDirection
}

// HasAPIFilters returns true if there are any API-level filters.
func (p *APIParams) HasAPIFilters() bool {
	return p.Query != "" ||
		p.UpdatedSince != nil ||
		p.Tag != "" ||
		p.AssignedToUser != "" ||
		p.WorkflowStatus != "" ||
		p.CreatedSince != nil ||
		p.CreatedBefore != nil ||
		p.UserID != "" ||
		p.Spam != nil
}

// IsEmpty returns true if there are no API parameters set.
func (p *APIParams) IsEmpty() bool {
	return !p.HasAPIFilters() && p.Page == 0 && p.PerPage == 0
}
