package promtop

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TabSet manages multiple charts with tab navigation
type TabSet struct {
	charts      []Chart
	selectedTab int
	width       int
	height      int
}

// NewTabSet creates a new TabSet
func NewTabSet() *TabSet {
	return &TabSet{
		charts:      []Chart{},
		selectedTab: 0,
		width:       40,
		height:      10,
	}
}

// AddChart adds a chart to the tab set
func (ts *TabSet) AddChart(chart Chart) *TabSet {
	ts.charts = append(ts.charts, chart)
	return ts
}

// SetSize sets the dimensions for rendering
func (ts *TabSet) SetSize(width, height int) *TabSet {
	ts.width = width
	ts.height = height
	return ts
}

// SelectTab changes the active tab
func (ts *TabSet) SelectTab(index int) *TabSet {
	if index >= 0 && index < len(ts.charts) {
		ts.selectedTab = index
	}
	return ts
}

// NextTab moves to the next tab (wraps around)
func (ts *TabSet) NextTab() *TabSet {
	if len(ts.charts) > 0 {
		ts.selectedTab = (ts.selectedTab + 1) % len(ts.charts)
	}
	return ts
}

// PrevTab moves to the previous tab (wraps around)
func (ts *TabSet) PrevTab() *TabSet {
	if len(ts.charts) > 0 {
		ts.selectedTab = (ts.selectedTab - 1 + len(ts.charts)) % len(ts.charts)
	}
	return ts
}

// GetCharts returns all charts in the tab set
func (ts *TabSet) GetCharts() []Chart {
	return ts.charts
}

// GetSelectedTab returns the currently selected tab index
func (ts *TabSet) GetSelectedTab() int {
	return ts.selectedTab
}

// RemoveCurrentTab removes the currently selected tab
// Returns true if tab was removed, false if it was the last tab
func (ts *TabSet) RemoveCurrentTab() bool {
	if len(ts.charts) == 0 {
		return false
	}

	// Remove the current tab
	ts.charts = append(ts.charts[:ts.selectedTab], ts.charts[ts.selectedTab+1:]...)

	// Adjust selectedTab if needed
	if ts.selectedTab >= len(ts.charts) && len(ts.charts) > 0 {
		ts.selectedTab = len(ts.charts) - 1
	}

	return len(ts.charts) > 0
}

// Render renders the tab set with tabs and active chart content
func (ts *TabSet) Render() string {
	if len(ts.charts) == 0 {
		return "No charts available"
	}

	var b strings.Builder

	selectedChart := ts.charts[ts.selectedTab]

	// Show hostname of currently selected chart above tabs
	hostnameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	b.WriteString(hostnameStyle.Render(selectedChart.NodeRef.DisplayName))
	b.WriteString("\n")

	// Render tabs if more than one chart
	if len(ts.charts) > 1 {
		b.WriteString(ts.renderTabs())
		b.WriteString("\n")
	}

	// Render active chart content
	contentHeight := ts.height - 2 // Account for hostname line
	if len(ts.charts) > 1 {
		contentHeight -= 3 // Account for tab bar height
	}

	chartContent := ts.renderChartContent(selectedChart, ts.width, contentHeight)
	b.WriteString(chartContent)

	return b.String()
}

// renderTabs renders the tab navigation bar
func (ts *TabSet) renderTabs() string {
	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("235")).
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("170"))

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("236"))

	var renderedTabs []string
	for i, chart := range ts.charts {
		// Create tab label based on chart type
		var label string
		switch chart.ChartType {
		case "cpu":
			label = "CPU"
		case "memory":
			label = "Memory"
		case "disk":
			label = "Disk"
		case "network":
			label = "Network"
		default:
			label = chart.ChartType
		}

		if i == ts.selectedTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(label))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// renderChartContent renders the content of a chart
func (ts *TabSet) renderChartContent(chart Chart, width, height int) string {
	var content strings.Builder

	switch chart.ChartType {
	case "cpu":
		if len(chart.CpuData) > 0 {
			// Create table for CPU data
			rows := [][]string{}

			// Sort CPU names for consistent display
			cpuNames := make([]string, 0, len(chart.CpuData))
			for cpuName := range chart.CpuData {
				cpuNames = append(cpuNames, cpuName)
			}
			// Sort numerically by converting to int
			sort.Slice(cpuNames, func(i, j int) bool {
				numI, errI := strconv.Atoi(cpuNames[i])
				numJ, errJ := strconv.Atoi(cpuNames[j])
				if errI == nil && errJ == nil {
					return numI < numJ
				}
				return cpuNames[i] < cpuNames[j]
			})

			for _, cpuName := range cpuNames {
				data := chart.CpuData[cpuName]
				if len(data) > 0 {
					latest := data[len(data)-1]
					rows = append(rows, []string{
						fmt.Sprintf("Core %s", cpuName),
						fmt.Sprintf("%.1f%%", latest),
					})
				}
			}

			// Use WrapTable to handle wrapping when content exceeds height
			t := NewWrapTable().
				BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
				MaxHeight(height).
				Headers("Core", "Usage").
				Rows(rows...)

			content.WriteString(t.Render())
		} else {
			content.WriteString("Waiting for data...")
		}
	case "memory":
		content.WriteString("Memory metrics coming soon...")
	case "disk":
		content.WriteString("Disk metrics coming soon...")
	case "network":
		content.WriteString("Network metrics coming soon...")
	default:
		content.WriteString("Unsupported chart type")
	}

	return content.String()
}

// String is a convenience method that calls Render
func (ts *TabSet) String() string {
	return ts.Render()
}
