package executor

import (
	"context"
	"fmt"
	"sort"
	"strings"

	aha "github.com/grokify/aha-go"
	ahagraphql "github.com/grokify/aha-go/graphql"
	"github.com/grokify/aha-go/graphql/generated"
	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
	"github.com/grokify/aha-studio/schema"
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

	standardUpdates, customUpdates := splitAssignments(stmt.Assignments)

	// Execute updates
	for _, rec := range records {
		id, ok := rec["id"].(string)
		if !ok {
			res.Errors = append(res.Errors, fmt.Errorf("record missing id field"))
			continue
		}

		var errs []error
		if len(standardUpdates) > 0 {
			if err := e.updateRecord(ctx, stmt.Entity, id, standardUpdates); err != nil {
				errs = append(errs, fmt.Errorf("updating standard fields: %w", err))
			}
		}
		if len(customUpdates) > 0 {
			if err := e.updateCustomFields(ctx, stmt.Entity, id, customUpdates); err != nil {
				errs = append(errs, fmt.Errorf("updating custom fields: %w", err))
			}
		}

		// A partial success (standard field written, custom field
		// rejected, or vice versa) is a record-level failure, not a
		// silent partial win.
		if len(errs) > 0 {
			res.Errors = append(res.Errors, fmt.Errorf("updating %s: %w", id, joinErrors(errs)))
			continue
		}

		res.IDs = append(res.IDs, id)
		res.AffectedCount++
	}

	return res, nil
}

// splitAssignments partitions UPDATE SET-clause assignments into standard
// fields (routed per-entity through REST, today only implemented for
// features) and custom.* fields (routed through the GraphQL
// SetCustomFieldValues mutation, entity-agnostic - works for features,
// initiatives, and releases). Custom field keys are stored bare (prefix
// stripped), matching what SetCustomFieldValues expects.
func splitAssignments(assignments []ast.Assignment) (standard, custom map[string]any) {
	standard = make(map[string]any)
	custom = make(map[string]any)
	for _, assign := range assignments {
		if schema.IsCustomFieldName(assign.Field) {
			custom[schema.CustomFieldName(assign.Field)] = valueToAny(&assign.Value)
		} else {
			standard[assign.Field] = valueToAny(&assign.Value)
		}
	}
	return standard, custom
}

// joinErrors combines multiple errors into one, since standard-field and
// custom-field updates for the same record can each fail independently.
func joinErrors(errs []error) error {
	if len(errs) == 1 {
		return errs[0]
	}
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
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

// updateRecord updates a single record's standard (non-custom.*) fields.
func (e *Executor) updateRecord(ctx context.Context, entity ast.EntityType, id string, updates map[string]any) error {
	switch entity {
	case ast.EntityFeatures:
		return e.updateFeature(ctx, id, updates)
	case ast.EntityIdeas:
		return fmt.Errorf("UPDATE not yet supported for ideas (API not implemented in aha-go)")
	case ast.EntityInitiatives, ast.EntityReleases:
		return fmt.Errorf("standard-field UPDATE not supported for entity: %s; only custom.* fields are supported", entity)
	default:
		return fmt.Errorf("UPDATE not supported for entity: %s", entity)
	}
}

// featureUpdateFields are the standard (non-custom.*) fields updateFeature
// recognizes. Any key in `updates` outside this set is an error - see the
// unrecognizedKeys check below - rather than being silently dropped.
var featureUpdateFields = map[string]bool{
	"name": true, "description": true, "workflow_status": true, "status": true,
	"assigned_to": true, "tags": true, "release": true,
}

// updateFeature updates a feature using functional options. Fails fast
// (before calling the API) if `updates` contains any key updateFeature
// doesn't recognize, rather than silently applying only the recognized
// subset - a previously-existing bug (recognized keys were applied, an
// unrecognized key silently no-op'd) is fixed generally here, not just
// for custom.* fields.
func (e *Executor) updateFeature(ctx context.Context, id string, updates map[string]any) error {
	if unrecognized := unrecognizedKeys(updates, featureUpdateFields); len(unrecognized) > 0 {
		return fmt.Errorf("unsupported field(s) for feature UPDATE: %s", strings.Join(unrecognized, ", "))
	}

	var updateOpts []aha.UpdateFeatureOption

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

// unrecognizedKeys returns the keys in m that aren't in known, sorted for
// deterministic error messages.
func unrecognizedKeys(m map[string]any, known map[string]bool) []string {
	var out []string
	for k := range m {
		if !known[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// customFieldableType maps an AQL entity type to the GraphQL
// CustomFieldableTypeEnum value SetCustomFieldValues expects. Scoped to
// the entities this feature was built for (Feature, Initiative, Release)
// even though the underlying enum has more values (Epic, Goal, Idea,
// etc.) - support for those isn't tested or requested yet.
func customFieldableType(entity ast.EntityType) (generated.CustomFieldableTypeEnum, error) {
	switch entity {
	case ast.EntityFeatures:
		return generated.CustomFieldableTypeEnumFeature, nil
	case ast.EntityInitiatives:
		return generated.CustomFieldableTypeEnumInitiative, nil
	case ast.EntityReleases:
		return generated.CustomFieldableTypeEnumRelease, nil
	default:
		return "", fmt.Errorf("custom field UPDATE not supported for entity: %s", entity)
	}
}

// updateCustomFields sets custom.* field values on a record via the
// GraphQL SetCustomFieldValues mutation (no REST equivalent exists).
func (e *Executor) updateCustomFields(ctx context.Context, entity ast.EntityType, id string, values map[string]any) error {
	if e.graphqlClient == nil {
		return fmt.Errorf("custom field UPDATE requires a GraphQL client (call Executor.WithGraphQL)")
	}
	recordType, err := customFieldableType(entity)
	if err != nil {
		return err
	}
	_, err = ahagraphql.SetCustomFieldValues(ctx, e.graphqlClient, id, recordType, values)
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
