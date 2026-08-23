package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/ent/syncmeta"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
	"github.com/grokify/aha-studio/schema"
)

// QueryOffline executes a query against the local SQLite database.
func (d *DB) QueryOffline(ctx context.Context, plan *planner.Plan) (*result.Result, error) {
	// Build SQL query from plan
	sqlQuery, args, err := d.buildSQL(plan)
	if err != nil {
		return nil, fmt.Errorf("building SQL: %w", err)
	}

	// Execute query
	rows, err := d.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Collect results, flattening the data JSON column into each record
	// (shared with ListRecords/GetRecord).
	flattened, err := rowsToFlattenedRecords(rows)
	if err != nil {
		return nil, err
	}
	records := make([]result.Record, len(flattened))
	for i, rec := range flattened {
		records[i] = result.Record(rec)
	}

	return &result.Result{
		Entity:  plan.Entity,
		Records: records,
	}, nil
}

// buildSQL converts a planner.Plan to a SQL query.
//
// Field names in SELECT/WHERE/ORDER BY come from user-supplied AQL text.
// Every one is validated against schema.GetEntity(plan.Entity)'s allow-list
// (the same source of truth the AQL parser/validator already uses) before
// being interpolated into SQL - custom.* fields are handled separately via
// filterToSQL's json_each predicate rather than raw column interpolation.
func (d *DB) buildSQL(plan *planner.Plan) (string, []interface{}, error) {
	var args []interface{}

	// Determine table name
	table := string(plan.Entity)

	// Build SELECT clause
	selectClause := "*"
	if len(plan.SelectFields) > 0 {
		for _, f := range plan.SelectFields {
			if !isFieldAllowed(plan.Entity, f) {
				return "", nil, fmt.Errorf("field not allowed in SELECT for %s: %s", plan.Entity, f)
			}
		}
		selectClause = strings.Join(plan.SelectFields, ", ")
	}

	// Build WHERE clause
	whereClause := ""
	if len(plan.ClientFilters) > 0 {
		var conditions []string
		for _, f := range plan.ClientFilters {
			cond, arg, err := d.filterToSQL(plan.Entity, f)
			if err != nil {
				return "", nil, err
			}
			conditions = append(conditions, cond)
			if arg != nil {
				args = append(args, arg...)
			}
		}
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add product filter if set
	if d.product != "" && table != "users" && table != "products" {
		if whereClause == "" {
			whereClause = " WHERE product = ?"
		} else {
			whereClause += " AND product = ?"
		}
		args = append(args, d.product)
	}

	// Build ORDER BY clause
	orderClause := ""
	if plan.OrderBy != nil {
		if !isFieldAllowed(plan.Entity, plan.OrderBy.Field) {
			return "", nil, fmt.Errorf("field not allowed in ORDER BY for %s: %s", plan.Entity, plan.OrderBy.Field)
		}
		dir := "ASC"
		if plan.OrderBy.Direction == ast.SortDesc {
			dir = "DESC"
		}
		orderClause = fmt.Sprintf(" ORDER BY %s %s", plan.OrderBy.Field, dir)
	}

	// Build LIMIT clause
	limitClause := ""
	if plan.Limit != nil {
		limitClause = fmt.Sprintf(" LIMIT %d", *plan.Limit)
	}

	query := fmt.Sprintf("SELECT %s FROM %s%s%s%s",
		selectClause, table, whereClause, orderClause, limitClause)

	return query, args, nil
}

// isFieldAllowed reports whether field may be safely interpolated into SQL
// for the given entity: either it's declared in that entity's
// schema.Entity.Fields allow-list, or it's a custom.* field (those are
// never interpolated as column names - see customFieldSQL - so they don't
// need a schema.Entity.Fields entry to be safe, only to be *meaningful*,
// which the AQL validator already checks upstream).
func isFieldAllowed(entity ast.EntityType, field string) bool {
	if schema.IsCustomFieldName(field) {
		return true
	}
	e := schema.GetEntity(entity)
	return e != nil && e.HasField(field)
}

// filterToSQL converts a planner.Filter to a SQL condition.
func (d *DB) filterToSQL(entity ast.EntityType, f planner.Filter) (string, []interface{}, error) {
	if !isFieldAllowed(entity, f.Field) {
		return "", nil, fmt.Errorf("field not allowed in WHERE for %s: %s", entity, f.Field)
	}
	if schema.IsCustomFieldName(f.Field) {
		return customFieldSQL(f.Op, schema.CustomFieldName(f.Field), f.Value)
	}

	field := f.Field
	var args []interface{}

	switch f.Op {
	case ast.OpEQ:
		return fmt.Sprintf("%s = ?", field), []interface{}{valueToArg(f.Value)}, nil
	case ast.OpNE:
		return fmt.Sprintf("%s != ?", field), []interface{}{valueToArg(f.Value)}, nil
	case ast.OpLT:
		return fmt.Sprintf("%s < ?", field), []interface{}{valueToArg(f.Value)}, nil
	case ast.OpLE:
		return fmt.Sprintf("%s <= ?", field), []interface{}{valueToArg(f.Value)}, nil
	case ast.OpGT:
		return fmt.Sprintf("%s > ?", field), []interface{}{valueToArg(f.Value)}, nil
	case ast.OpGE:
		return fmt.Sprintf("%s >= ?", field), []interface{}{valueToArg(f.Value)}, nil
	case ast.OpIN:
		if f.Value == nil || f.Value.Strings == nil {
			return "1=0", nil, nil // No values, always false
		}
		placeholders := make([]string, len(f.Value.Strings))
		for i, s := range f.Value.Strings {
			placeholders[i] = "?"
			args = append(args, s)
		}
		return fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ",")), args, nil
	case ast.OpNotIn:
		if f.Value == nil || f.Value.Strings == nil {
			return "1=1", nil, nil // No values, always true
		}
		placeholders := make([]string, len(f.Value.Strings))
		for i, s := range f.Value.Strings {
			placeholders[i] = "?"
			args = append(args, s)
		}
		return fmt.Sprintf("%s NOT IN (%s)", field, strings.Join(placeholders, ",")), args, nil
	case ast.OpContains:
		return fmt.Sprintf("%s LIKE ?", field), []interface{}{"%" + valueToString(f.Value) + "%"}, nil
	case ast.OpLike:
		// Convert SQL LIKE pattern (using % and _) to SQLite
		return fmt.Sprintf("%s LIKE ?", field), []interface{}{valueToString(f.Value)}, nil
	case ast.OpIsNull:
		return fmt.Sprintf("%s IS NULL", field), nil, nil
	case ast.OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", field), nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", f.Op)
	}
}

// customFieldSQL builds a predicate against the `data.custom_fields` array
// (see sync.go's customFieldsToMaps: a JSON array of {key, name, type,
// value} objects, not a nested map - so a dotted path like
// data.custom.priority never matches anything and json_each is the only
// way to search it). Fully parameterized - key and value are always bound
// arguments, never interpolated.
//
// Note: values are compared as text (json_extract's natural return type
// for a JSON string). Custom fields are stored as opaque values from
// Aha's API, so this covers the common "text/select/dropdown field equals
// X" case well; numeric comparison operators (<, <=, >, >=) do a text
// comparison rather than a numeric one for custom fields specifically -
// no known caller needs numeric custom-field range queries today, but
// this is a known simplification worth revisiting if one does.
func customFieldSQL(op ast.CompareOp, key string, v *ast.Value) (string, []interface{}, error) {
	const existsPrefix = `EXISTS (SELECT 1 FROM json_each(data, '$.custom_fields') je WHERE json_extract(je.value, '$.key') = ?`
	const notExistsPrefix = `NOT EXISTS (SELECT 1 FROM json_each(data, '$.custom_fields') je WHERE json_extract(je.value, '$.key') = ?`

	switch op {
	case ast.OpIsNull:
		return notExistsPrefix + ")", []interface{}{key}, nil
	case ast.OpIsNotNull:
		return existsPrefix + ")", []interface{}{key}, nil
	case ast.OpIN:
		if v == nil || v.Strings == nil {
			return "1=0", nil, nil
		}
		placeholders := make([]string, len(v.Strings))
		args := []interface{}{key}
		for i, s := range v.Strings {
			placeholders[i] = "?"
			args = append(args, s)
		}
		return existsPrefix + fmt.Sprintf(" AND json_extract(je.value, '$.value') IN (%s))", strings.Join(placeholders, ",")), args, nil
	case ast.OpNotIn:
		if v == nil || v.Strings == nil {
			return "1=1", nil, nil
		}
		placeholders := make([]string, len(v.Strings))
		args := []interface{}{key}
		for i, s := range v.Strings {
			placeholders[i] = "?"
			args = append(args, s)
		}
		return notExistsPrefix + fmt.Sprintf(" AND json_extract(je.value, '$.value') IN (%s))", strings.Join(placeholders, ",")), args, nil
	case ast.OpContains:
		return existsPrefix + " AND json_extract(je.value, '$.value') LIKE ?)", []interface{}{key, "%" + valueToString(v) + "%"}, nil
	case ast.OpLike:
		return existsPrefix + " AND json_extract(je.value, '$.value') LIKE ?)", []interface{}{key, valueToString(v)}, nil
	case ast.OpEQ:
		return existsPrefix + " AND json_extract(je.value, '$.value') = ?)", []interface{}{key, valueToString(v)}, nil
	case ast.OpNE:
		return notExistsPrefix + " AND json_extract(je.value, '$.value') = ?)", []interface{}{key, valueToString(v)}, nil
	case ast.OpLT:
		return existsPrefix + " AND json_extract(je.value, '$.value') < ?)", []interface{}{key, valueToString(v)}, nil
	case ast.OpLE:
		return existsPrefix + " AND json_extract(je.value, '$.value') <= ?)", []interface{}{key, valueToString(v)}, nil
	case ast.OpGT:
		return existsPrefix + " AND json_extract(je.value, '$.value') > ?)", []interface{}{key, valueToString(v)}, nil
	case ast.OpGE:
		return existsPrefix + " AND json_extract(je.value, '$.value') >= ?)", []interface{}{key, valueToString(v)}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operator for custom field: %s", op)
	}
}

func valueToArg(v *ast.Value) interface{} {
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
		if v.Bool {
			return 1
		}
		return 0
	case ast.ValueTypeTime:
		return v.Time
	default:
		return v.Raw
	}
}

func valueToString(v *ast.Value) string {
	if v == nil {
		return ""
	}
	switch v.Type {
	case ast.ValueTypeString:
		return v.String
	default:
		return v.Raw
	}
}

func normalizeValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	// SQLite returns int64 for integers
	switch val := v.(type) {
	case []byte:
		return string(val)
	case sql.NullString:
		if val.Valid {
			return val.String
		}
		return nil
	case sql.NullInt64:
		if val.Valid {
			return val.Int64
		}
		return nil
	case sql.NullFloat64:
		if val.Valid {
			return val.Float64
		}
		return nil
	case sql.NullBool:
		if val.Valid {
			return val.Bool
		}
		return nil
	default:
		return v
	}
}

// GetSyncStatus returns the sync status for all entities.
func (d *DB) GetSyncStatus(ctx context.Context, product string) (map[string]SyncStatus, error) {
	rows, err := d.ent.SyncMeta.Query().Where(syncmeta.Product(product)).All(ctx)
	if err != nil {
		return nil, err
	}

	status := make(map[string]SyncStatus)
	for _, r := range rows {
		status[r.Entity] = SyncStatus{
			LastSync:    r.LastSync.Format(time.RFC3339Nano),
			RecordCount: r.RecordCount,
		}
	}
	return status, nil
}

// SyncStatus contains sync status for an entity.
type SyncStatus struct {
	LastSync    string
	RecordCount int
}
