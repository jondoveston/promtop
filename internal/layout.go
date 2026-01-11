package promtop

import (
	"github.com/charmbracelet/lipgloss"
)

// DashboardLayout handles positioning and rendering of multiple panes
type DashboardLayout struct {
	width  int
	height int
	panes  []Pane
}

// NewDashboardLayout creates a new dashboard layout
func NewDashboardLayout(width, height int) *DashboardLayout {
	return &DashboardLayout{
		width:  width,
		height: height,
		panes:  make([]Pane, 0),
	}
}

// AddPane adds a pane to the layout
func (d *DashboardLayout) AddPane(pane Pane) {
	d.panes = append(d.panes, pane)
}

// SetSize updates the layout dimensions
func (d *DashboardLayout) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Horizontal renders panes side by side
func Horizontal(panes ...Pane) string {
	if len(panes) == 0 {
		return ""
	}

	views := make([]string, len(panes))
	for i, pane := range panes {
		views[i] = pane.Render()
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
}

// Vertical renders panes stacked vertically
func Vertical(panes ...Pane) string {
	if len(panes) == 0 {
		return ""
	}

	views := make([]string, len(panes))
	for i, pane := range panes {
		views[i] = pane.Render()
	}

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

// Grid renders panes in a 2D grid layout
type GridLayout struct {
	rows [][]Pane
}

// NewGrid creates a new grid layout
func NewGrid() *GridLayout {
	return &GridLayout{
		rows: make([][]Pane, 0),
	}
}

// AddRow adds a row of panes to the grid
func (g *GridLayout) AddRow(panes ...Pane) {
	g.rows = append(g.rows, panes)
}

// Render renders the grid layout
func (g *GridLayout) Render() string {
	if len(g.rows) == 0 {
		return ""
	}

	rowViews := make([]string, len(g.rows))
	for i, row := range g.rows {
		rowViews[i] = Horizontal(row...)
	}

	return lipgloss.JoinVertical(lipgloss.Left, rowViews...)
}
