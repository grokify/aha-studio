package result

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
)

func TestRecordGet(t *testing.T) {
	rec := Record{
		"name":   "Test Feature",
		"status": "Done",
		"votes":  10,
	}

	tests := []struct {
		key      string
		expected any
	}{
		{"name", "Test Feature"},
		{"status", "Done"},
		{"votes", 10},
		{"missing", nil},
	}

	for _, tt := range tests {
		got := rec.Get(tt.key)
		if got != tt.expected {
			t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.expected)
		}
	}
}

func TestRecordKeys(t *testing.T) {
	rec := Record{
		"name":   "Test",
		"status": "Done",
		"votes":  10,
	}

	keys := rec.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Keys should contain all record keys
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	for k := range rec {
		if !keySet[k] {
			t.Errorf("missing key %q", k)
		}
	}
}

func TestResultIsEmpty(t *testing.T) {
	empty := &Result{
		Entity:  ast.EntityFeatures,
		Records: nil,
	}
	if !empty.IsEmpty() {
		t.Error("expected IsEmpty() = true for nil records")
	}

	emptySlice := &Result{
		Entity:  ast.EntityFeatures,
		Records: []Record{},
	}
	if !emptySlice.IsEmpty() {
		t.Error("expected IsEmpty() = true for empty slice")
	}

	nonEmpty := &Result{
		Entity:  ast.EntityFeatures,
		Records: []Record{{"name": "Test"}},
	}
	if nonEmpty.IsEmpty() {
		t.Error("expected IsEmpty() = false for non-empty result")
	}
}

func TestResultCount(t *testing.T) {
	tests := []struct {
		records []Record
		count   int
	}{
		{nil, 0},
		{[]Record{}, 0},
		{[]Record{{"a": 1}}, 1},
		{[]Record{{"a": 1}, {"b": 2}, {"c": 3}}, 3},
	}

	for _, tt := range tests {
		r := &Result{Records: tt.records}
		if got := r.Count(); got != tt.count {
			t.Errorf("Count() = %d, want %d", got, tt.count)
		}
	}
}

func TestResultFirst(t *testing.T) {
	empty := &Result{Records: nil}
	if empty.First() != nil {
		t.Error("expected nil for empty result")
	}

	nonEmpty := &Result{
		Records: []Record{
			{"name": "First"},
			{"name": "Second"},
		},
	}
	first := nonEmpty.First()
	if first == nil || first["name"] != "First" {
		t.Error("expected first record")
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		hasError bool
	}{
		{"table", FormatTable, false},
		{"TABLE", FormatTable, false},
		{"", FormatTable, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"csv", FormatCSV, false},
		{"CSV", FormatCSV, false},
		{"unknown", "", true},
		{"xml", "", true},
	}

	for _, tt := range tests {
		got, err := ParseFormat(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseFormat(%q) expected error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseFormat(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("ParseFormat(%q) = %s, want %s", tt.input, got, tt.expected)
			}
		}
	}
}

func TestFormatterJSON(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done"},
			{"name": "Feature 2", "status": "Pending"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatJSON)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	// Verify JSON is valid
	var output struct {
		Entity  string   `json:"entity"`
		Count   int      `json:"count"`
		Records []Record `json:"records"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if output.Entity != "features" {
		t.Errorf("expected entity 'features', got %q", output.Entity)
	}
	if output.Count != 2 {
		t.Errorf("expected count 2, got %d", output.Count)
	}
	if len(output.Records) != 2 {
		t.Errorf("expected 2 records, got %d", len(output.Records))
	}
}

func TestFormatterJSONWithFields(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done", "votes": 10},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatJSON).WithFields([]string{"name", "status"})
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	var output struct {
		Records []Record `json:"records"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Only specified fields should be present
	rec := output.Records[0]
	if rec["name"] == nil || rec["status"] == nil {
		t.Error("expected name and status in filtered output")
	}
	if rec["votes"] != nil {
		t.Error("votes should not be in filtered output")
	}
}

func TestFormatterCSV(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done"},
			{"name": "Feature 2", "status": "Pending"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatCSV).WithFields([]string{"name", "status"})
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 2 data rows
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}

	// Check header
	if lines[0] != "name,status" {
		t.Errorf("expected header 'name,status', got %q", lines[0])
	}
}

func TestFormatterCSVEmpty(t *testing.T) {
	result := &Result{
		Entity:  ast.EntityFeatures,
		Records: nil,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatCSV)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	// Empty result should produce no output
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestFormatterTable(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done"},
			{"name": "Feature 2", "status": "Pending"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatTable)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()

	// Should contain the record count
	if !strings.Contains(output, "2 record(s) returned") {
		t.Error("expected '2 record(s) returned' in output")
	}

	// Should contain the feature names
	if !strings.Contains(output, "Feature 1") {
		t.Error("expected 'Feature 1' in output")
	}
	if !strings.Contains(output, "Feature 2") {
		t.Error("expected 'Feature 2' in output")
	}
}

func TestFormatterTableEmpty(t *testing.T) {
	result := &Result{
		Entity:  ast.EntityFeatures,
		Records: nil,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatTable)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No results found") {
		t.Errorf("expected 'No results found', got %q", output)
	}
}

func TestFormatValue(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		input    any
		expected string
	}{
		{nil, ""},
		{"Hello", "Hello"},
		{true, "yes"},
		{false, "no"},
		{now, "2024-01-15 10:30"},
		{123, "123"},
		{3.14, "3.14"},
	}

	for _, tt := range tests {
		got := formatValue(tt.input)
		if got != tt.expected {
			t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestFormatValueTruncate(t *testing.T) {
	longString := strings.Repeat("a", 100)
	result := formatValue(longString)

	// Should be truncated to 50 chars with "..."
	if len(result) != 50 {
		t.Errorf("expected length 50, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("expected truncated string to end with '...'")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Hello", 10, "Hello"},
		{"Hello", 5, "Hello"},
		{"Hello World", 8, "Hello..."},
		{"", 5, ""},
		{"abc", 3, "abc"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.expected)
		}
	}
}

func TestIndexOf(t *testing.T) {
	slice := []string{"a", "b", "c", "d"}

	tests := []struct {
		s        string
		expected int
	}{
		{"a", 0},
		{"b", 1},
		{"c", 2},
		{"d", 3},
		{"e", -1},
		{"", -1},
	}

	for _, tt := range tests {
		got := indexOf(slice, tt.s)
		if got != tt.expected {
			t.Errorf("indexOf(%q) = %d, want %d", tt.s, got, tt.expected)
		}
	}
}

func TestGetColumns(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Test", "status": "Done", "custom_field": "value"},
		},
	}

	// Without specified fields
	formatter := NewFormatter(FormatTable)
	columns := formatter.getColumns(result)

	// Should contain all keys
	if len(columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(columns))
	}

	// With specified fields
	formatter = NewFormatter(FormatTable).WithFields([]string{"name", "status"})
	columns = formatter.getColumns(result)

	if len(columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(columns))
	}
	if columns[0] != "name" || columns[1] != "status" {
		t.Error("expected columns to match specified fields")
	}
}

func TestColumnOrdering(t *testing.T) {
	// Test that common columns come first in expected order
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{
				"custom":        "value",
				"name":          "Test",
				"id":            "123",
				"status":        "Done",
				"reference_num": "FEAT-1",
			},
		},
	}

	formatter := NewFormatter(FormatTable)
	columns := formatter.getColumns(result)

	// id should come before name, name before status, etc.
	// Custom columns should come last
	idIdx := indexOf(columns, "id")
	nameIdx := indexOf(columns, "name")
	statusIdx := indexOf(columns, "status")
	customIdx := indexOf(columns, "custom")

	if idIdx > nameIdx {
		t.Error("id should come before name")
	}
	if nameIdx > statusIdx {
		t.Error("name should come before status")
	}
	if statusIdx > customIdx {
		t.Error("standard columns should come before custom")
	}
}

func TestParseFormatNewFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		hasError bool
	}{
		{"markdown", FormatMarkdown, false},
		{"md", FormatMarkdown, false},
		{"MARKDOWN", FormatMarkdown, false},
		{"yaml", FormatYAML, false},
		{"yml", FormatYAML, false},
		{"YAML", FormatYAML, false},
		{"html", FormatHTML, false},
		{"HTML", FormatHTML, false},
	}

	for _, tt := range tests {
		got, err := ParseFormat(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("ParseFormat(%q) expected error", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseFormat(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("ParseFormat(%q) = %s, want %s", tt.input, got, tt.expected)
			}
		}
	}
}

func TestFormatterMarkdown(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done"},
			{"name": "Feature 2", "status": "Pending"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatMarkdown).WithFields([]string{"name", "status"})
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()

	// Check header row
	if !strings.Contains(output, "| name |") {
		t.Error("expected markdown header with 'name'")
	}
	if !strings.Contains(output, "| status |") {
		t.Error("expected markdown header with 'status'")
	}

	// Check separator row
	if !strings.Contains(output, "|---|") {
		t.Error("expected markdown separator")
	}

	// Check data rows
	if !strings.Contains(output, "Feature 1") {
		t.Error("expected 'Feature 1' in output")
	}
	if !strings.Contains(output, "Feature 2") {
		t.Error("expected 'Feature 2' in output")
	}

	// Check record count
	if !strings.Contains(output, "*2 record(s)*") {
		t.Error("expected '*2 record(s)*' in output")
	}
}

func TestFormatterMarkdownEmpty(t *testing.T) {
	result := &Result{
		Entity:  ast.EntityFeatures,
		Records: nil,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatMarkdown)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "*No results found.*") {
		t.Errorf("expected '*No results found.*', got %q", output)
	}
}

func TestFormatterMarkdownEscapesPipes(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature | with pipe", "status": "Done"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatMarkdown).WithFields([]string{"name", "status"})
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()
	// Pipe should be escaped
	if !strings.Contains(output, `Feature \| with pipe`) {
		t.Errorf("expected escaped pipe, got %q", output)
	}
}

func TestFormatterYAML(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatYAML)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()

	// Check for YAML structure
	if !strings.Contains(output, "entity: features") {
		t.Error("expected 'entity: features' in YAML output")
	}
	if !strings.Contains(output, "count: 1") {
		t.Error("expected 'count: 1' in YAML output")
	}
	if !strings.Contains(output, "records:") {
		t.Error("expected 'records:' in YAML output")
	}
	if !strings.Contains(output, "name: Feature 1") {
		t.Error("expected 'name: Feature 1' in YAML output")
	}
}

func TestFormatterHTML(t *testing.T) {
	result := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{"name": "Feature 1", "status": "Done"},
			{"name": "Feature 2", "status": "Pending"},
		},
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatHTML).WithFields([]string{"name", "status"})
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()

	// Check for HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(output, "<table>") {
		t.Error("expected <table> tag")
	}
	if !strings.Contains(output, "<th>name</th>") {
		t.Error("expected header with 'name'")
	}
	if !strings.Contains(output, "<td>Feature 1</td>") {
		t.Error("expected data cell with 'Feature 1'")
	}
	if !strings.Contains(output, "2 record(s)") {
		t.Error("expected record count")
	}
}

func TestFormatterHTMLEmpty(t *testing.T) {
	result := &Result{
		Entity:  ast.EntityFeatures,
		Records: nil,
	}

	var buf bytes.Buffer
	formatter := NewFormatter(FormatHTML)
	err := formatter.Format(&buf, result)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "<p>No results found.</p>") {
		t.Errorf("expected '<p>No results found.</p>', got %q", output)
	}
}

func TestRecordCustomFields(t *testing.T) {
	rec := Record{
		"name":   "Test Feature",
		"status": "Done",
	}

	// Test SetCustomField
	rec.SetCustomField("priority", "High")
	rec.SetCustomField("team", "Platform")

	// Test Get with custom.* prefix
	if got := rec.Get("custom.priority"); got != "High" {
		t.Errorf("Get(custom.priority) = %v, want High", got)
	}
	if got := rec.Get("custom.team"); got != "Platform" {
		t.Errorf("Get(custom.team) = %v, want Platform", got)
	}

	// Test HasCustomFields
	if !rec.HasCustomFields() {
		t.Error("expected HasCustomFields() = true")
	}

	// Test CustomFieldKeys
	keys := rec.CustomFieldKeys()
	if len(keys) != 2 {
		t.Errorf("expected 2 custom field keys, got %d", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["priority"] || !keySet["team"] {
		t.Error("expected custom field keys to include priority and team")
	}
}

func TestRecordNoCustomFields(t *testing.T) {
	rec := Record{
		"name":   "Test",
		"status": "Done",
	}

	if rec.HasCustomFields() {
		t.Error("expected HasCustomFields() = false for record without custom fields")
	}

	keys := rec.CustomFieldKeys()
	if len(keys) != 0 {
		t.Errorf("expected 0 custom field keys, got %d", len(keys))
	}
}

func TestRecordCustomFieldGet(t *testing.T) {
	rec := Record{
		"name":            "Test",
		"custom.priority": "High",
		"custom.score":    42,
	}

	tests := []struct {
		key      string
		expected any
	}{
		{"name", "Test"},
		{"custom.priority", "High"},
		{"custom.score", 42},
		{"custom.missing", nil},
		{"missing", nil},
	}

	for _, tt := range tests {
		got := rec.Get(tt.key)
		if got != tt.expected {
			t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.expected)
		}
	}
}
