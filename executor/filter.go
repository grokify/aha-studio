package executor

import (
	"fmt"
	"strings"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// matchFilter checks if a record matches a filter.
func matchFilter(r result.Record, f planner.Filter) bool {
	// Empty field indicates a complex expression that we can't evaluate
	// In a full implementation, we'd store and evaluate the full expression
	if f.Field == "" {
		return true // Skip complex expressions for now
	}

	value := r.Get(f.Field)
	if value == nil && f.Op != ast.OpIsNull && f.Op != ast.OpIsNotNull {
		return false
	}

	// Check if this filter uses a subquery result
	if f.SubqueryIndex >= 0 {
		return compareWithSubqueryResult(value, f.Op, f.SubqueryResult)
	}

	return compareWithOp(value, f.Op, f.Value)
}

// compareWithSubqueryResult compares a value with a subquery result.
func compareWithSubqueryResult(recordValue any, op ast.CompareOp, subqueryResult any) bool {
	switch op {
	case ast.OpIN:
		// Subquery result should be a list
		if list, ok := subqueryResult.([]any); ok {
			return valueInAnyList(recordValue, list)
		}
		return false
	case ast.OpNotIn:
		if list, ok := subqueryResult.([]any); ok {
			return !valueInAnyList(recordValue, list)
		}
		return true
	case ast.OpEQ:
		return compareValues(recordValue, subqueryResult) == 0
	case ast.OpNE:
		return compareValues(recordValue, subqueryResult) != 0
	case ast.OpLT:
		return compareValues(recordValue, subqueryResult) < 0
	case ast.OpLE:
		return compareValues(recordValue, subqueryResult) <= 0
	case ast.OpGT:
		return compareValues(recordValue, subqueryResult) > 0
	case ast.OpGE:
		return compareValues(recordValue, subqueryResult) >= 0
	default:
		return false
	}
}

// valueInAnyList checks if a record value is in a list of any values.
func valueInAnyList(recordValue any, list []any) bool {
	for _, v := range list {
		if compareValues(recordValue, v) == 0 {
			return true
		}
	}
	return false
}

// compareWithOp compares a value with a filter value using the given operator.
func compareWithOp(recordValue any, op ast.CompareOp, filterValue *ast.Value) bool {
	switch op {
	case ast.OpEQ:
		return valuesEqual(recordValue, filterValue)
	case ast.OpNE:
		return !valuesEqual(recordValue, filterValue)
	case ast.OpLT:
		return compareValues(recordValue, filterValueToAny(filterValue)) < 0
	case ast.OpLE:
		return compareValues(recordValue, filterValueToAny(filterValue)) <= 0
	case ast.OpGT:
		return compareValues(recordValue, filterValueToAny(filterValue)) > 0
	case ast.OpGE:
		return compareValues(recordValue, filterValueToAny(filterValue)) >= 0
	case ast.OpIN:
		return valueInList(recordValue, filterValue)
	case ast.OpNotIn:
		return !valueInList(recordValue, filterValue)
	case ast.OpContains:
		return valueContains(recordValue, filterValue)
	case ast.OpLike:
		return valueLike(recordValue, filterValue)
	case ast.OpIsNull:
		return recordValue == nil
	case ast.OpIsNotNull:
		return recordValue != nil
	default:
		return false
	}
}

// valuesEqual checks if a record value equals a filter value.
func valuesEqual(recordValue any, filterValue *ast.Value) bool {
	if filterValue == nil {
		return recordValue == nil
	}

	switch filterValue.Type {
	case ast.ValueTypeString:
		if s, ok := recordValue.(string); ok {
			return strings.EqualFold(s, filterValue.String)
		}
	case ast.ValueTypeInt:
		return toInt64(recordValue) == filterValue.Int
	case ast.ValueTypeFloat:
		return toFloat64(recordValue) == filterValue.Float
	case ast.ValueTypeBool:
		if b, ok := recordValue.(bool); ok {
			return b == filterValue.Bool
		}
	case ast.ValueTypeTime:
		if t, ok := recordValue.(time.Time); ok {
			return t.Equal(filterValue.Time)
		}
	case ast.ValueTypeNull:
		return recordValue == nil
	}
	return false
}

// valueInList checks if a record value is in a list.
func valueInList(recordValue any, filterValue *ast.Value) bool {
	if filterValue == nil || filterValue.Type != ast.ValueTypeStringList {
		return false
	}

	recordStr := fmt.Sprintf("%v", recordValue)
	for _, s := range filterValue.Strings {
		if strings.EqualFold(recordStr, s) {
			return true
		}
	}
	return false
}

// valueContains checks if a record value contains a substring.
func valueContains(recordValue any, filterValue *ast.Value) bool {
	if filterValue == nil {
		return false
	}

	recordStr := fmt.Sprintf("%v", recordValue)
	return strings.Contains(strings.ToLower(recordStr), strings.ToLower(filterValue.String))
}

// valueLike checks if a record value matches a LIKE pattern.
// Supports % as wildcard.
func valueLike(recordValue any, filterValue *ast.Value) bool {
	if filterValue == nil {
		return false
	}

	recordStr := strings.ToLower(fmt.Sprintf("%v", recordValue))
	pattern := strings.ToLower(filterValue.String)

	// Simple LIKE implementation
	// % at start and end = contains
	// % at end only = starts with
	// % at start only = ends with
	// No % = exact match

	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		return strings.Contains(recordStr, pattern[1:len(pattern)-1])
	}
	if strings.HasSuffix(pattern, "%") {
		return strings.HasPrefix(recordStr, pattern[:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "%") {
		return strings.HasSuffix(recordStr, pattern[1:])
	}
	return recordStr == pattern
}

// compareValues compares two values and returns -1, 0, or 1.
func compareValues(a, b any) int {
	// Handle nil
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// Compare by type
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return strings.Compare(av, bv)
		}
	case int:
		return compareInt64(int64(av), toInt64(b))
	case int64:
		return compareInt64(av, toInt64(b))
	case float64:
		return compareFloat64(av, toFloat64(b))
	case bool:
		if bv, ok := b.(bool); ok {
			if av == bv {
				return 0
			}
			if av {
				return 1
			}
			return -1
		}
	case time.Time:
		if bv, ok := b.(time.Time); ok {
			if av.Before(bv) {
				return -1
			}
			if av.After(bv) {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison
	return strings.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
}

func compareInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFloat64(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	}
	return 0
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	}
	return 0
}

func filterValueToAny(v *ast.Value) any {
	if v == nil {
		return nil
	}
	switch v.Type {
	case ast.ValueTypeString:
		return v.String
	case ast.ValueTypeInt:
		return v.Int
	case ast.ValueTypeFloat:
		return v.Float
	case ast.ValueTypeBool:
		return v.Bool
	case ast.ValueTypeTime:
		return v.Time
	case ast.ValueTypeDuration:
		return v.Duration
	case ast.ValueTypeNull:
		return nil
	}
	return nil
}
