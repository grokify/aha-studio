package result

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
)

func TestExcelFormatter_FormatToFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.xlsx")

	// Create test data
	res := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{
				"id":           "123",
				"name":         "Test Feature",
				"status":       "In Progress",
				"created_at":   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				"votes":        int64(42),
				"is_completed": false,
			},
			{
				"id":           "456",
				"name":         "Another Feature",
				"status":       "Done",
				"created_at":   time.Date(2024, 2, 20, 14, 0, 0, 0, time.UTC),
				"votes":        int64(100),
				"is_completed": true,
			},
		},
	}

	// Format to Excel
	formatter := NewExcelFormatter()
	err := formatter.FormatToFile(filePath, res)
	if err != nil {
		t.Fatalf("FormatToFile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Excel file was not created")
	}

	// Verify file has content
	fi, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if fi.Size() == 0 {
		t.Error("Excel file is empty")
	}
}

func TestExcelFormatter_EmptyResult(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.xlsx")

	res := &Result{
		Entity:  ast.EntityFeatures,
		Records: []Record{},
	}

	formatter := NewExcelFormatter()
	err := formatter.FormatToFile(filePath, res)
	if err != nil {
		t.Fatalf("FormatToFile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Excel file was not created for empty result")
	}
}

func TestExcelFormatter_WithFields(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "filtered.xlsx")

	res := &Result{
		Entity: ast.EntityFeatures,
		Records: []Record{
			{
				"id":     "123",
				"name":   "Test Feature",
				"status": "In Progress",
				"extra":  "should be excluded",
			},
		},
	}

	formatter := NewExcelFormatter().WithFields([]string{"id", "name"})
	err := formatter.FormatToFile(filePath, res)
	if err != nil {
		t.Fatalf("FormatToFile() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Excel file was not created")
	}
}

func TestExcelFormatter_MultiSheet(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "multi.xlsx")

	sheets := map[string]*Result{
		"Features": {
			Entity: ast.EntityFeatures,
			Records: []Record{
				{"id": "1", "name": "Feature 1"},
				{"id": "2", "name": "Feature 2"},
			},
		},
		"Ideas": {
			Entity: ast.EntityIdeas,
			Records: []Record{
				{"id": "100", "name": "Idea 1", "votes": int64(10)},
			},
		},
	}

	formatter := NewExcelFormatter()
	err := formatter.FormatMultiSheet(filePath, sheets)
	if err != nil {
		t.Fatalf("FormatMultiSheet() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Excel file was not created")
	}
}
