package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/planner"
	"github.com/grokify/aha-studio/result"
)

// QueryOffline executes a query against the local SQLite database.
func (d *DB) QueryOffline(plan *planner.Plan) (*result.Result, error) {
	// Build SQL query from plan
	sqlQuery, args, err := d.buildSQL(plan)
	if err != nil {
		return nil, fmt.Errorf("building SQL: %w", err)
	}

	// Execute query
	rows, err := d.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	// Collect results
	var records []result.Record
	for rows.Next() {
		// Create a slice of interface{} to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		// Convert to record
		rec := make(result.Record)
		for i, col := range columns {
			if col == "data" {
				// Parse JSON data column if present
				if s, ok := values[i].(string); ok && s != "" {
					var data map[string]any
					if err := json.Unmarshal([]byte(s), &data); err == nil {
						for k, v := range data {
							if _, exists := rec[k]; !exists {
								rec[k] = v
							}
						}
					}
				}
			} else {
				rec[col] = normalizeValue(values[i])
			}
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	return &result.Result{
		Entity:  plan.Entity,
		Records: records,
	}, nil
}

// buildSQL converts a planner.Plan to a SQL query.
func (d *DB) buildSQL(plan *planner.Plan) (string, []interface{}, error) {
	var args []interface{}

	// Determine table name
	table := string(plan.Entity)

	// Build SELECT clause
	selectClause := "*"
	if len(plan.SelectFields) > 0 {
		selectClause = strings.Join(plan.SelectFields, ", ")
	}

	// Build WHERE clause
	whereClause := ""
	if len(plan.ClientFilters) > 0 {
		var conditions []string
		for _, f := range plan.ClientFilters {
			cond, arg, err := d.filterToSQL(f)
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

// filterToSQL converts a planner.Filter to a SQL condition.
func (d *DB) filterToSQL(f planner.Filter) (string, []interface{}, error) {
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
func (d *DB) GetSyncStatus(product string) (map[string]SyncStatus, error) {
	rows, err := d.db.Query(
		"SELECT entity, last_sync, record_count FROM sync_meta WHERE product = ?",
		product,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	status := make(map[string]SyncStatus)
	for rows.Next() {
		var entity string
		var s SyncStatus
		if err := rows.Scan(&entity, &s.LastSync, &s.RecordCount); err != nil {
			return nil, err
		}
		status[entity] = s
	}

	return status, rows.Err()
}

// SyncStatus contains sync status for an entity.
type SyncStatus struct {
	LastSync    string
	RecordCount int
}
