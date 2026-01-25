package promtop

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/lipgloss/tree"
)

type NodeRef struct {
	Type        string // prometheus, prometheus_node, node_exporter
	SourceIndex int    // Index into sources array
	SourceName  string // Human-readable source name (hostname from URL)
	NodeName    string // Node name from GetNodes() (empty if IsSourceHeader)
	DisplayName string // Formatted for UI
}

type dashboardModel struct {
	sources        []Cache
	sourceNames    []string
	nodeRefs       []NodeRef
	selectedNode   int
	selectedPane   int // Index of selected pane (TabSet)
	selectedTab    int
	tabs           []string
	cpuData        [][]float64
	activePanes    []*TabSet // Each pane can contain multiple tabbed charts
	showModal      bool      // true = modal open, false = modal closed
	modalNewPane   bool      // true = create new pane, false = add to current pane
	modalChartType string    // Chart type being added in modal
	width          int
	height         int
	ready          bool
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func NewDashboard(sources []Cache, sourceNames []string) *dashboardModel {
	m := &dashboardModel{
		sources:      sources,
		sourceNames:  sourceNames,
		selectedNode: 0,
		selectedPane: 0,
		selectedTab:  0,
		tabs:         []string{"CPU", "Memory", "Disk", "Network"},
		cpuData:      make([][]float64, 0),
		activePanes:  make([]*TabSet, 0),
	}
	m.nodeRefs = m.refreshNodes()
	return m
}

func (m dashboardModel) refreshNodes() []NodeRef {
	var nodeRefs []NodeRef

	// Iterate through each source
	for sourceIdx, source := range m.sources {
		nodes := source.GetNodes()
		sort.Strings(nodes)

		if source.GetType() == "prometheus" {
			// For Prometheus: add source header, then nodes
			nodeRefs = append(nodeRefs, NodeRef{
				Type:        "prometheus",
				SourceIndex: sourceIdx,
				SourceName:  m.sourceNames[sourceIdx],
				NodeName:    "",
				DisplayName: m.sourceNames[sourceIdx],
			})

			for _, nodeName := range nodes {
				nodeRefs = append(nodeRefs, NodeRef{
					Type:        "prometheus_node",
					SourceIndex: sourceIdx,
					SourceName:  m.sourceNames[sourceIdx],
					NodeName:    nodeName,
					DisplayName: nodeName,
				})
			}
		} else {
			// For node_exporter: single line per node (no header)
			for _, nodeName := range nodes {
				nodeRefs = append(nodeRefs, NodeRef{
					Type:        "node_exporter",
					SourceIndex: sourceIdx,
					SourceName:  m.sourceNames[sourceIdx],
					NodeName:    nodeName,
					DisplayName: nodeName,
				})
			}
		}
	}

	return nodeRefs
}

func (m dashboardModel) Init() tea.Cmd {
	return tickCmd()
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			// Close modal if open
			if m.showModal {
				m.showModal = false
				m.modalNewPane = false
				m.modalChartType = ""
			}
		case "n":
			if m.showModal {
				// Add Network chart for selected node (from modal)
				m = m.addChart("network")
				m.showModal = false
				m.modalNewPane = false
			} else if len(m.activePanes) < 9 {
				// Open modal to add new pane
				m.showModal = true
				m.modalNewPane = true
			}
		case "a":
			if m.showModal {
				// Add all chart types for selected node (from modal)
				m = m.addChart("cpu")
				m = m.addChart("memory")
				m = m.addChart("disk")
				m = m.addChart("network")
				m.showModal = false
				m.modalNewPane = false
			} else {
				// Open modal to add charts to current pane
				m.showModal = true
				m.modalNewPane = false
			}
		case "j", "down":
			if m.showModal {
				// Navigate nodes in modal
				if m.selectedNode < len(m.nodeRefs)-1 {
					m.selectedNode++
				}
			} else {
				// Navigate panes in grid
				columns := m.getGridColumns()
				if m.selectedPane+columns < len(m.activePanes) {
					m.selectedPane += columns
				}
			}
		case "k", "up":
			if m.showModal {
				// Navigate nodes in modal
				if m.selectedNode > 0 {
					m.selectedNode--
				}
			} else {
				// Navigate panes in grid
				columns := m.getGridColumns()
				if m.selectedPane-columns >= 0 {
					m.selectedPane -= columns
				}
			}
		case "h", "left":
			if !m.showModal {
				// Navigate panes in grid
				if m.selectedPane > 0 {
					m.selectedPane--
				}
			}
		case "l", "right":
			if !m.showModal {
				// Navigate panes in grid
				if m.selectedPane < len(m.activePanes)-1 {
					m.selectedPane++
				}
			}
		case "[":
			// Previous tab in current pane
			if !m.showModal && len(m.activePanes) > 0 && m.selectedPane < len(m.activePanes) {
				m.activePanes[m.selectedPane].PrevTab()
			}
		case "]":
			// Next tab in current pane
			if !m.showModal && len(m.activePanes) > 0 && m.selectedPane < len(m.activePanes) {
				m.activePanes[m.selectedPane].NextTab()
			}
		case "g":
			if m.showModal {
				m.selectedNode = 0
			} else {
				m.selectedPane = 0
			}
		case "G":
			if m.showModal {
				m.selectedNode = len(m.nodeRefs) - 1
			} else {
				m.selectedPane = len(m.activePanes) - 1
			}
		case "ctrl+d":
			if m.showModal {
				m.selectedNode = min(m.selectedNode+5, len(m.nodeRefs)-1)
			}
		case "ctrl+u":
			if m.showModal {
				m.selectedNode = max(m.selectedNode-5, 0)
			}
		case "c":
			// Add CPU chart for selected node (from modal)
			if m.showModal {
				m = m.addChart("cpu")
				m.showModal = false
				m.modalNewPane = false
			}
		case "m":
			// Add Memory chart for selected node (from modal)
			if m.showModal {
				m = m.addChart("memory")
				m.showModal = false
				m.modalNewPane = false
			}
		case "s":
			// Add Storage/Disk chart for selected node (from modal)
			if m.showModal {
				m = m.addChart("disk")
				m.showModal = false
				m.modalNewPane = false
			}
		case "x":
			// Remove current tab, or pane if only one tab left
			if !m.showModal && len(m.activePanes) > 0 && m.selectedPane < len(m.activePanes) {
				selectedPane := m.activePanes[m.selectedPane]
				charts := selectedPane.GetCharts()

				if len(charts) > 1 {
					// Remove current tab from pane
					selectedPane.RemoveCurrentTab()
				} else {
					// Only one tab left, remove entire pane
					m.activePanes = append(m.activePanes[:m.selectedPane], m.activePanes[m.selectedPane+1:]...)
					// Bounds check selectedPane
					if m.selectedPane >= len(m.activePanes) {
						m.selectedPane = max(0, len(m.activePanes)-1)
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tickMsg:
		// Update nodes
		m.nodeRefs = m.refreshNodes()

		// Bounds check after refresh
		if m.selectedNode >= len(m.nodeRefs) {
			m.selectedNode = max(0, len(m.nodeRefs)-1)
		}

		// Update CPU data for all charts in all panes
		maxDataPoints := max(m.width-40, 20)
		for _, pane := range m.activePanes {
			charts := pane.GetCharts()
			for i := range charts {
				chart := &charts[i]
				if chart.ChartType == "cpu" {
					cpus := m.sources[chart.NodeRef.SourceIndex].GetCpu(chart.NodeRef.NodeName)

					// Initialize CPU data if needed
					if len(chart.CpuData) != len(cpus) {
						chart.CpuData = make([][]float64, len(cpus))
						for j := range chart.CpuData {
							chart.CpuData[j] = []float64{}
						}
					}

					// Append new data and trim
					for j, c := range cpus {
						chart.CpuData[j] = append(chart.CpuData[j], c)
						if len(chart.CpuData[j]) > maxDataPoints {
							chart.CpuData[j] = chart.CpuData[j][len(chart.CpuData[j])-maxDataPoints:]
						}
					}
				}
			}
		}

		return m, tickCmd()
	}

	return m, nil
}

// getGridColumns returns the number of columns in the current grid layout
func (m dashboardModel) getGridColumns() int {
	if len(m.activePanes) == 1 {
		return 1
	} else if len(m.activePanes) <= 4 {
		return 2
	} else {
		return 3
	}
}

// addChart adds a chart of the specified type for the currently selected node
// Adds to the currently selected pane if one exists, otherwise creates a new pane
// This allows mixing charts from different sources in the same pane
func (m dashboardModel) addChart(chartType string) dashboardModel {
	if m.selectedNode >= len(m.nodeRefs) {
		return m
	}

	selectedRef := m.nodeRefs[m.selectedNode]

	// Don't add if it's a prometheus header
	if selectedRef.Type == "prometheus" {
		return m
	}

	// Create the new chart
	newChart := Chart{
		NodeRef:   selectedRef,
		ChartType: chartType,
		CpuData:   make([][]float64, 0),
	}

	// If modalNewPane is true, always create a new pane
	if m.modalNewPane || len(m.activePanes) == 0 {
		// Create new pane if we haven't reached the limit
		if len(m.activePanes) < 9 {
			newPane := NewTabSet().AddChart(newChart)
			m.activePanes = append(m.activePanes, newPane)
			m.selectedPane = len(m.activePanes) - 1 // Auto-select the new pane
		}
	} else if m.selectedPane < len(m.activePanes) {
		// Add to currently selected pane
		selectedPane := m.activePanes[m.selectedPane]

		// Check if this exact chart already exists (same node + same chart type)
		charts := selectedPane.GetCharts()
		for _, chart := range charts {
			if chart.NodeRef.SourceIndex == selectedRef.SourceIndex &&
				chart.NodeRef.NodeName == selectedRef.NodeName &&
				chart.ChartType == chartType {
				// Exact duplicate, don't add
				return m
			}
		}

		// Add chart to selected pane
		selectedPane.AddChart(newChart)
	}

	return m
}

func (m dashboardModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var baseView string

	// Render active panes or show instructions
	if len(m.activePanes) == 0 {
		// Show instructions when no charts are active
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(2, 4)

		baseView = helpStyle.Render(
			"No charts active\n\n" +
				"Press 'n' to add a new pane\n" +
				"Press 'hjkl' or arrow keys to navigate\n\n" +
				"Max 9 panes",
		)
	} else {
		// Determine grid layout based on number of panes
		var columns int
		if len(m.activePanes) == 1 {
			// 1 pane: use full space
			columns = 1
		} else if len(m.activePanes) <= 4 {
			// 2-4 panes: 2x2 grid
			columns = 2
		} else {
			// 5-9 panes: 3x3 grid
			columns = 3
		}

		// Calculate pane dimensions based on columns - use full screen width
		// Account for help bar at bottom (2 lines)
		availableHeight := m.height - 2
		paneWidth := m.width/columns - 2
		paneHeight := availableHeight/columns - 2

		// Create panes for each TabSet
		var renderedPanes []Pane
		for i, tabSet := range m.activePanes {
			// Set TabSet dimensions to fit inside pane content area
			// Account for: top border (1) + bottom border (1)
			contentHeight := paneHeight - 2
			tabSet.SetSize(paneWidth, contentHeight)

			// Render the TabSet content (includes hostname above tabs)
			content := tabSet.Render()

			// Create pane without title (TabSet shows hostname)
			pane := NewPane("", paneWidth, paneHeight).SetContent(content)

			// Highlight selected pane
			if i == m.selectedPane {
				pane = pane.SetFocused(true)
			}
			renderedPanes = append(renderedPanes, pane)
		}

		// Wrap panes in a dynamic grid
		panesView := Wrap(columns, renderedPanes...)

		// Add status bar with help text
		helpBar := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Width(m.width).
			Align(lipgloss.Center).
			Render("n=New Pane  a=Add to Pane  []=Switch Tabs  x=Remove  hjkl/arrows=Navigate  q=Quit")

		baseView = panesView + "\n" + helpBar
	}

	// If modal is open, render it over the base view
	if m.showModal {
		return m.renderModal(baseView)
	}

	return baseView
}

// renderModal renders the modal dialog for adding a new chart
func (m dashboardModel) renderModal(baseView string) string {
	// Calculate modal dimensions (centered, 60% of screen)
	modalWidth := int(float64(m.width) * 0.6)
	modalHeight := int(float64(m.height) * 0.6)

	// Build node list content
	nodeListContent := m.renderNodeList()

	// Create modal content with appropriate title based on mode
	var modalTitle string
	if m.modalNewPane {
		modalTitle = "Select Node - Creating new pane"
	} else {
		modalTitle = "Select Node - Adding to current pane"
	}
	modalPane := NewPane(modalTitle, modalWidth, modalHeight).
		SetContent(nodeListContent).
		SetFocused(true)

	// Create help text for modal
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("c=CPU  m=Memory  s=Storage  n=Network  a=All Charts  ESC=Cancel")

	modalContent := modalPane.Render() + "\n" + helpText

	// Overlay modal on base view with semi-transparent background
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modalContent,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}

// calculateNodeListWidth calculates the minimum width for the node list pane
func (m dashboardModel) calculateNodeListWidth() int {
	nodeListWidth := 1 // minimum width
	for _, nodeRef := range m.nodeRefs {
		// All items: "  " + "▶ " + name (when selected) or "    " + name (when not)
		// Maximum width is "  ▶ " + longest name
		itemLen := len(nodeRef.DisplayName) + 5 // "  " + "▶ " + " " + 1 for padding
		if itemLen > nodeListWidth {
			nodeListWidth = itemLen
		}
	}
	// Add border padding
	return nodeListWidth + 1
}

// renderNodeList builds the node list content string using lipgloss tree
func (m dashboardModel) renderNodeList() string {
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	sourceHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	normalStyle := lipgloss.NewStyle()

	// Build tree structure
	var trees []string

	i := 0
	for i < len(m.nodeRefs) {
		nodeRef := m.nodeRefs[i]

		if nodeRef.Type == "prometheus" {
			// Create tree for prometheus source with child nodes
			var rootLabel string
			if i == m.selectedNode {
				rootLabel = selectedStyle.Render("▶ " + nodeRef.DisplayName)
			} else {
				rootLabel = sourceHeaderStyle.Render(nodeRef.DisplayName)
			}

			t := tree.New().Root(rootLabel)

			// Add child nodes
			i++
			for i < len(m.nodeRefs) && m.nodeRefs[i].Type == "prometheus_node" && m.nodeRefs[i].SourceIndex == nodeRef.SourceIndex {
				childRef := m.nodeRefs[i]
				var childLabel string
				if i == m.selectedNode {
					childLabel = selectedStyle.Render("▶ " + childRef.DisplayName)
				} else {
					childLabel = normalStyle.Render(childRef.DisplayName)
				}
				t = t.Child(childLabel)
				i++
			}

			trees = append(trees, t.String())

		} else if nodeRef.Type == "node_exporter" {
			// Create simple tree for node_exporter (no children)
			var label string
			if i == m.selectedNode {
				label = selectedStyle.Render("▶ " + nodeRef.DisplayName)
			} else {
				label = sourceHeaderStyle.Render(nodeRef.DisplayName)
			}
			t := tree.New().Root(label)
			trees = append(trees, t.String())
			i++
		} else {
			// Shouldn't happen, but skip if orphaned prometheus_node
			i++
		}
	}

	return strings.Join(trees, "\n")
}

// renderTabs builds the tab navigation string with lipgloss styling
func (m dashboardModel) renderTabs() string {
	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("235")).
		Bold(true).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("170"))

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("236"))

	var renderedTabs []string
	for i, tab := range m.tabs {
		if i == m.selectedTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(tab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

// renderCPUMetrics renders CPU metrics content
func (m dashboardModel) renderCPUMetrics() string {
	// Check if a source header is selected
	isSourceHeaderSelected := len(m.nodeRefs) > 0 && m.selectedNode < len(m.nodeRefs) && m.nodeRefs[m.selectedNode].Type == "prometheus"

	if isSourceHeaderSelected {
		return "Select a node to view metrics"
	}

	if len(m.cpuData) > 0 {
		// Create table for CPU data
		rows := [][]string{}
		for i, data := range m.cpuData {
			if len(data) > 0 {
				latest := data[len(data)-1]
				rows = append(rows, []string{
					fmt.Sprintf("Core %d", i),
					fmt.Sprintf("%.1f%%", latest),
				})
			}
		}

		t := table.New().
			Border(lipgloss.NormalBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
			Headers("Core", "Usage").
			Rows(rows...)

		return t.String()
	}

	return "No CPU data available"
}

// renderMemoryMetrics renders memory metrics content
func (m dashboardModel) renderMemoryMetrics() string {
	// Check if a source header is selected
	isSourceHeaderSelected := len(m.nodeRefs) > 0 && m.selectedNode < len(m.nodeRefs) && m.nodeRefs[m.selectedNode].Type == "prometheus"

	if isSourceHeaderSelected {
		return "Select a node to view metrics"
	}

	return "Memory metrics coming soon..."
}

// renderDiskMetrics renders disk metrics content
func (m dashboardModel) renderDiskMetrics() string {
	// Check if a source header is selected
	isSourceHeaderSelected := len(m.nodeRefs) > 0 && m.selectedNode < len(m.nodeRefs) && m.nodeRefs[m.selectedNode].Type == "prometheus"

	if isSourceHeaderSelected {
		return "Select a node to view metrics"
	}

	return "Disk metrics coming soon..."
}

// renderNetworkMetrics renders network metrics content
func (m dashboardModel) renderNetworkMetrics() string {
	// Check if a source header is selected
	isSourceHeaderSelected := len(m.nodeRefs) > 0 && m.selectedNode < len(m.nodeRefs) && m.nodeRefs[m.selectedNode].Type == "prometheus"

	if isSourceHeaderSelected {
		return "Select a node to view metrics"
	}

	return "Network metrics coming soon..."
}

func Dashboard(sources []Cache, sourceNames []string) {
	m := NewDashboard(sources, sourceNames)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running bubbletea program: %v", err)
	}
}
