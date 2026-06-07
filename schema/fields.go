package schema

// FieldType represents the data type of a field.
type FieldType int

// Field types.
const (
	FieldTypeString FieldType = iota
	FieldTypeInt
	FieldTypeFloat
	FieldTypeBool
	FieldTypeDate
	FieldTypeDatetime
	FieldTypeStringArray
)

// String returns the string representation of the field type.
func (t FieldType) String() string {
	switch t {
	case FieldTypeString:
		return "string"
	case FieldTypeInt:
		return "int"
	case FieldTypeFloat:
		return "float"
	case FieldTypeBool:
		return "bool"
	case FieldTypeDate:
		return "date"
	case FieldTypeDatetime:
		return "datetime"
	case FieldTypeStringArray:
		return "string_array"
	default:
		return "unknown"
	}
}

// Field represents a field definition in an entity schema.
type Field struct {
	// Name is the field name.
	Name string

	// Type is the field's data type.
	Type FieldType

	// Filterable indicates if the field can be used in WHERE clauses.
	Filterable bool

	// Sortable indicates if the field can be used in ORDER BY clauses.
	Sortable bool

	// APIParam is the Aha API parameter name for this field.
	// If empty, the field can only be filtered client-side.
	APIParam string

	// CustomField indicates if this is a custom field (prefixed with "custom.").
	CustomField bool
}

// IsPushable returns true if filters on this field can be pushed to the Aha API.
func (f *Field) IsPushable() bool {
	return f.APIParam != ""
}

// CustomFieldPrefix is the prefix for custom field names.
const CustomFieldPrefix = "custom."

// IsCustomFieldName returns true if the field name indicates a custom field.
func IsCustomFieldName(name string) bool {
	return len(name) > len(CustomFieldPrefix) &&
		name[:len(CustomFieldPrefix)] == CustomFieldPrefix
}

// CustomFieldName extracts the custom field name without the prefix.
func CustomFieldName(name string) string {
	if IsCustomFieldName(name) {
		return name[len(CustomFieldPrefix):]
	}
	return name
}
