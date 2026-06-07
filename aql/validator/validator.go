// Package validator provides semantic validation for AQL queries.
package validator

import (
	"fmt"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/schema"
)

// ValidationError represents a validation error with position information.
type ValidationError struct {
	Message string
	Pos     ast.Position
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Pos.Line, e.Pos.Column, e.Message)
}

// Validator validates AQL queries against the schema.
type Validator struct {
	errors []*ValidationError
	entity *schema.Entity
}

// New creates a new validator.
func New() *Validator {
	return &Validator{}
}

// Validate validates a query and returns any validation errors.
func (v *Validator) Validate(query *ast.Query) error {
	v.errors = nil
	v.entity = nil

	// Validate FROM clause
	if query.From == nil {
		return &ValidationError{Message: "missing FROM clause"}
	}
	v.validateFromClause(query.From)

	// Validate JOIN clauses
	for _, join := range query.Joins {
		v.validateJoinClause(join)
	}

	// Validate WHERE clause
	if query.Where != nil {
		v.validateWhereClause(query.Where)
	}

	// Validate SELECT clause
	if query.Select != nil {
		v.validateSelectClause(query.Select, query)
	}

	// Validate GROUP BY clause
	if query.GroupBy != nil {
		v.validateGroupByClause(query.GroupBy)
	}

	// Validate HAVING clause
	if query.Having != nil {
		v.validateHavingClause(query.Having, query)
	}

	// Validate ORDER BY clause
	if query.OrderBy != nil {
		v.validateOrderByClause(query.OrderBy)
	}

	// Validate LIMIT
	if query.Limit != nil && *query.Limit < 0 {
		v.addError("LIMIT must be non-negative", ast.Position{})
	}

	// Cross-validation: HAVING requires GROUP BY
	if query.Having != nil && query.GroupBy == nil {
		v.addError("HAVING requires GROUP BY", query.Having.Pos)
	}

	if len(v.errors) > 0 {
		return v.errors[0] // Return first error
	}
	return nil
}

// validateFromClause validates the FROM clause.
func (v *Validator) validateFromClause(from *ast.FromClause) {
	if !from.Entity.IsValid() {
		v.addError(fmt.Sprintf("invalid entity type: %s", from.Entity), from.Pos)
		return
	}

	v.entity = schema.GetEntity(from.Entity)
	if v.entity == nil {
		v.addError(fmt.Sprintf("unknown entity: %s", from.Entity), from.Pos)
	}
}

// validateWhereClause validates the WHERE clause.
func (v *Validator) validateWhereClause(where *ast.WhereClause) {
	v.validateExpr(where.Expr)
}

// validateExpr validates an expression.
func (v *Validator) validateExpr(expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		v.validateExpr(e.Left)
		v.validateExpr(e.Right)

	case *ast.NotExpr:
		v.validateExpr(e.Expr)

	case *ast.ParenExpr:
		v.validateExpr(e.Expr)

	case *ast.CompareExpr:
		v.validateCompareExpr(e)
	}
}

// validateCompareExpr validates a comparison expression.
func (v *Validator) validateCompareExpr(expr *ast.CompareExpr) {
	// Validate field exists
	field := v.lookupField(expr.Field)
	if field == nil {
		// Check if it's a custom field
		if schema.IsCustomFieldName(expr.Field.String()) {
			// Custom fields are allowed but not validated
			return
		}
		v.addError(fmt.Sprintf("unknown field: %s", expr.Field.String()), expr.Pos())
		return
	}

	// Validate field is filterable
	if !field.Filterable {
		v.addError(fmt.Sprintf("field %s is not filterable", expr.Field.String()), expr.Pos())
		return
	}

	// Validate operator/value type compatibility
	if expr.Value != nil {
		v.validateTypeCompatibility(field, expr.Op, expr.Value, expr.Pos())
	}
}

// lookupField looks up a field in the current entity.
func (v *Validator) lookupField(ref ast.FieldRef) *schema.Field {
	if v.entity == nil {
		return nil
	}

	// Handle qualified names (e.g., features.name)
	fieldName := ref.Name
	if ref.Qualifier != "" {
		// For now, ignore the qualifier - it should match the current entity
		fieldName = ref.Name
	}

	return v.entity.Field(fieldName)
}

// validateTypeCompatibility validates that the operator and value are compatible with the field type.
func (v *Validator) validateTypeCompatibility(field *schema.Field, op ast.CompareOp, value *ast.Value, pos ast.Position) {
	// IN/NOT IN requires a list
	if op == ast.OpIN || op == ast.OpNotIn {
		if value.Type != ast.ValueTypeStringList {
			v.addError(fmt.Sprintf("IN/NOT IN requires a list value for field %s", field.Name), pos)
		}
		return
	}

	// CONTAINS requires string
	if op == ast.OpContains || op == ast.OpLike {
		if field.Type != schema.FieldTypeString && field.Type != schema.FieldTypeStringArray {
			v.addError(fmt.Sprintf("CONTAINS/LIKE can only be used with string fields, got %s", field.Type), pos)
		}
		return
	}

	// Check type compatibility
	switch field.Type {
	case schema.FieldTypeString, schema.FieldTypeStringArray:
		if value.Type != ast.ValueTypeString && value.Type != ast.ValueTypeNull {
			v.addError(fmt.Sprintf("field %s expects string value", field.Name), pos)
		}

	case schema.FieldTypeInt:
		if value.Type != ast.ValueTypeInt && value.Type != ast.ValueTypeNull {
			v.addError(fmt.Sprintf("field %s expects integer value", field.Name), pos)
		}

	case schema.FieldTypeFloat:
		if value.Type != ast.ValueTypeFloat && value.Type != ast.ValueTypeInt && value.Type != ast.ValueTypeNull {
			v.addError(fmt.Sprintf("field %s expects numeric value", field.Name), pos)
		}

	case schema.FieldTypeBool:
		if value.Type != ast.ValueTypeBool && value.Type != ast.ValueTypeNull {
			v.addError(fmt.Sprintf("field %s expects boolean value", field.Name), pos)
		}

	case schema.FieldTypeDate, schema.FieldTypeDatetime:
		if value.Type != ast.ValueTypeTime && value.Type != ast.ValueTypeDuration && value.Type != ast.ValueTypeString && value.Type != ast.ValueTypeNull {
			v.addError(fmt.Sprintf("field %s expects date/time value", field.Name), pos)
		}
	}
}

// validateSelectClause validates the SELECT clause.
func (v *Validator) validateSelectClause(sel *ast.SelectClause, query *ast.Query) {
	hasAggregates := false
	hasNonAggregates := false

	for _, item := range sel.Items {
		if item.Star {
			hasNonAggregates = true
			continue
		}

		if item.Aggregate != nil {
			hasAggregates = true
			v.validateAggregateFunc(item.Aggregate)
		}

		if item.Field != nil {
			hasNonAggregates = true
			f := v.lookupField(*item.Field)
			if f == nil && !schema.IsCustomFieldName(item.Field.String()) {
				v.addError(fmt.Sprintf("unknown field in SELECT: %s", item.Field.String()), item.Field.Pos)
			}
		}
	}

	// If we have aggregates without GROUP BY, non-aggregate fields are invalid
	// (unless they're in the GROUP BY clause)
	// This is a warning-level issue in most SQL dialects - we allow it for now
	_ = hasAggregates && hasNonAggregates && query.GroupBy == nil
}

// validateAggregateFunc validates an aggregate function.
func (v *Validator) validateAggregateFunc(agg *ast.AggregateFunc) {
	// COUNT(*) is always valid
	if agg.Field == nil {
		if agg.Func != ast.AggCount {
			v.addError(fmt.Sprintf("%s requires a field argument", agg.Func), agg.Pos)
		}
		return
	}

	// Validate the field exists
	field := v.lookupField(*agg.Field)
	if field == nil && !schema.IsCustomFieldName(agg.Field.String()) {
		v.addError(fmt.Sprintf("unknown field in %s: %s", agg.Func, agg.Field.String()), agg.Pos)
		return
	}

	// SUM/AVG require numeric fields
	if field != nil {
		if agg.Func == ast.AggSum || agg.Func == ast.AggAvg {
			if field.Type != schema.FieldTypeInt && field.Type != schema.FieldTypeFloat {
				v.addError(fmt.Sprintf("%s requires a numeric field, got %s", agg.Func, field.Type), agg.Pos)
			}
		}
	}
}

// validateJoinClause validates a JOIN clause.
func (v *Validator) validateJoinClause(join *ast.JoinClause) {
	if !join.Entity.IsValid() {
		v.addError(fmt.Sprintf("invalid entity type in JOIN: %s", join.Entity), join.Pos)
		return
	}

	entity := schema.GetEntity(join.Entity)
	if entity == nil {
		v.addError(fmt.Sprintf("unknown entity in JOIN: %s", join.Entity), join.Pos)
	}

	// Validate join condition
	if join.Condition != nil {
		v.validateExpr(join.Condition)
	}
}

// validateGroupByClause validates a GROUP BY clause.
func (v *Validator) validateGroupByClause(groupBy *ast.GroupByClause) {
	for _, field := range groupBy.Fields {
		f := v.lookupField(field)
		if f == nil && !schema.IsCustomFieldName(field.String()) {
			v.addError(fmt.Sprintf("unknown field in GROUP BY: %s", field.String()), field.Pos)
		}
	}
}

// validateHavingClause validates a HAVING clause.
func (v *Validator) validateHavingClause(having *ast.HavingClause, _ *ast.Query) {
	// HAVING condition is validated like any other expression
	v.validateExpr(having.Expr)
}

// validateOrderByClause validates the ORDER BY clause.
func (v *Validator) validateOrderByClause(orderBy *ast.OrderByClause) {
	field := v.lookupField(orderBy.Field)
	if field == nil {
		if schema.IsCustomFieldName(orderBy.Field.String()) {
			v.addError(fmt.Sprintf("custom fields cannot be used in ORDER BY: %s", orderBy.Field.String()), orderBy.Pos)
		} else {
			v.addError(fmt.Sprintf("unknown field in ORDER BY: %s", orderBy.Field.String()), orderBy.Pos)
		}
		return
	}

	if !field.Sortable {
		v.addError(fmt.Sprintf("field %s is not sortable", orderBy.Field.String()), orderBy.Pos)
	}
}

// addError adds a validation error.
func (v *Validator) addError(message string, pos ast.Position) {
	v.errors = append(v.errors, &ValidationError{
		Message: message,
		Pos:     pos,
	})
}

// Errors returns all validation errors.
func (v *Validator) Errors() []*ValidationError {
	return v.errors
}
