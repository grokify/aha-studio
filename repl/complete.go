package repl

import (
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/schema"
)

// complete provides context-aware tab completion.
func (r *REPL) complete(d prompt.Document) []prompt.Suggest {
	text := d.TextBeforeCursor()
	word := d.GetWordBeforeCursor()

	// Empty input - suggest starting keywords
	if strings.TrimSpace(text) == "" {
		return filterSuggestions(startSuggestions, word)
	}

	// Dot commands
	if strings.HasPrefix(text, ".") {
		return filterSuggestions(commandSuggestions, word)
	}

	// Parse context
	textLower := strings.ToLower(text)

	// After SELECT - suggest fields and aggregates
	if strings.HasSuffix(textLower, "select ") || strings.HasSuffix(textLower, "select distinct ") {
		entity := extractEntity(textLower)
		suggestions := append(aggregateSuggestions, fieldSuggestionsFor(entity)...)
		suggestions = append(suggestions, prompt.Suggest{Text: "*", Description: "All fields"})
		return filterSuggestions(suggestions, word)
	}

	// After FROM - suggest entities
	if strings.HasSuffix(textLower, "from ") {
		return filterSuggestions(entitySuggestions, word)
	}

	// After JOIN - suggest entities
	if strings.HasSuffix(textLower, "join ") {
		return filterSuggestions(entitySuggestions, word)
	}

	// After GROUP - suggest BY
	if strings.HasSuffix(textLower, "group ") {
		return []prompt.Suggest{{Text: "BY", Description: "Group by fields"}}
	}

	// After GROUP BY - suggest fields
	if strings.HasSuffix(textLower, "group by ") {
		entity := extractEntity(textLower)
		return filterSuggestions(fieldSuggestionsFor(entity), word)
	}

	// After HAVING - suggest aggregate conditions
	if strings.HasSuffix(textLower, "having ") {
		return filterSuggestions(aggregateSuggestions, word)
	}

	// After WHERE or AND/OR - suggest fields
	if endsWithKeyword(textLower, "where", "and", "or", "not") {
		entity := extractEntity(textLower)
		return filterSuggestions(fieldSuggestionsFor(entity), word)
	}

	// After field name - suggest operators
	if isAfterFieldName(textLower) {
		return filterSuggestions(operatorSuggestions, word)
	}

	// After = or IN - suggest values (if we know the field)
	if endsWithKeyword(textLower, "=", "in", "contains", "like") {
		// For status fields, suggest common values
		if containsWord(textLower, "status") {
			return filterSuggestions(statusSuggestions, word)
		}
	}

	// After ORDER - suggest BY
	if strings.HasSuffix(textLower, "order ") {
		return []prompt.Suggest{{Text: "BY", Description: "Sort direction"}}
	}

	// After ORDER BY - suggest fields
	if strings.HasSuffix(textLower, "order by ") {
		entity := extractEntity(textLower)
		return filterSuggestions(sortableFieldsFor(entity), word)
	}

	// After ORDER BY field - suggest ASC/DESC
	if isAfterOrderByField(textLower) {
		return filterSuggestions(sortDirectionSuggestions, word)
	}

	// After ASC/DESC or after closing paren - suggest LIMIT or more conditions
	if endsWithKeyword(textLower, "asc", "desc", ")") {
		return filterSuggestions(continuationSuggestions, word)
	}

	// Default - suggest keywords
	return filterSuggestions(keywordSuggestions, word)
}

// Suggestion lists

var startSuggestions = []prompt.Suggest{
	{Text: "SELECT", Description: "Select fields/aggregates"},
	{Text: "FROM", Description: "Start a query"},
}

var commandSuggestions = []prompt.Suggest{
	{Text: ".help", Description: "Show help"},
	{Text: ".exit", Description: "Exit REPL"},
	{Text: ".output", Description: "Set output format"},
	{Text: ".product", Description: "Set product context"},
	{Text: ".verbose", Description: "Toggle verbose mode"},
	{Text: ".history", Description: "Show query history"},
	{Text: ".clear", Description: "Clear screen"},
	{Text: ".save", Description: "Save last query"},
	{Text: ".run", Description: "Run saved query"},
	{Text: ".queries", Description: "List saved queries"},
	{Text: ".delete", Description: "Delete saved query"},
}

var entitySuggestions = []prompt.Suggest{
	{Text: "features", Description: "Product features"},
	{Text: "ideas", Description: "Customer ideas"},
	{Text: "releases", Description: "Product releases"},
	{Text: "initiatives", Description: "Strategic initiatives"},
}

var keywordSuggestions = []prompt.Suggest{
	{Text: "SELECT", Description: "Select fields/aggregates"},
	{Text: "FROM", Description: "Specify entity"},
	{Text: "JOIN", Description: "Join entities"},
	{Text: "LEFT JOIN", Description: "Left outer join"},
	{Text: "WHERE", Description: "Filter conditions"},
	{Text: "GROUP BY", Description: "Group results"},
	{Text: "HAVING", Description: "Filter after grouping"},
	{Text: "ORDER BY", Description: "Sort results"},
	{Text: "LIMIT", Description: "Limit results"},
	{Text: "AND", Description: "Logical AND"},
	{Text: "OR", Description: "Logical OR"},
	{Text: "NOT", Description: "Logical NOT"},
}

var aggregateSuggestions = []prompt.Suggest{
	{Text: "COUNT(*)", Description: "Count all records"},
	{Text: "COUNT(", Description: "Count field values"},
	{Text: "SUM(", Description: "Sum field values"},
	{Text: "AVG(", Description: "Average field values"},
	{Text: "MIN(", Description: "Minimum value"},
	{Text: "MAX(", Description: "Maximum value"},
	{Text: "DISTINCT", Description: "Distinct values"},
}

var operatorSuggestions = []prompt.Suggest{
	{Text: "=", Description: "Equals"},
	{Text: "!=", Description: "Not equals"},
	{Text: "<", Description: "Less than"},
	{Text: "<=", Description: "Less than or equal"},
	{Text: ">", Description: "Greater than"},
	{Text: ">=", Description: "Greater than or equal"},
	{Text: "IN", Description: "In list"},
	{Text: "NOT IN", Description: "Not in list"},
	{Text: "CONTAINS", Description: "Contains substring"},
	{Text: "LIKE", Description: "Pattern match"},
	{Text: "IS NULL", Description: "Is null"},
	{Text: "IS NOT NULL", Description: "Is not null"},
}

var statusSuggestions = []prompt.Suggest{
	{Text: `"In Progress"`, Description: "Work in progress"},
	{Text: `"Done"`, Description: "Completed"},
	{Text: `"Ready"`, Description: "Ready to start"},
	{Text: `"Backlog"`, Description: "In backlog"},
	{Text: `"New"`, Description: "Newly created"},
	{Text: `"Under Review"`, Description: "Being reviewed"},
}

var sortDirectionSuggestions = []prompt.Suggest{
	{Text: "ASC", Description: "Ascending order"},
	{Text: "DESC", Description: "Descending order"},
}

var continuationSuggestions = []prompt.Suggest{
	{Text: "GROUP BY", Description: "Group results"},
	{Text: "HAVING", Description: "Filter after grouping"},
	{Text: "ORDER BY", Description: "Sort results"},
	{Text: "LIMIT", Description: "Limit results"},
	{Text: "AND", Description: "Add condition"},
	{Text: "OR", Description: "Alternative condition"},
}

// fieldSuggestionsFor returns field suggestions for an entity.
func fieldSuggestionsFor(entityName string) []prompt.Suggest {
	entity := schema.GetEntity(ast.EntityType(entityName))
	if entity == nil {
		return commonFieldSuggestions
	}

	var suggestions []prompt.Suggest
	for name, field := range entity.Fields {
		if field.Filterable {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        name,
				Description: field.Type.String(),
			})
		}
	}
	return suggestions
}

// sortableFieldsFor returns sortable field suggestions for an entity.
func sortableFieldsFor(entityName string) []prompt.Suggest {
	entity := schema.GetEntity(ast.EntityType(entityName))
	if entity == nil {
		return commonFieldSuggestions
	}

	var suggestions []prompt.Suggest
	for name, field := range entity.Fields {
		if field.Sortable {
			suggestions = append(suggestions, prompt.Suggest{
				Text:        name,
				Description: field.Type.String(),
			})
		}
	}
	return suggestions
}

var commonFieldSuggestions = []prompt.Suggest{
	{Text: "id", Description: "Record ID"},
	{Text: "name", Description: "Name"},
	{Text: "reference_num", Description: "Reference number"},
	{Text: "status", Description: "Workflow status"},
	{Text: "created_at", Description: "Creation date"},
	{Text: "updated_at", Description: "Last update date"},
}

// Helper functions

func filterSuggestions(suggestions []prompt.Suggest, word string) []prompt.Suggest {
	if word == "" {
		return suggestions
	}

	var filtered []prompt.Suggest
	wordLower := strings.ToLower(word)
	for _, s := range suggestions {
		if strings.HasPrefix(strings.ToLower(s.Text), wordLower) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func endsWithKeyword(text string, keywords ...string) bool {
	text = strings.TrimSpace(text)
	for _, kw := range keywords {
		if strings.HasSuffix(text, kw+" ") || strings.HasSuffix(text, kw) {
			return true
		}
	}
	return false
}

func containsWord(text, word string) bool {
	return strings.Contains(text, word+" ") || strings.HasSuffix(text, word)
}

func extractEntity(text string) string {
	// Find "FROM <entity>"
	idx := strings.Index(text, "from ")
	if idx == -1 {
		return ""
	}
	rest := text[idx+5:]
	fields := strings.Fields(rest)
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func isAfterFieldName(text string) bool {
	// Check if the last word before cursor looks like a field name
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return false
	}

	// Last word should be a field name (lowercase, possibly with underscore)
	lastWord := fields[len(fields)-1]
	if strings.HasSuffix(text, " ") {
		// Cursor is after a space, check if previous word is a field
		for _, entity := range []ast.EntityType{ast.EntityFeatures, ast.EntityIdeas, ast.EntityReleases, ast.EntityInitiatives} {
			e := schema.GetEntity(entity)
			if e != nil && e.HasField(lastWord) {
				return true
			}
		}
	}
	return false
}

func isAfterOrderByField(text string) bool {
	// Check for "ORDER BY field " pattern
	idx := strings.Index(text, "order by ")
	if idx == -1 {
		return false
	}
	rest := text[idx+9:]
	fields := strings.Fields(rest)
	// Should have exactly one field and end with space
	return len(fields) == 1 && strings.HasSuffix(text, " ")
}
