package result

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"gopkg.in/yaml.v3"
)

// Format represents an output format.
type Format string

// Output formats.
const (
	FormatTable    Format = "table"
	FormatJSON     Format = "json"
	FormatCSV      Format = "csv"
	FormatMarkdown Format = "markdown"
	FormatYAML     Format = "yaml"
	FormatHTML     Format = "html"
	FormatExcel    Format = "xlsx"
)

// Formatter formats query results for output.
type Formatter struct {
	format Format
	fields []string // specific fields to output (nil = all)
}

// NewFormatter creates a new formatter with the given format.
func NewFormatter(format Format) *Formatter {
	return &Formatter{format: format}
}

// WithFields sets specific fields to output.
func (f *Formatter) WithFields(fields []string) *Formatter {
	f.fields = fields
	return f
}

// Format formats the result and writes to the writer.
func (f *Formatter) Format(w io.Writer, result *Result) error {
	switch f.format {
	case FormatJSON:
		return f.formatJSON(w, result)
	case FormatCSV:
		return f.formatCSV(w, result)
	case FormatMarkdown:
		return f.formatMarkdown(w, result)
	case FormatYAML:
		return f.formatYAML(w, result)
	case FormatHTML:
		return f.formatHTML(w, result)
	case FormatTable:
		fallthrough
	default:
		return f.formatTable(w, result)
	}
}

// formatTable formats the result as an ASCII table.
func (f *Formatter) formatTable(w io.Writer, result *Result) error {
	if result.IsEmpty() {
		_, _ = fmt.Fprintln(w, "No results found.")
		return nil
	}

	// Determine columns
	columns := f.getColumns(result)

	// Create table with v1.x API
	table := tablewriter.NewTable(w,
		tablewriter.WithRowAlignmentConfig(tw.CellAlignment{Global: tw.AlignLeft}),
		tablewriter.WithHeaderAlignmentConfig(tw.CellAlignment{Global: tw.AlignLeft}),
	)

	// Set header
	table.Header(columns)

	// Add rows
	for _, rec := range result.Records {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = formatValue(rec.Get(col))
		}
		_ = table.Append(row)
	}

	_ = table.Render()

	// Print count
	_, _ = fmt.Fprintf(w, "\n%d record(s) returned.\n", result.Count())

	return nil
}

// formatJSON formats the result as JSON.
func (f *Formatter) formatJSON(w io.Writer, result *Result) error {
	output := struct {
		Entity  string   `json:"entity"`
		Count   int      `json:"count"`
		Records []Record `json:"records"`
	}{
		Entity:  string(result.Entity),
		Count:   result.Count(),
		Records: result.Records,
	}

	// Filter fields if specified
	if f.fields != nil {
		filteredRecords := make([]Record, len(result.Records))
		fieldSet := make(map[string]bool)
		for _, field := range f.fields {
			fieldSet[field] = true
		}

		for i, rec := range result.Records {
			filtered := make(Record)
			for k, v := range rec {
				if fieldSet[k] {
					filtered[k] = v
				}
			}
			filteredRecords[i] = filtered
		}
		output.Records = filteredRecords
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// formatCSV formats the result as CSV.
func (f *Formatter) formatCSV(w io.Writer, result *Result) error {
	if result.IsEmpty() {
		return nil
	}

	columns := f.getColumns(result)
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write(columns); err != nil {
		return err
	}

	// Write rows
	for _, rec := range result.Records {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = formatValue(rec.Get(col))
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// formatMarkdown formats the result as a Markdown table.
func (f *Formatter) formatMarkdown(w io.Writer, result *Result) error {
	if result.IsEmpty() {
		_, _ = fmt.Fprintln(w, "*No results found.*")
		return nil
	}

	columns := f.getColumns(result)

	// Write header
	_, _ = fmt.Fprint(w, "|")
	for _, col := range columns {
		_, _ = fmt.Fprintf(w, " %s |", col)
	}
	_, _ = fmt.Fprintln(w)

	// Write separator
	_, _ = fmt.Fprint(w, "|")
	for range columns {
		_, _ = fmt.Fprint(w, "---|")
	}
	_, _ = fmt.Fprintln(w)

	// Write rows
	for _, rec := range result.Records {
		_, _ = fmt.Fprint(w, "|")
		for _, col := range columns {
			value := formatValueForMarkdown(rec.Get(col))
			_, _ = fmt.Fprintf(w, " %s |", value)
		}
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintf(w, "\n*%d record(s)*\n", result.Count())

	return nil
}

// formatYAML formats the result as YAML.
func (f *Formatter) formatYAML(w io.Writer, result *Result) error {
	output := struct {
		Entity  string   `yaml:"entity"`
		Count   int      `yaml:"count"`
		Records []Record `yaml:"records"`
	}{
		Entity:  string(result.Entity),
		Count:   result.Count(),
		Records: result.Records,
	}

	// Filter fields if specified
	if f.fields != nil {
		filteredRecords := make([]Record, len(result.Records))
		fieldSet := make(map[string]bool)
		for _, field := range f.fields {
			fieldSet[field] = true
		}

		for i, rec := range result.Records {
			filtered := make(Record)
			for k, v := range rec {
				if fieldSet[k] {
					filtered[k] = v
				}
			}
			filteredRecords[i] = filtered
		}
		output.Records = filteredRecords
	}

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(output)
}

// formatHTML formats the result as an HTML table.
func (f *Formatter) formatHTML(w io.Writer, result *Result) error {
	if result.IsEmpty() {
		_, _ = fmt.Fprintln(w, "<p>No results found.</p>")
		return nil
	}

	columns := f.getColumns(result)

	const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>{{.Entity}} - Query Results</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 20px; }
    table { border-collapse: collapse; width: 100%; margin-top: 10px; }
    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
    th { background-color: #f5f5f5; font-weight: 600; }
    tr:nth-child(even) { background-color: #fafafa; }
    tr:hover { background-color: #f0f0f0; }
    .count { color: #666; margin-top: 10px; }
  </style>
</head>
<body>
  <h1>{{.Entity}}</h1>
  <table>
    <thead>
      <tr>
        {{range .Columns}}<th>{{.}}</th>{{end}}
      </tr>
    </thead>
    <tbody>
      {{range .Rows}}
      <tr>
        {{range .}}<td>{{.}}</td>{{end}}
      </tr>
      {{end}}
    </tbody>
  </table>
  <p class="count">{{.Count}} record(s)</p>
</body>
</html>
`

	// Build rows
	rows := make([][]string, len(result.Records))
	for i, rec := range result.Records {
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = formatValue(rec.Get(col))
		}
		rows[i] = row
	}

	data := struct {
		Entity  string
		Columns []string
		Rows    [][]string
		Count   int
	}{
		Entity:  string(result.Entity),
		Columns: columns,
		Rows:    rows,
		Count:   result.Count(),
	}

	tmpl, err := template.New("html").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}

	return tmpl.Execute(w, data)
}

// formatValueForMarkdown formats a value for Markdown, escaping pipe characters.
func formatValueForMarkdown(v any) string {
	s := formatValue(v)
	// Escape pipe characters which would break markdown tables
	s = strings.ReplaceAll(s, "|", "\\|")
	// Escape newlines
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// getColumns returns the columns to display.
func (f *Formatter) getColumns(result *Result) []string {
	if f.fields != nil {
		return f.fields
	}

	// Get all fields from the first record and sort them
	if result.IsEmpty() {
		return nil
	}

	// Use a consistent column order based on common fields
	commonOrder := []string{
		"id", "reference_num", "name", "status", "votes",
		"assigned_to", "tag", "created_at", "updated_at", "url",
	}

	columns := result.First().Keys()
	sort.Slice(columns, func(i, j int) bool {
		iOrder := indexOf(commonOrder, columns[i])
		jOrder := indexOf(commonOrder, columns[j])
		if iOrder == -1 && jOrder == -1 {
			return columns[i] < columns[j]
		}
		if iOrder == -1 {
			return false
		}
		if jOrder == -1 {
			return true
		}
		return iOrder < jOrder
	})

	return columns
}

// formatValue formats a value for display.
func formatValue(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return truncate(val, 50)
	case time.Time:
		return val.Format("2006-01-02 15:04")
	case bool:
		if val {
			return "yes"
		}
		return "no"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// indexOf returns the index of s in slice, or -1 if not found.
func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

// ParseFormat parses a format string.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "csv":
		return FormatCSV, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "html":
		return FormatHTML, nil
	case "xlsx", "excel":
		return FormatExcel, nil
	default:
		return "", fmt.Errorf("unknown format: %s (valid: table, json, csv, markdown, yaml, html, xlsx)", s)
	}
}

// IsFileFormat returns true if the format requires writing to a file.
func (f Format) IsFileFormat() bool {
	return f == FormatExcel
}
