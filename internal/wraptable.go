package promtop

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// WrapTable wraps lipgloss table to support height-based wrapping
// When data exceeds maxHeight, it creates multiple tables side-by-side
type WrapTable struct {
	headers   []string
	rows      [][]string
	maxHeight int
	maxWidth  int
	border    lipgloss.Border
	borderStyle lipgloss.Style
}

// NewWrapTable creates a new wrap table
func NewWrapTable() *WrapTable {
	return &WrapTable{
		border: lipgloss.NormalBorder(),
		borderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	}
}

// Headers sets the table headers
func (wt *WrapTable) Headers(headers ...string) *WrapTable {
	wt.headers = headers
	return wt
}

// Rows sets the table rows
func (wt *WrapTable) Rows(rows ...[]string) *WrapTable {
	wt.rows = rows
	return wt
}

// MaxHeight sets the maximum height constraint
func (wt *WrapTable) MaxHeight(height int) *WrapTable {
	wt.maxHeight = height
	return wt
}

// MaxWidth sets the maximum width constraint
func (wt *WrapTable) MaxWidth(width int) *WrapTable {
	wt.maxWidth = width
	return wt
}

// Border sets the table border style
func (wt *WrapTable) Border(border lipgloss.Border) *WrapTable {
	wt.border = border
	return wt
}

// BorderStyle sets the border styling
func (wt *WrapTable) BorderStyle(style lipgloss.Style) *WrapTable {
	wt.borderStyle = style
	return wt
}

// Render renders the table with wrapping if needed
func (wt *WrapTable) Render() string {
	if len(wt.rows) == 0 {
		return ""
	}

	// Calculate rows per table based on maxHeight
	// Account for header (1 line) + borders (top + bottom + header separator = 3)
	rowsPerTable := wt.maxHeight - 4
	if rowsPerTable < 1 {
		rowsPerTable = 1
	}

	// If all rows fit in one table, render normally
	if len(wt.rows) <= rowsPerTable {
		t := table.New().
			Border(wt.border).
			BorderStyle(wt.borderStyle).
			Headers(wt.headers...).
			Rows(wt.rows...)

		if wt.maxWidth > 0 {
			t = t.Width(wt.maxWidth)
		}

		return t.String()
	}

	// Split rows into multiple tables
	var tables []string
	for i := 0; i < len(wt.rows); i += rowsPerTable {
		end := i + rowsPerTable
		if end > len(wt.rows) {
			end = len(wt.rows)
		}

		chunk := wt.rows[i:end]

		t := table.New().
			Border(wt.border).
			BorderStyle(wt.borderStyle).
			Headers(wt.headers...).
			Rows(chunk...)

		tables = append(tables, t.String())
	}

	// Join tables horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, tables...)
}

// String is a convenience method that calls Render
func (wt *WrapTable) String() string {
	return wt.Render()
}
