package promtop

import (
	"fmt"
)

// ExampleDashboard demonstrates how to use Pane and layout helpers
func ExampleDashboard(width, height int) string {
	// Create individual panes
	leftPane := NewPane("Nodes", 30, height-2).
		SetContent("  server1\n  server2\n▶ server3")

	topRightPane := NewPane("CPU Usage", width-36, height/2-2).
		SetContent("Core 0: 45% ▃▄▅▆▇█▇▆\nCore 1: 23% ▂▃▃▄▅▅▄▃")

	bottomRightPane := NewPane("Memory", width-36, height/2-2).
		SetContent("Used: 8.2GB / 16GB\n████████░░░░░░░░ 51%")

	// Create a grid layout
	grid := NewGrid()
	grid.AddRow(topRightPane)
	grid.AddRow(bottomRightPane)

	// Combine left pane with right grid
	rightColumn := grid.Render()

	// For manual rendering without grid:
	// rightColumn := Vertical(topRightPane, bottomRightPane)

	return Horizontal(leftPane, NewPane("", width-36, height-2).SetContent(rightColumn))
}

// SimpleTwoColumnLayout creates a basic two-column dashboard
func SimpleTwoColumnLayout(leftContent, rightContent string, width, height int) string {
	leftWidth := width / 3
	rightWidth := width - leftWidth - 6

	left := NewPane("", leftWidth, height-2).SetContent(leftContent)
	right := NewPane("", rightWidth, height-2).SetContent(rightContent)

	return Horizontal(left, right)
}

// ThreeColumnLayout creates a three-column dashboard
func ThreeColumnLayout(left, center, right string, width, height int) string {
	colWidth := (width - 12) / 3

	leftPane := NewPane("Left", colWidth, height-2).SetContent(left)
	centerPane := NewPane("Center", colWidth, height-2).SetContent(center).SetFocused(true)
	rightPane := NewPane("Right", colWidth, height-2).SetContent(right)

	return Horizontal(leftPane, centerPane, rightPane)
}

// MetricPane creates a formatted metric display pane
func MetricPane(title string, metrics map[string]string, width, height int) Pane {
	var content string
	for key, value := range metrics {
		content += fmt.Sprintf("%s: %s\n", key, value)
	}
	return NewPane(title, width, height).SetContent(content)
}
