package executor

import (
	"context"
	"fmt"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// executeWithJoins executes a query plan that includes JOIN clauses.
func (e *Executor) executeWithJoins(ctx context.Context, plan *planner.Plan, baseRecords []result.Record) ([]result.Record, error) {
	if len(plan.Joins) == 0 {
		return baseRecords, nil
	}

	currentRecords := baseRecords

	for _, join := range plan.Joins {
		joinedRecords, err := e.executeJoin(ctx, join, currentRecords, plan)
		if err != nil {
			return nil, err
		}
		currentRecords = joinedRecords
	}

	return currentRecords, nil
}

// executeJoin executes a single JOIN operation.
func (e *Executor) executeJoin(ctx context.Context, join *planner.JoinPlan, leftRecords []result.Record, plan *planner.Plan) ([]result.Record, error) {
	// Fetch records from the joined entity
	rightRecords, err := e.fetchEntityRecords(ctx, join.Entity, plan)
	if err != nil {
		return nil, fmt.Errorf("fetching joined entity %s: %w", join.Entity, err)
	}

	// Perform the join
	var joinedRecords []result.Record

	switch join.Type {
	case ast.JoinInner:
		joinedRecords = e.innerJoin(leftRecords, rightRecords, join)
	case ast.JoinLeft:
		joinedRecords = e.leftJoin(leftRecords, rightRecords, join)
	case ast.JoinRight:
		joinedRecords = e.rightJoin(leftRecords, rightRecords, join)
	default:
		joinedRecords = e.innerJoin(leftRecords, rightRecords, join)
	}

	return joinedRecords, nil
}

// fetchEntityRecords fetches all records for an entity type.
func (e *Executor) fetchEntityRecords(ctx context.Context, entity ast.EntityType, plan *planner.Plan) ([]result.Record, error) {
	// Create a simple plan for fetching the entity
	fetchPlan := &planner.Plan{
		Entity:             entity,
		APIParams:          planner.APIParams{ProductID: plan.APIParams.ProductID},
		RequiresPagination: true, // Always fetch all for joins
	}

	switch entity {
	case ast.EntityFeatures:
		return e.executeFeatures(ctx, fetchPlan)
	case ast.EntityIdeas:
		return e.executeIdeas(ctx, fetchPlan)
	case ast.EntityReleases:
		return e.executeReleases(ctx, fetchPlan)
	case ast.EntityInitiatives:
		return e.executeInitiatives(ctx, fetchPlan)
	default:
		return nil, fmt.Errorf("unsupported entity type for join: %s", entity)
	}
}

// innerJoin performs an inner join between left and right records.
func (e *Executor) innerJoin(left, right []result.Record, join *planner.JoinPlan) []result.Record {
	var joined []result.Record

	for _, l := range left {
		for _, r := range right {
			if e.matchesJoinCondition(l, r, join) {
				merged := e.mergeRecords(l, r, join)
				joined = append(joined, merged)
			}
		}
	}

	return joined
}

// leftJoin performs a left outer join.
func (e *Executor) leftJoin(left, right []result.Record, join *planner.JoinPlan) []result.Record {
	var joined []result.Record

	for _, l := range left {
		matched := false
		for _, r := range right {
			if e.matchesJoinCondition(l, r, join) {
				merged := e.mergeRecords(l, r, join)
				joined = append(joined, merged)
				matched = true
			}
		}
		if !matched {
			// Include left record with null right side
			merged := e.mergeRecords(l, nil, join)
			joined = append(joined, merged)
		}
	}

	return joined
}

// rightJoin performs a right outer join.
func (e *Executor) rightJoin(left, right []result.Record, join *planner.JoinPlan) []result.Record {
	var joined []result.Record

	for _, r := range right {
		matched := false
		for _, l := range left {
			if e.matchesJoinCondition(l, r, join) {
				merged := e.mergeRecords(l, r, join)
				joined = append(joined, merged)
				matched = true
			}
		}
		if !matched {
			// Include right record with null left side
			merged := e.mergeRecords(nil, r, join)
			joined = append(joined, merged)
		}
	}

	return joined
}

// matchesJoinCondition checks if two records match the join condition.
func (e *Executor) matchesJoinCondition(left, right result.Record, join *planner.JoinPlan) bool {
	if join.Condition == nil {
		// No condition = cross join (match everything)
		return true
	}

	// Create a merged record for condition evaluation
	merged := e.mergeRecordsForEval(left, right, join)

	return e.evalExpr(merged, join.Condition)
}

// mergeRecords merges two records, prefixing fields from the right side.
func (e *Executor) mergeRecords(left, right result.Record, join *planner.JoinPlan) result.Record {
	merged := make(result.Record)

	// Add left fields
	for k, v := range left {
		merged[k] = v
	}

	// Add right fields with prefix
	prefix := join.Alias
	if prefix == "" {
		prefix = string(join.Entity)
	}

	for k, v := range right {
		// Use qualified name for right side fields
		merged[prefix+"."+k] = v
	}

	return merged
}

// mergeRecordsForEval creates a merged record for evaluating join conditions.
func (e *Executor) mergeRecordsForEval(left, right result.Record, join *planner.JoinPlan) result.Record {
	merged := make(result.Record)

	// Add left fields (can be accessed directly or with FROM entity prefix)
	for k, v := range left {
		merged[k] = v
	}

	// Add right fields with entity/alias prefix
	prefix := join.Alias
	if prefix == "" {
		prefix = string(join.Entity)
	}

	for k, v := range right {
		merged[prefix+"."+k] = v
		// Also add without prefix for simple conditions
		if _, exists := merged[k]; !exists {
			merged[k] = v
		}
	}

	return merged
}
