package executor

import (
	"fmt"
	"strings"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// applyAggregations applies GROUP BY and aggregate functions to records.
func (e *Executor) applyAggregations(records []result.Record, plan *planner.Plan) []result.Record {
	// If no aggregations, return original records
	if !plan.HasAggregates {
		return records
	}

	// Group records
	groups := e.groupRecords(records, plan.GroupBy)

	// Apply aggregations to each group
	var aggregated []result.Record
	for key, groupRecords := range groups {
		aggRecord := e.computeAggregates(groupRecords, plan)

		// Add group key fields to record
		if len(plan.GroupBy) > 0 && len(groupRecords) > 0 {
			for _, field := range plan.GroupBy {
				aggRecord[field] = groupRecords[0].Get(field)
			}
		}

		// Store group key for debugging
		if key != "" {
			aggRecord["_group_key"] = key
		}

		aggregated = append(aggregated, aggRecord)
	}

	return aggregated
}

// groupRecords groups records by the specified fields.
func (e *Executor) groupRecords(records []result.Record, groupBy []string) map[string][]result.Record {
	groups := make(map[string][]result.Record)

	if len(groupBy) == 0 {
		// No GROUP BY - all records in one group
		groups[""] = records
		return groups
	}

	for _, r := range records {
		key := e.makeGroupKey(r, groupBy)
		groups[key] = append(groups[key], r)
	}

	return groups
}

// makeGroupKey creates a grouping key from record values.
func (e *Executor) makeGroupKey(r result.Record, groupBy []string) string {
	var parts []string
	for _, field := range groupBy {
		val := r.Get(field)
		parts = append(parts, fmt.Sprintf("%v", val))
	}
	return strings.Join(parts, "\x00")
}

// computeAggregates computes aggregate values for a group of records.
func (e *Executor) computeAggregates(records []result.Record, plan *planner.Plan) result.Record {
	aggRecord := make(result.Record)

	for _, agg := range plan.Aggregations {
		var value any
		switch agg.Func {
		case ast.AggCount:
			value = e.computeCount(records, agg)
		case ast.AggSum:
			value = e.computeSum(records, agg)
		case ast.AggAvg:
			value = e.computeAvg(records, agg)
		case ast.AggMin:
			value = e.computeMin(records, agg)
		case ast.AggMax:
			value = e.computeMax(records, agg)
		}
		aggRecord[agg.Alias] = value
	}

	return aggRecord
}

// computeCount computes COUNT aggregate.
func (e *Executor) computeCount(records []result.Record, agg planner.Aggregation) int {
	if agg.Field == "" {
		// COUNT(*)
		return len(records)
	}

	if agg.Distinct {
		seen := make(map[any]bool)
		for _, r := range records {
			val := r.Get(agg.Field)
			if val != nil {
				seen[val] = true
			}
		}
		return len(seen)
	}

	// COUNT(field) - count non-null values
	count := 0
	for _, r := range records {
		if r.Get(agg.Field) != nil {
			count++
		}
	}
	return count
}

// computeSum computes SUM aggregate.
func (e *Executor) computeSum(records []result.Record, agg planner.Aggregation) float64 {
	var sum float64
	for _, r := range records {
		sum += toFloat64(r.Get(agg.Field))
	}
	return sum
}

// computeAvg computes AVG aggregate.
func (e *Executor) computeAvg(records []result.Record, agg planner.Aggregation) float64 {
	if len(records) == 0 {
		return 0
	}

	var sum float64
	var count int
	for _, r := range records {
		val := r.Get(agg.Field)
		if val != nil {
			sum += toFloat64(val)
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// computeMin computes MIN aggregate.
func (e *Executor) computeMin(records []result.Record, agg planner.Aggregation) any {
	if len(records) == 0 {
		return nil
	}

	var minVal any
	for _, r := range records {
		val := r.Get(agg.Field)
		if val == nil {
			continue
		}
		if minVal == nil || compareValues(val, minVal) < 0 {
			minVal = val
		}
	}
	return minVal
}

// computeMax computes MAX aggregate.
func (e *Executor) computeMax(records []result.Record, agg planner.Aggregation) any {
	if len(records) == 0 {
		return nil
	}

	var maxVal any
	for _, r := range records {
		val := r.Get(agg.Field)
		if val == nil {
			continue
		}
		if maxVal == nil || compareValues(val, maxVal) > 0 {
			maxVal = val
		}
	}
	return maxVal
}

// applyHaving applies the HAVING clause filter to aggregated records.
func (e *Executor) applyHaving(records []result.Record, having ast.Expr) []result.Record {
	if having == nil {
		return records
	}

	var filtered []result.Record
	for _, r := range records {
		if e.evalExpr(r, having) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// evalExpr evaluates an expression against a record.
func (e *Executor) evalExpr(r result.Record, expr ast.Expr) bool {
	switch ex := expr.(type) {
	case *ast.BinaryExpr:
		left := e.evalExpr(r, ex.Left)
		right := e.evalExpr(r, ex.Right)
		if ex.Op == ast.OpAnd {
			return left && right
		}
		return left || right

	case *ast.NotExpr:
		return !e.evalExpr(r, ex.Expr)

	case *ast.ParenExpr:
		return e.evalExpr(r, ex.Expr)

	case *ast.CompareExpr:
		val := r.Get(ex.Field.String())
		return evalCompare(val, ex.Op, ex.Value)
	}

	return true
}

// evalCompare evaluates a comparison.
func evalCompare(val any, op ast.CompareOp, expected *ast.Value) bool {
	if expected == nil {
		switch op {
		case ast.OpIsNull:
			return val == nil
		case ast.OpIsNotNull:
			return val != nil
		}
		return false
	}

	// Get expected value for comparison
	var expectedVal any
	switch expected.Type {
	case ast.ValueTypeString:
		expectedVal = expected.String
	case ast.ValueTypeInt:
		expectedVal = expected.Int
	case ast.ValueTypeFloat:
		expectedVal = expected.Float
	case ast.ValueTypeBool:
		expectedVal = expected.Bool
	default:
		expectedVal = expected.Raw
	}

	cmp := compareValues(val, expectedVal)

	switch op {
	case ast.OpEQ:
		return cmp == 0
	case ast.OpNE:
		return cmp != 0
	case ast.OpLT:
		return cmp < 0
	case ast.OpLE:
		return cmp <= 0
	case ast.OpGT:
		return cmp > 0
	case ast.OpGE:
		return cmp >= 0
	}

	return false
}
