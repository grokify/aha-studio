package result

import (
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExcelFormatter formats query results as Excel files.
type ExcelFormatter struct {
	fields []string // specific fields to output (nil = all)
}

// NewExcelFormatter creates a new Excel formatter.
func NewExcelFormatter() *ExcelFormatter {
	return &ExcelFormatter{}
}

// WithFields sets specific fields to output.
func (f *ExcelFormatter) WithFields(fields []string) *ExcelFormatter {
	f.fields = fields
	return f
}

// FormatToFile writes the result to an Excel file.
//
//nolint:dupl // Similar row-writing logic but different file lifecycle
func (f *ExcelFormatter) FormatToFile(path string, result *Result) error {
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()

	// Use entity name as sheet name, default to "Sheet1"
	sheetName := "Sheet1"
	if result.Entity != "" {
		sheetName = string(result.Entity)
		// Excel sheet names have max 31 chars
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}
	}

	// Rename default sheet
	if err := file.SetSheetName("Sheet1", sheetName); err != nil {
		return fmt.Errorf("setting sheet name: %w", err)
	}

	if result.IsEmpty() {
		// Write "No results" message
		if err := file.SetCellValue(sheetName, "A1", "No results found."); err != nil {
			return fmt.Errorf("setting cell value: %w", err)
		}
		return file.SaveAs(path)
	}

	// Get columns
	columns := f.getColumns(result)

	// Create header style
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("creating header style: %w", err)
	}

	// Write header row
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := file.SetCellValue(sheetName, cell, col); err != nil {
			return fmt.Errorf("setting header cell value: %w", err)
		}
		if err := file.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("setting header cell style: %w", err)
		}
	}

	// Create date style
	dateStyle, err := file.NewStyle(&excelize.Style{
		NumFmt: 22, // m/d/yy h:mm
	})
	if err != nil {
		return fmt.Errorf("creating date style: %w", err)
	}

	// Write data rows
	for rowIdx, rec := range result.Records {
		row := rowIdx + 2 // Excel rows are 1-indexed, plus header row
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			value := rec.Get(col)

			// Handle different types
			switch v := value.(type) {
			case time.Time:
				if err := file.SetCellValue(sheetName, cell, v); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
				if err := file.SetCellStyle(sheetName, cell, cell, dateStyle); err != nil {
					return fmt.Errorf("setting cell style: %w", err)
				}
			case bool:
				cellVal := "No"
				if v {
					cellVal = "Yes"
				}
				if err := file.SetCellValue(sheetName, cell, cellVal); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
			case nil:
				if err := file.SetCellValue(sheetName, cell, ""); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
			default:
				if err := file.SetCellValue(sheetName, cell, v); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
			}
		}
	}

	// Auto-fit column widths (approximate)
	for i, col := range columns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		// Set width based on header length with minimum
		width := float64(len(col)) + 2
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		if err := file.SetColWidth(sheetName, colName, colName, width); err != nil {
			return fmt.Errorf("setting column width: %w", err)
		}
	}

	// Freeze header row
	if err := file.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return fmt.Errorf("setting panes: %w", err)
	}

	// Add record count in footer area
	lastRow := len(result.Records) + 3
	countCell, _ := excelize.CoordinatesToCellName(1, lastRow)
	if err := file.SetCellValue(sheetName, countCell, fmt.Sprintf("%d record(s)", result.Count())); err != nil {
		return fmt.Errorf("setting count cell: %w", err)
	}

	return file.SaveAs(path)
}

// FormatMultiSheet writes multiple results to an Excel file with multiple sheets.
func (f *ExcelFormatter) FormatMultiSheet(path string, sheets map[string]*Result) error {
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()

	first := true
	for name, result := range sheets {
		sheetName := name
		if len(sheetName) > 31 {
			sheetName = sheetName[:31]
		}

		if first {
			// Rename default sheet
			if err := file.SetSheetName("Sheet1", sheetName); err != nil {
				return fmt.Errorf("setting sheet name: %w", err)
			}
			first = false
		} else {
			// Create new sheet
			_, err := file.NewSheet(sheetName)
			if err != nil {
				return fmt.Errorf("creating sheet %s: %w", sheetName, err)
			}
		}

		if err := f.writeSheet(file, sheetName, result); err != nil {
			return fmt.Errorf("writing sheet %s: %w", sheetName, err)
		}
	}

	return file.SaveAs(path)
}

// writeSheet writes a result to a specific sheet.
//
//nolint:dupl // Similar row-writing logic but different file lifecycle
func (f *ExcelFormatter) writeSheet(file *excelize.File, sheetName string, result *Result) error {
	if result.IsEmpty() {
		return file.SetCellValue(sheetName, "A1", "No results found.")
	}

	columns := f.getColumns(result)

	// Create header style
	headerStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("creating header style: %w", err)
	}

	// Write header row
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := file.SetCellValue(sheetName, cell, col); err != nil {
			return fmt.Errorf("setting header cell value: %w", err)
		}
		if err := file.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("setting header cell style: %w", err)
		}
	}

	// Create date style
	dateStyle, err := file.NewStyle(&excelize.Style{
		NumFmt: 22,
	})
	if err != nil {
		return fmt.Errorf("creating date style: %w", err)
	}

	// Write data rows
	for rowIdx, rec := range result.Records {
		row := rowIdx + 2
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			value := rec.Get(col)

			switch v := value.(type) {
			case time.Time:
				if err := file.SetCellValue(sheetName, cell, v); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
				if err := file.SetCellStyle(sheetName, cell, cell, dateStyle); err != nil {
					return fmt.Errorf("setting cell style: %w", err)
				}
			case bool:
				cellVal := "No"
				if v {
					cellVal = "Yes"
				}
				if err := file.SetCellValue(sheetName, cell, cellVal); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
			case nil:
				if err := file.SetCellValue(sheetName, cell, ""); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
			default:
				if err := file.SetCellValue(sheetName, cell, v); err != nil {
					return fmt.Errorf("setting cell value: %w", err)
				}
			}
		}
	}

	// Auto-fit column widths
	for i, col := range columns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		width := float64(len(col)) + 2
		if width < 10 {
			width = 10
		}
		if width > 50 {
			width = 50
		}
		if err := file.SetColWidth(sheetName, colName, colName, width); err != nil {
			return fmt.Errorf("setting column width: %w", err)
		}
	}

	// Freeze header row
	if err := file.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return fmt.Errorf("setting panes: %w", err)
	}

	return nil
}

// getColumns returns the columns to display.
func (f *ExcelFormatter) getColumns(result *Result) []string {
	if f.fields != nil {
		return f.fields
	}

	if result.IsEmpty() {
		return nil
	}

	// Use same ordering as the main formatter
	commonOrder := []string{
		"id", "reference_num", "name", "status", "votes",
		"assigned_to", "tag", "created_at", "updated_at", "url",
	}

	columns := result.First().Keys()
	for i := 0; i < len(columns); i++ {
		for j := i + 1; j < len(columns); j++ {
			iOrder := indexOf(commonOrder, columns[i])
			jOrder := indexOf(commonOrder, columns[j])
			swap := false
			if iOrder == -1 && jOrder == -1 {
				swap = columns[i] > columns[j]
			} else if iOrder == -1 {
				swap = true
			} else if jOrder == -1 {
				swap = false
			} else {
				swap = iOrder > jOrder
			}
			if swap {
				columns[i], columns[j] = columns[j], columns[i]
			}
		}
	}

	return columns
}
