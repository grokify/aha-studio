// Package result provides result types for AQL query execution.
package result

import (
	"strings"

	"github.com/grokify/aha-studio/aql/ast"
)

// Result represents the result of an AQL query execution.
type Result struct {
	// Entity is the entity type that was queried.
	Entity ast.EntityType

	// Records contains the query results.
	Records []Record

	// Metadata contains optional result metadata.
	Metadata Metadata

	// Stats contains execution statistics (optional).
	Stats *ExecutionStats
}

// Record represents a single result row as a map of field names to values.
type Record map[string]any

// Get returns the value for a field, or nil if not found.
// Supports custom field lookup using "custom.fieldname" syntax.
func (r Record) Get(field string) any {
	// Check for custom field syntax (custom.fieldname)
	if strings.HasPrefix(field, "custom.") {
		return r[field] // Custom fields are stored with full key
	}
	return r[field]
}

// GetString returns the string value for a field, or empty string if not found.
func (r Record) GetString(field string) string {
	v, ok := r[field]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetInt returns the int value for a field, or 0 if not found.
func (r Record) GetInt(field string) int {
	v, ok := r[field]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// Keys returns all field names in the record.
func (r Record) Keys() []string {
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	return keys
}

// SetCustomField sets a custom field value using the "custom.key" format.
func (r Record) SetCustomField(key string, value any) {
	r["custom."+key] = value
}

// CustomFieldKeys returns all custom field keys (without the "custom." prefix).
func (r Record) CustomFieldKeys() []string {
	var keys []string
	for k := range r {
		if strings.HasPrefix(k, "custom.") {
			keys = append(keys, strings.TrimPrefix(k, "custom."))
		}
	}
	return keys
}

// HasCustomFields returns true if the record has any custom fields.
func (r Record) HasCustomFields() bool {
	for k := range r {
		if strings.HasPrefix(k, "custom.") {
			return true
		}
	}
	return false
}

// Metadata contains optional result metadata.
type Metadata struct {
	// TotalCount is the total number of matching records (before LIMIT).
	TotalCount int

	// ExecutionTimeMs is the query execution time in milliseconds.
	ExecutionTimeMs int64

	// IsTruncated indicates if results were truncated due to LIMIT.
	IsTruncated bool
}

// ExecutionStats contains query execution statistics.
type ExecutionStats struct {
	// ExecutionTime is the total execution time.
	ExecutionTime int64 // milliseconds

	// APICallCount is the number of API calls made.
	APICallCount int

	// RecordsFetched is the total number of records fetched from API.
	RecordsFetched int

	// RecordsReturned is the number of records in the final result.
	RecordsReturned int

	// CacheHit indicates if the result was served from cache.
	CacheHit bool

	// FiltersPushed is the number of filters pushed to the API.
	FiltersPushed int

	// ClientFilters is the number of filters applied client-side.
	ClientFilters int
}

// Count returns the number of records in the result.
func (r *Result) Count() int {
	return len(r.Records)
}

// IsEmpty returns true if there are no records.
func (r *Result) IsEmpty() bool {
	return len(r.Records) == 0
}

// First returns the first record, or nil if empty.
func (r *Result) First() Record {
	if len(r.Records) == 0 {
		return nil
	}
	return r.Records[0]
}

// AllFields returns all unique field names across all records.
func (r *Result) AllFields() []string {
	fieldSet := make(map[string]bool)
	for _, rec := range r.Records {
		for k := range rec {
			fieldSet[k] = true
		}
	}

	fields := make([]string, 0, len(fieldSet))
	for k := range fieldSet {
		fields = append(fields, k)
	}
	return fields
}
