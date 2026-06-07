package executor

import (
	"context"
	"fmt"

	aha "github.com/grokify/aha-go"
	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// MutationOptions configures mutation execution.
type MutationOptions struct {
	// DryRun if true, shows what would happen without making changes.
	DryRun bool

	// ProductID is required for most mutations.
	ProductID string

	// ReleaseID is required for creating features.
	ReleaseID string
}

// MutationResult represents the result of a mutation operation.
type MutationResult struct {
	// Operation is the type of mutation (INSERT, UPDATE, DELETE).
	Operation string

	// Entity is the entity type affected.
	Entity ast.EntityType

	// AffectedCount is the number of records affected.
	AffectedCount int

	// Records contains the affected records (for INSERT/UPDATE).
	Records []result.Record

	// IDs contains the IDs of affected records.
	IDs []string

	// DryRun indicates if this was a dry run.
	DryRun bool

	// Errors contains any errors encountered during the operation.
	Errors []error
}

// ExecuteInsert executes an INSERT statement.
func (e *Executor) ExecuteInsert(ctx context.Context, stmt *ast.InsertStatement, opts MutationOptions) (*MutationResult, error) {
	res := &MutationResult{
		Operation: "INSERT",
		Entity:    stmt.Entity,
		DryRun:    opts.DryRun,
	}

	// Build the record from columns and values
	record := make(map[string]any)
	for i, col := range stmt.Columns {
		record[col] = valueToAny(&stmt.Values[i])
	}

	if opts.DryRun {
		res.Records = []result.Record{record}
		res.AffectedCount = 1
		return res, nil
	}

	// Execute the insert based on entity type
	switch stmt.Entity {
	case ast.EntityFeatures:
		return e.insertFeature(ctx, record, opts)
	case ast.EntityIdeas:
		return nil, fmt.Errorf("INSERT not yet supported for ideas (API not implemented in aha-go)")
	case ast.EntityReleases:
		return nil, fmt.Errorf("INSERT not supported for releases")
	case ast.EntityInitiatives:
		return nil, fmt.Errorf("INSERT not yet supported for initiatives")
	default:
		return nil, fmt.Errorf("INSERT not supported for entity: %s", stmt.Entity)
	}
}

// ExecuteUpdate executes an UPDATE statement.
func (e *Executor) ExecuteUpdate(ctx context.Context, stmt *ast.UpdateStatement, opts MutationOptions) (*MutationResult, error) {
	res := &MutationResult{
		Operation: "UPDATE",
		Entity:    stmt.Entity,
		DryRun:    opts.DryRun,
	}

	// First, find records matching the WHERE clause
	records, err := e.findRecordsForMutation(ctx, stmt.Entity, stmt.Where, opts)
	if err != nil {
		return nil, fmt.Errorf("finding records to update: %w", err)
	}

	if opts.DryRun {
		res.AffectedCount = len(records)
		res.Records = records
		return res, nil
	}

	// Build the update map
	updates := make(map[string]any)
	for _, assign := range stmt.Assignments {
		updates[assign.Field] = valueToAny(&assign.Value)
	}

	// Execute updates
	for _, rec := range records {
		id, ok := rec["id"].(string)
		if !ok {
			res.Errors = append(res.Errors, fmt.Errorf("record missing id field"))
			continue
		}

		err := e.updateRecord(ctx, stmt.Entity, id, updates)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("updating %s: %w", id, err))
			continue
		}

		res.IDs = append(res.IDs, id)
		res.AffectedCount++
	}

	return res, nil
}

// ExecuteDelete executes a DELETE statement.
func (e *Executor) ExecuteDelete(ctx context.Context, stmt *ast.DeleteStatement, opts MutationOptions) (*MutationResult, error) {
	res := &MutationResult{
		Operation: "DELETE",
		Entity:    stmt.Entity,
		DryRun:    opts.DryRun,
	}

	// DELETE is not supported by the Aha API for most entities
	// We'll return an error for now
	return res, fmt.Errorf("DELETE not supported by Aha API for entity: %s", stmt.Entity)
}

// findRecordsForMutation finds records matching a WHERE clause.
func (e *Executor) findRecordsForMutation(ctx context.Context, entity ast.EntityType, where *ast.WhereClause, opts MutationOptions) ([]result.Record, error) {
	// Build a query plan
	query := &ast.Query{
		From: &ast.FromClause{Entity: entity},
	}
	if where != nil {
		query.Where = where
	}

	plan := planner.New().Plan(query)
	plan.APIParams.ProductID = opts.ProductID
	plan.RequiresPagination = true // Fetch all matching records

	res, err := e.Execute(ctx, plan)
	if err != nil {
		return nil, err
	}

	return res.Records, nil
}

// insertFeature inserts a new feature.
func (e *Executor) insertFeature(ctx context.Context, record map[string]any, opts MutationOptions) (*MutationResult, error) {
	if opts.ReleaseID == "" {
		return nil, fmt.Errorf("release ID required for creating features (use --release flag)")
	}

	// Build functional options from record
	var createOpts []aha.CreateFeatureOption

	if name := getString(record, "name"); name != "" {
		createOpts = append(createOpts, aha.WithFeatureName(name))
	} else {
		return nil, fmt.Errorf("name is required for creating features")
	}

	if desc := getString(record, "description"); desc != "" {
		createOpts = append(createOpts, aha.WithFeatureDescription(desc))
	}
	if status := getString(record, "workflow_status"); status != "" {
		createOpts = append(createOpts, aha.WithFeatureStatus(status))
	}
	if status := getString(record, "status"); status != "" {
		createOpts = append(createOpts, aha.WithFeatureStatus(status))
	}
	if assignee := getString(record, "assigned_to"); assignee != "" {
		createOpts = append(createOpts, aha.WithFeatureAssignedTo(assignee))
	}
	if tags := getString(record, "tags"); tags != "" {
		createOpts = append(createOpts, aha.WithFeatureTags(tags))
	}

	feature, err := e.client.CreateFeature(ctx, opts.ReleaseID, createOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating feature: %w", err)
	}

	return &MutationResult{
		Operation:     "INSERT",
		Entity:        ast.EntityFeatures,
		AffectedCount: 1,
		IDs:           []string{feature.ID},
		Records: []result.Record{{
			"id":            feature.ID,
			"reference_num": feature.ReferenceNum,
			"name":          feature.Name,
		}},
	}, nil
}

// updateRecord updates a single record.
func (e *Executor) updateRecord(ctx context.Context, entity ast.EntityType, id string, updates map[string]any) error {
	switch entity {
	case ast.EntityFeatures:
		return e.updateFeature(ctx, id, updates)
	case ast.EntityIdeas:
		return fmt.Errorf("UPDATE not yet supported for ideas (API not implemented in aha-go)")
	default:
		return fmt.Errorf("UPDATE not supported for entity: %s", entity)
	}
}

// updateFeature updates a feature using functional options.
func (e *Executor) updateFeature(ctx context.Context, id string, updates map[string]any) error {
	var updateOpts []aha.UpdateFeatureOption

	// Build functional options from updates map
	if name := getString(updates, "name"); name != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.Name = name })
	}
	if desc := getString(updates, "description"); desc != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.Description = desc })
	}
	if status := getString(updates, "workflow_status"); status != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.WorkflowStatus = status })
	}
	if status := getString(updates, "status"); status != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.WorkflowStatus = status })
	}
	if assignee := getString(updates, "assigned_to"); assignee != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.AssignedToUser = assignee })
	}
	if tags := getString(updates, "tags"); tags != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.Tags = tags })
	}
	if release := getString(updates, "release"); release != "" {
		updateOpts = append(updateOpts, func(o *aha.UpdateFeatureOptions) { o.Release = release })
	}

	if len(updateOpts) == 0 {
		return nil // Nothing to update
	}

	_, err := e.client.UpdateFeature(ctx, id, updateOpts...)
	return err
}

// valueToAny converts an AST Value to a Go any.
func valueToAny(v *ast.Value) any {
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
	case ast.ValueTypeNull:
		return nil
	default:
		return v.Raw
	}
}

// getString extracts a string value from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
