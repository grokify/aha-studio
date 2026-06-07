// Package ast defines the abstract syntax tree types for AQL (Aha Query Language).
package ast

import "time"

// Query represents a complete AQL query.
type Query struct {
	Select  *SelectClause
	From    *FromClause
	Joins   []*JoinClause
	Where   *WhereClause
	GroupBy *GroupByClause
	Having  *HavingClause
	OrderBy *OrderByClause
	Limit   *int
}

// HasAggregates returns true if the query contains aggregate functions.
func (q *Query) HasAggregates() bool {
	if q.Select == nil {
		return false
	}
	for _, item := range q.Select.Items {
		if item.Aggregate != nil {
			return true
		}
	}
	return false
}

// FromClause specifies the entity type to query.
type FromClause struct {
	Entity EntityType
	Alias  string // optional table alias (e.g., FROM features AS f)
	Pos    Position
}

// EntityType represents the type of Aha entity being queried.
type EntityType string

// Valid entity types.
const (
	EntityComments     EntityType = "comments"
	EntityEpics        EntityType = "epics"
	EntityFeatures     EntityType = "features"
	EntityGoals        EntityType = "goals"
	EntityIdeas        EntityType = "ideas"
	EntityReleases     EntityType = "releases"
	EntityRequirements EntityType = "requirements"
	EntityInitiatives  EntityType = "initiatives"
	EntityProducts     EntityType = "products"
	EntityTags         EntityType = "tags"
	EntityUsers        EntityType = "users"
)

// IsValid returns true if the entity type is valid.
func (e EntityType) IsValid() bool {
	switch e {
	case EntityComments, EntityEpics, EntityFeatures, EntityGoals, EntityIdeas, EntityReleases, EntityRequirements, EntityInitiatives, EntityProducts, EntityTags, EntityUsers:
		return true
	}
	return false
}

// WhereClause contains the filter expression.
type WhereClause struct {
	Expr Expr
	Pos  Position
}

// SelectClause specifies which fields or expressions to return.
type SelectClause struct {
	Items    []SelectItem
	Distinct bool
	Pos      Position
}

// SelectItem represents a single item in a SELECT clause.
type SelectItem struct {
	// Field is set for simple field references
	Field *FieldRef
	// Aggregate is set for aggregate functions
	Aggregate *AggregateFunc
	// Star is true for SELECT *
	Star bool
	// Alias is the optional AS alias
	Alias string
	Pos   Position
}

// String returns the string representation of the select item.
func (s SelectItem) String() string {
	var base string
	if s.Star {
		base = "*"
	} else if s.Aggregate != nil {
		base = s.Aggregate.String()
	} else if s.Field != nil {
		base = s.Field.String()
	}
	if s.Alias != "" {
		return base + " AS " + s.Alias
	}
	return base
}

// OutputName returns the name to use for this item in results.
func (s SelectItem) OutputName() string {
	if s.Alias != "" {
		return s.Alias
	}
	if s.Star {
		return "*"
	}
	if s.Aggregate != nil {
		return s.Aggregate.String()
	}
	if s.Field != nil {
		return s.Field.Name
	}
	return ""
}

// AggregateFunc represents an aggregate function call.
type AggregateFunc struct {
	Func     AggregateType
	Field    *FieldRef // nil for COUNT(*)
	Distinct bool      // COUNT(DISTINCT field)
	Pos      Position
}

// String returns the string representation of the aggregate function.
func (a AggregateFunc) String() string {
	if a.Field == nil {
		return string(a.Func) + "(*)"
	}
	if a.Distinct {
		return string(a.Func) + "(DISTINCT " + a.Field.String() + ")"
	}
	return string(a.Func) + "(" + a.Field.String() + ")"
}

// AggregateType represents the type of aggregate function.
type AggregateType string

// Aggregate function types.
const (
	AggCount AggregateType = "COUNT"
	AggSum   AggregateType = "SUM"
	AggAvg   AggregateType = "AVG"
	AggMin   AggregateType = "MIN"
	AggMax   AggregateType = "MAX"
)

// FieldRef references a field, possibly with a table/entity qualifier.
type FieldRef struct {
	Name      string
	Qualifier string // optional entity qualifier (e.g., "features.name")
	Pos       Position
}

// String returns the string representation of the field reference.
func (f FieldRef) String() string {
	if f.Qualifier != "" {
		return f.Qualifier + "." + f.Name
	}
	return f.Name
}

// JoinClause represents a JOIN between entities.
type JoinClause struct {
	Type      JoinType
	Entity    EntityType
	Alias     string // optional table alias
	Condition Expr   // ON condition
	Pos       Position
}

// JoinType represents the type of join.
type JoinType string

// Join types.
const (
	JoinInner JoinType = "JOIN"
	JoinLeft  JoinType = "LEFT JOIN"
	JoinRight JoinType = "RIGHT JOIN"
)

// GroupByClause specifies the grouping fields.
type GroupByClause struct {
	Fields []FieldRef
	Pos    Position
}

// HavingClause contains the filter expression for grouped results.
type HavingClause struct {
	Expr Expr
	Pos  Position
}

// OrderByClause specifies the sort order.
type OrderByClause struct {
	Field FieldRef
	Dir   SortDirection
	Pos   Position
}

// SortDirection specifies the sort order.
type SortDirection string

// Sort directions.
const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

// Position represents a position in the source query.
type Position struct {
	Offset int // byte offset
	Line   int // 1-based line number
	Column int // 1-based column number
}

// Value represents a literal value in an expression.
type Value struct {
	Type ValueType
	Raw  string // original string representation

	// Typed values (only one is set based on Type)
	String   string
	Int      int64
	Float    float64
	Bool     bool
	Time     time.Time
	Duration time.Duration
	Strings  []string // for IN clauses
	Subquery *Query   // for subquery expressions
}

// ValueType represents the type of a value.
type ValueType int

// Value types.
const (
	ValueTypeString ValueType = iota
	ValueTypeInt
	ValueTypeFloat
	ValueTypeBool
	ValueTypeTime
	ValueTypeDuration
	ValueTypeNull
	ValueTypeStringList
	ValueTypeSubquery // for scalar or list subqueries
)

// String returns the string representation of the value type.
func (v ValueType) String() string {
	switch v {
	case ValueTypeString:
		return "string"
	case ValueTypeInt:
		return "int"
	case ValueTypeFloat:
		return "float"
	case ValueTypeBool:
		return "bool"
	case ValueTypeTime:
		return "time"
	case ValueTypeDuration:
		return "duration"
	case ValueTypeNull:
		return "null"
	case ValueTypeStringList:
		return "string_list"
	case ValueTypeSubquery:
		return "subquery"
	default:
		return "unknown"
	}
}

// CompareOp represents a comparison operator.
type CompareOp string

// Comparison operators.
const (
	OpEQ        CompareOp = "="
	OpNE        CompareOp = "!="
	OpLT        CompareOp = "<"
	OpLE        CompareOp = "<="
	OpGT        CompareOp = ">"
	OpGE        CompareOp = ">="
	OpIN        CompareOp = "IN"
	OpNotIn     CompareOp = "NOT IN"
	OpContains  CompareOp = "CONTAINS"
	OpLike      CompareOp = "LIKE"
	OpIsNull    CompareOp = "IS NULL"
	OpIsNotNull CompareOp = "IS NOT NULL"
)

// LogicalOp represents a logical operator.
type LogicalOp string

// Logical operators.
const (
	OpAnd LogicalOp = "AND"
	OpOr  LogicalOp = "OR"
)
