package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// recordEntityTables is the allow-list of entity types readable via
// ListRecords/GetRecord, mapping entity type to its table name (identical
// today, but the indirection keeps table names out of caller-supplied
// strings).
var recordEntityTables = map[string]string{
	"features":    "features",
	"initiatives": "initiatives",
	"epics":       "epics",
	"releases":    "releases",
}

// ListRecords returns all cached records of an entity type
// ("features"|"initiatives"|"epics"|"releases"), optionally filtered by
// product (empty product = all products). Records are returned in the
// flattened shape QueryOffline uses: real columns plus the data JSON
// column's keys merged in (column values win), so custom_fields — when the
// record was detail-synced — surfaces as a []any of {key,name,value,type}
// maps.
func (d *DB) ListRecords(ctx context.Context, entityType, product string) ([]map[string]any, error) {
	table, ok := recordEntityTables[entityType]
	if !ok {
		return nil, fmt.Errorf("unsupported entity type %q", entityType)
	}

	query := "SELECT * FROM " + table //nolint:gosec // G202: table from allow-list above
	var args []any
	if product != "" {
		query += " WHERE product = ?"
		args = append(args, product)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	return rowsToFlattenedRecords(rows)
}

// GetRecord returns one cached record by entity type and ID (or reference
// number), in the same flattened shape as ListRecords. Returns
// sql.ErrNoRows when no record matches.
func (d *DB) GetRecord(ctx context.Context, entityType, id string) (map[string]any, error) {
	table, ok := recordEntityTables[entityType]
	if !ok {
		return nil, fmt.Errorf("unsupported entity type %q", entityType)
	}

	query := "SELECT * FROM " + table + " WHERE id = ? OR reference_num = ? LIMIT 1" //nolint:gosec // G202: table from allow-list above
	rows, err := d.db.QueryContext(ctx, query, id, id)
	if err != nil {
		return nil, fmt.Errorf("querying %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	records, err := rowsToFlattenedRecords(rows)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

// rowsToFlattenedRecords scans rows into maps, merging each row's data
// JSON column into the record (real column values win on key conflicts) —
// the same flattening QueryOffline performs.
func rowsToFlattenedRecords(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	var records []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		rec := make(map[string]any)
		for i, col := range columns {
			if col == "data" {
				// The JSON column arrives as string or []byte depending on
				// how the driver types the read.
				var raw []byte
				switch v := values[i].(type) {
				case string:
					raw = []byte(v)
				case []byte:
					raw = v
				}
				if len(raw) > 0 {
					var data map[string]any
					if err := json.Unmarshal(raw, &data); err == nil {
						for k, v := range data {
							if _, exists := rec[k]; !exists {
								rec[k] = v
							}
						}
					}
				}
				continue
			}
			rec[col] = normalizeValue(values[i])
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return records, nil
}
