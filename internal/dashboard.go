package promtop

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NodeRef struct {
	Type        string // prometheus, prometheus_node, node_exporter
	SourceIndex int    // Index into sources array
	SourceName  string // Human-readable source name (hostname from URL)
	NodeName    string // Node name from GetNodes() (empty if IsSourceHeader)
	DisplayName string // Formatted for UI
}

type dashboardModel struct {
	sources      []Cache
	sourceNames  []string
	nodeRefs     []NodeRef
	selectedNode int
	selectedTab  int
	tabs         []string
	cpuData      [][]float64
	width        int
	height       int
	ready        bool
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
		selectedTab:  0,
		tabs:         []string{"CPU", "Memory", "Disk", "Network"},
		cpuData:      make([][]float64, 0),
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
		case "j", "down":
			if m.selectedNode < len(m.nodeRefs)-1 {
				m.selectedNode++
			}
		case "k", "up":
			if m.selectedNode > 0 {
				m.selectedNode--
			}
		case "h", "left":
			if m.selectedTab > 0 {
				m.selectedTab--
			}
		case "l", "right":
			if m.selectedTab < len(m.tabs)-1 {
				m.selectedTab++
			}
		case "g":
			m.selectedNode = 0
		case "G":
			m.selectedNode = len(m.nodeRefs) - 1
		case "ctrl+d":
			m.selectedNode = min(m.selectedNode+5, len(m.nodeRefs)-1)
		case "ctrl+u":
			m.selectedNode = max(m.selectedNode-5, 0)
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

		// Update CPU data for selected node (skip if source header is selected)
		if len(m.nodeRefs) > 0 && m.selectedNode < len(m.nodeRefs) {
			selectedRef := m.nodeRefs[m.selectedNode]

			// Only fetch CPU data if a node is selected (not a prometheus)
			if selectedRef.Type != "prometheus" {
				cpus := m.sources[selectedRef.SourceIndex].GetCpu(selectedRef.NodeName)

				// Initialize CPU data if needed
				if len(m.cpuData) != len(cpus) {
					m.cpuData = make([][]float64, len(cpus))
					for i := range m.cpuData {
						m.cpuData[i] = []float64{}
					}
				}

				// Append new data and trim
				maxDataPoints := max(m.width-40, 20)
				for i, c := range cpus {
					m.cpuData[i] = append(m.cpuData[i], c)
					if len(m.cpuData[i]) > maxDataPoints {
						m.cpuData[i] = m.cpuData[i][len(m.cpuData[i])-maxDataPoints:]
					}
				}

				// log.Printf("Updated CPU data: %d cores, %d data points", len(cpus), len(m.cpuData[0]))
			}
		}

		return m, tickCmd()
	}

	return m, nil
}

func (m dashboardModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Styles
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	sourceHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	// Calculate dimensions
	// Calculate minimum width based on longest node name and source name
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
	nodeListWidth += 1

	// Render node list
	var nodeList strings.Builder
	nodeList.WriteString(titleStyle.Render("Nodes") + "\n")

	lineCount := 1 // Start at 1 for title
	maxLines := m.height - 3

	for i, nodeRef := range m.nodeRefs {
		if lineCount >= maxLines {
			break
		}

		if nodeRef.Type == "prometheus" || nodeRef.Type == "node_exporter" {
			// Render prometheus and node_exporter (single indent)
			if i == m.selectedNode {
				nodeList.WriteString("  " + selectedStyle.Render("▶ "+nodeRef.DisplayName) + "\n")
			} else {
				nodeList.WriteString("    " + sourceHeaderStyle.Render(nodeRef.DisplayName) + "\n")
			}
		} else {
			// Render prometheus_node (double indent)
			if i == m.selectedNode {
				nodeList.WriteString("  " + selectedStyle.Render("▶  "+nodeRef.DisplayName) + "\n")
			} else {
				nodeList.WriteString("     " + nodeRef.DisplayName + "\n")
			}
		}
		lineCount++
	}

	nodeListBox := borderStyle.
		Width(nodeListWidth).
		Height(min(lineCount+1, m.height-2)).
		Render(strings.TrimSuffix(nodeList.String(), "\n"))

	// Render tabs
	var tabsView strings.Builder
	for i, tab := range m.tabs {
		if i == m.selectedTab {
			tabsView.WriteString(selectedStyle.Render(" [" + tab + "] "))
		} else {
			tabsView.WriteString(" " + tab + " ")
		}
	}

	// Render content based on selected tab
	var content strings.Builder
	content.WriteString(tabsView.String() + "\n\n")

	// Check if a source header is selected
	isSourceHeaderSelected := len(m.nodeRefs) > 0 && m.selectedNode < len(m.nodeRefs) && m.nodeRefs[m.selectedNode].Type == "prometheus"

	if isSourceHeaderSelected {
		// Render blank content for source headers
		content.WriteString("\n")
	} else {
		// Render normal content for nodes
		switch m.selectedTab {
		case 0: // CPU
			content.WriteString(titleStyle.Render("CPU Cores") + "\n")
			if len(m.cpuData) > 0 {
				for i, data := range m.cpuData {
					if len(data) > 0 {
						latest := data[len(data)-1]
						content.WriteString(fmt.Sprintf("Core %d: %.1f%% ", i, latest))
						content.WriteString(m.renderSparkline(data) + "\n")
					}
				}
			} else {
				content.WriteString("No CPU data available\n")
			}
		case 1: // Memory
			content.WriteString("Memory metrics coming soon...\n")
		case 2: // Disk
			content.WriteString("Disk metrics coming soon...\n")
		case 3: // Network
			content.WriteString("Network metrics coming soon...\n")
		}
	}

	contentWidth := m.width - nodeListWidth - 6
	contentBox := borderStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(content.String())

	// Combine node list and content side by side
	return lipgloss.JoinHorizontal(lipgloss.Top, nodeListBox, contentBox)
}

// renderSparkline creates a simple sparkline visualization
func (m dashboardModel) renderSparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}

	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var result strings.Builder
	maxWidth := min(len(data), 50)
	step := 1
	if len(data) > maxWidth {
		step = len(data) / maxWidth
	}

	for i := 0; i < len(data); i += step {
		val := data[i]
		idx := int(val / 100.0 * float64(len(sparkChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		result.WriteRune(sparkChars[idx])
	}

	return result.String()
}

func Dashboard(sources []Cache, sourceNames []string) {
	m := NewDashboard(sources, sourceNames)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running bubbletea program: %v", err)
	}
}
