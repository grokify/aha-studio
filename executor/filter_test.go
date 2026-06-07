package executor

import (
	"testing"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

func TestCompareWithOpEQ(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		// String equality (case-insensitive)
		{"Done", &ast.Value{Type: ast.ValueTypeString, String: "Done"}, true},
		{"done", &ast.Value{Type: ast.ValueTypeString, String: "Done"}, true},
		{"DONE", &ast.Value{Type: ast.ValueTypeString, String: "done"}, true},
		{"Pending", &ast.Value{Type: ast.ValueTypeString, String: "Done"}, false},

		// Integer equality
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 20}, false},
		{10, &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},

		// Float equality
		{3.14, &ast.Value{Type: ast.ValueTypeFloat, Float: 3.14}, true},
		{3.14, &ast.Value{Type: ast.ValueTypeFloat, Float: 2.71}, false},

		// Bool equality
		{true, &ast.Value{Type: ast.ValueTypeBool, Bool: true}, true},
		{true, &ast.Value{Type: ast.ValueTypeBool, Bool: false}, false},
		{false, &ast.Value{Type: ast.ValueTypeBool, Bool: false}, true},

		// Null
		{nil, &ast.Value{Type: ast.ValueTypeNull}, true},
		{"something", &ast.Value{Type: ast.ValueTypeNull}, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpEQ, tt.filterValue)
		if got != tt.expected {
			t.Errorf("EQ(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpNE(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		{"Done", &ast.Value{Type: ast.ValueTypeString, String: "Done"}, false},
		{"Pending", &ast.Value{Type: ast.ValueTypeString, String: "Done"}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 20}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpNE, tt.filterValue)
		if got != tt.expected {
			t.Errorf("NE(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpLT(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		{int64(5), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
		{int64(15), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
		{2.5, &ast.Value{Type: ast.ValueTypeFloat, Float: 3.0}, true},
		{3.0, &ast.Value{Type: ast.ValueTypeFloat, Float: 3.0}, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpLT, tt.filterValue)
		if got != tt.expected {
			t.Errorf("LT(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpLE(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		{int64(5), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(15), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpLE, tt.filterValue)
		if got != tt.expected {
			t.Errorf("LE(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpGT(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		{int64(15), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
		{int64(5), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpGT, tt.filterValue)
		if got != tt.expected {
			t.Errorf("GT(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpGE(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		{int64(15), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(10), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, true},
		{int64(5), &ast.Value{Type: ast.ValueTypeInt, Int: 10}, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpGE, tt.filterValue)
		if got != tt.expected {
			t.Errorf("GE(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpIN(t *testing.T) {
	listValue := &ast.Value{
		Type:    ast.ValueTypeStringList,
		Strings: []string{"New", "In Progress", "Ready"},
	}

	tests := []struct {
		recordValue any
		expected    bool
	}{
		{"New", true},
		{"new", true}, // case-insensitive
		{"In Progress", true},
		{"Done", false},
		{"Closed", false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpIN, listValue)
		if got != tt.expected {
			t.Errorf("IN(%v, %v) = %v, want %v", tt.recordValue, listValue.Strings, got, tt.expected)
		}
	}
}

func TestCompareWithOpNotIn(t *testing.T) {
	listValue := &ast.Value{
		Type:    ast.ValueTypeStringList,
		Strings: []string{"Closed", "Won't Fix"},
	}

	tests := []struct {
		recordValue any
		expected    bool
	}{
		{"New", true},
		{"Closed", false},
		{"Won't Fix", false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpNotIn, listValue)
		if got != tt.expected {
			t.Errorf("NOT IN(%v, %v) = %v, want %v", tt.recordValue, listValue.Strings, got, tt.expected)
		}
	}
}

func TestCompareWithOpContains(t *testing.T) {
	tests := []struct {
		recordValue any
		filterValue *ast.Value
		expected    bool
	}{
		{"Hello World", &ast.Value{Type: ast.ValueTypeString, String: "World"}, true},
		{"Hello World", &ast.Value{Type: ast.ValueTypeString, String: "world"}, true}, // case-insensitive
		{"Hello World", &ast.Value{Type: ast.ValueTypeString, String: "Foo"}, false},
		{"API Documentation", &ast.Value{Type: ast.ValueTypeString, String: "API"}, true},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpContains, tt.filterValue)
		if got != tt.expected {
			t.Errorf("CONTAINS(%v, %v) = %v, want %v", tt.recordValue, tt.filterValue.String, got, tt.expected)
		}
	}
}

func TestCompareWithOpLike(t *testing.T) {
	tests := []struct {
		recordValue any
		pattern     string
		expected    bool
	}{
		// Contains (% on both sides)
		{"Hello World", "%World%", true},
		{"Hello World", "%world%", true}, // case-insensitive
		{"Hello World", "%foo%", false},

		// Starts with (% at end)
		{"Hello World", "Hello%", true},
		{"Hello World", "World%", false},

		// Ends with (% at start)
		{"Hello World", "%World", true},
		{"Hello World", "%Hello", false},

		// Exact match (no %)
		{"Hello", "Hello", true},
		{"Hello", "hello", true}, // case-insensitive
		{"Hello", "World", false},
	}

	for _, tt := range tests {
		filterValue := &ast.Value{Type: ast.ValueTypeString, String: tt.pattern}
		got := compareWithOp(tt.recordValue, ast.OpLike, filterValue)
		if got != tt.expected {
			t.Errorf("LIKE(%q, %q) = %v, want %v", tt.recordValue, tt.pattern, got, tt.expected)
		}
	}
}

func TestCompareWithOpIsNull(t *testing.T) {
	tests := []struct {
		recordValue any
		expected    bool
	}{
		{nil, true},
		{"", false},
		{"value", false},
		{0, false},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpIsNull, nil)
		if got != tt.expected {
			t.Errorf("IS NULL(%v) = %v, want %v", tt.recordValue, got, tt.expected)
		}
	}
}

func TestCompareWithOpIsNotNull(t *testing.T) {
	tests := []struct {
		recordValue any
		expected    bool
	}{
		{nil, false},
		{"", true},
		{"value", true},
		{0, true},
	}

	for _, tt := range tests {
		got := compareWithOp(tt.recordValue, ast.OpIsNotNull, nil)
		if got != tt.expected {
			t.Errorf("IS NOT NULL(%v) = %v, want %v", tt.recordValue, got, tt.expected)
		}
	}
}

func TestCompareValuesTime(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)
	later := now.Add(time.Hour)

	if compareValues(earlier, now) >= 0 {
		t.Error("earlier should be < now")
	}
	if compareValues(later, now) <= 0 {
		t.Error("later should be > now")
	}
	if compareValues(now, now) != 0 {
		t.Error("same time should be equal")
	}
}

func TestCompareValuesNil(t *testing.T) {
	if compareValues(nil, nil) != 0 {
		t.Error("nil == nil should return 0")
	}
	if compareValues(nil, "value") >= 0 {
		t.Error("nil should be < any value")
	}
	if compareValues("value", nil) <= 0 {
		t.Error("any value should be > nil")
	}
}

func TestMatchFilter(t *testing.T) {
	record := result.Record{
		"name":   "Test Feature",
		"status": "Done",
		"votes":  int64(15),
	}

	tests := []struct {
		filter   planner.Filter
		expected bool
	}{
		// Match by field
		{
			filter: planner.Filter{
				Field:         "status",
				Op:            ast.OpEQ,
				Value:         &ast.Value{Type: ast.ValueTypeString, String: "Done"},
				SubqueryIndex: -1, // no subquery
			},
			expected: true,
		},
		// No match
		{
			filter: planner.Filter{
				Field:         "status",
				Op:            ast.OpEQ,
				Value:         &ast.Value{Type: ast.ValueTypeString, String: "Pending"},
				SubqueryIndex: -1, // no subquery
			},
			expected: false,
		},
		// Numeric comparison
		{
			filter: planner.Filter{
				Field:         "votes",
				Op:            ast.OpGT,
				Value:         &ast.Value{Type: ast.ValueTypeInt, Int: 10},
				SubqueryIndex: -1, // no subquery
			},
			expected: true,
		},
		// Missing field (returns false for non-null checks)
		{
			filter: planner.Filter{
				Field:         "missing",
				Op:            ast.OpEQ,
				Value:         &ast.Value{Type: ast.ValueTypeString, String: "value"},
				SubqueryIndex: -1, // no subquery
			},
			expected: false,
		},
		// Empty field (complex expression, returns true)
		{
			filter: planner.Filter{
				Field:         "",
				Op:            "",
				Value:         nil,
				SubqueryIndex: -1, // no subquery
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		got := matchFilter(record, tt.filter)
		if got != tt.expected {
			t.Errorf("matchFilter with %+v = %v, want %v", tt.filter, got, tt.expected)
		}
	}
}

func TestMatchFilterWithSubquery(t *testing.T) {
	record := result.Record{
		"votes": int64(50),
		"id":    "FEAT-1",
	}

	tests := []struct {
		name     string
		filter   planner.Filter
		expected bool
	}{
		{
			name: "scalar subquery > (value is 30)",
			filter: planner.Filter{
				Field:          "votes",
				Op:             ast.OpGT,
				SubqueryIndex:  0,
				SubqueryResult: int64(30), // subquery returned 30
			},
			expected: true, // 50 > 30
		},
		{
			name: "scalar subquery > (value is 60)",
			filter: planner.Filter{
				Field:          "votes",
				Op:             ast.OpGT,
				SubqueryIndex:  0,
				SubqueryResult: int64(60), // subquery returned 60
			},
			expected: false, // 50 > 60 is false
		},
		{
			name: "scalar subquery = (equal)",
			filter: planner.Filter{
				Field:          "votes",
				Op:             ast.OpEQ,
				SubqueryIndex:  0,
				SubqueryResult: int64(50),
			},
			expected: true, // 50 == 50
		},
		{
			name: "IN subquery (value in list)",
			filter: planner.Filter{
				Field:          "id",
				Op:             ast.OpIN,
				SubqueryIndex:  0,
				SubqueryResult: []any{"FEAT-1", "FEAT-2", "FEAT-3"},
			},
			expected: true, // FEAT-1 is in list
		},
		{
			name: "IN subquery (value not in list)",
			filter: planner.Filter{
				Field:          "id",
				Op:             ast.OpIN,
				SubqueryIndex:  0,
				SubqueryResult: []any{"FEAT-2", "FEAT-3"},
			},
			expected: false, // FEAT-1 is not in list
		},
		{
			name: "NOT IN subquery (value not in list)",
			filter: planner.Filter{
				Field:          "id",
				Op:             ast.OpNotIn,
				SubqueryIndex:  0,
				SubqueryResult: []any{"FEAT-2", "FEAT-3"},
			},
			expected: true, // FEAT-1 is not in list, so NOT IN is true
		},
		{
			name: "NOT IN subquery (value in list)",
			filter: planner.Filter{
				Field:          "id",
				Op:             ast.OpNotIn,
				SubqueryIndex:  0,
				SubqueryResult: []any{"FEAT-1", "FEAT-2"},
			},
			expected: false, // FEAT-1 is in list, so NOT IN is false
		},
		{
			name: "IN subquery with empty list",
			filter: planner.Filter{
				Field:          "id",
				Op:             ast.OpIN,
				SubqueryIndex:  0,
				SubqueryResult: []any{},
			},
			expected: false, // empty list means nothing matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchFilter(record, tt.filter)
			if got != tt.expected {
				t.Errorf("matchFilter = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompareWithSubqueryResult(t *testing.T) {
	tests := []struct {
		name           string
		recordValue    any
		op             ast.CompareOp
		subqueryResult any
		expected       bool
	}{
		// Scalar comparisons
		{"int > int (true)", int64(10), ast.OpGT, int64(5), true},
		{"int > int (false)", int64(5), ast.OpGT, int64(10), false},
		{"int >= int (equal)", int64(10), ast.OpGE, int64(10), true},
		{"int < int", int64(5), ast.OpLT, int64(10), true},
		{"int <= int", int64(10), ast.OpLE, int64(10), true},
		{"int = int", int64(10), ast.OpEQ, int64(10), true},
		{"int != int", int64(10), ast.OpNE, int64(5), true},

		// Float comparisons
		{"float > float", 10.5, ast.OpGT, 5.5, true},
		{"float = float", 10.5, ast.OpEQ, 10.5, true},

		// String comparisons
		{"string = string", "test", ast.OpEQ, "test", true},
		{"string != string", "test", ast.OpNE, "other", true},

		// IN with list
		{"IN list (found)", "a", ast.OpIN, []any{"a", "b", "c"}, true},
		{"IN list (not found)", "d", ast.OpIN, []any{"a", "b", "c"}, false},
		{"NOT IN list (not found)", "d", ast.OpNotIn, []any{"a", "b", "c"}, true},
		{"NOT IN list (found)", "a", ast.OpNotIn, []any{"a", "b", "c"}, false},

		// Edge cases
		{"IN with nil list", "a", ast.OpIN, nil, false},
		{"NOT IN with nil list", "a", ast.OpNotIn, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareWithSubqueryResult(tt.recordValue, tt.op, tt.subqueryResult)
			if got != tt.expected {
				t.Errorf("compareWithSubqueryResult(%v, %s, %v) = %v, want %v",
					tt.recordValue, tt.op, tt.subqueryResult, got, tt.expected)
			}
		})
	}
}

func TestValueInAnyList(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		list     []any
		expected bool
	}{
		{"string in list", "a", []any{"a", "b", "c"}, true},
		{"string not in list", "d", []any{"a", "b", "c"}, false},
		{"int in list", int64(1), []any{int64(1), int64(2), int64(3)}, true},
		{"int not in list", int64(4), []any{int64(1), int64(2), int64(3)}, false},
		{"empty list", "a", []any{}, false},
		{"mixed types", int64(1), []any{"1", int64(1), 1.0}, true}, // int64(1) matches int64(1)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueInAnyList(tt.value, tt.list)
			if got != tt.expected {
				t.Errorf("valueInAnyList(%v, %v) = %v, want %v",
					tt.value, tt.list, got, tt.expected)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		input    any
		expected int64
	}{
		{int(10), 10},
		{int64(20), 20},
		{int32(30), 30},
		{float64(40.9), 40},
		{float32(50.1), 50},
		{"not a number", 0},
	}

	for _, tt := range tests {
		got := toInt64(tt.input)
		if got != tt.expected {
			t.Errorf("toInt64(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
	}{
		{float64(3.14), 3.14},
		{float32(2.5), 2.5},
		{int(10), 10.0},
		{int64(20), 20.0},
		{int32(30), 30.0},
		{"not a number", 0},
	}

	for _, tt := range tests {
		got := toFloat64(tt.input)
		if got != tt.expected {
			t.Errorf("toFloat64(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestFilterValueToAny(t *testing.T) {
	now := time.Now()
	dur := time.Hour

	tests := []struct {
		input    *ast.Value
		expected any
	}{
		{nil, nil},
		{&ast.Value{Type: ast.ValueTypeString, String: "test"}, "test"},
		{&ast.Value{Type: ast.ValueTypeInt, Int: 42}, int64(42)},
		{&ast.Value{Type: ast.ValueTypeFloat, Float: 3.14}, 3.14},
		{&ast.Value{Type: ast.ValueTypeBool, Bool: true}, true},
		{&ast.Value{Type: ast.ValueTypeTime, Time: now}, now},
		{&ast.Value{Type: ast.ValueTypeDuration, Duration: dur}, dur},
		{&ast.Value{Type: ast.ValueTypeNull}, nil},
	}

	for _, tt := range tests {
		got := filterValueToAny(tt.input)
		if got != tt.expected {
			t.Errorf("filterValueToAny(%+v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
