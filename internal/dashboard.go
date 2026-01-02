package promtop

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type dashboardModel struct {
	cache        Cache
	instances    []string
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

func NewDashboard(cache Cache) *dashboardModel {
	return &dashboardModel{
		cache:        cache,
		instances:    cache.GetInstances(),
		selectedNode: 0,
		selectedTab:  0,
		tabs:         []string{"CPU", "Memory", "Disk", "Network"},
		cpuData:      make([][]float64, 0),
	}
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
			if m.selectedNode < len(m.instances)-1 {
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
			m.selectedNode = len(m.instances) - 1
		case "ctrl+d":
			m.selectedNode = min(m.selectedNode+5, len(m.instances)-1)
		case "ctrl+u":
			m.selectedNode = max(m.selectedNode-5, 0)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tickMsg:
		// Update instances
		m.instances = m.cache.GetInstances()

		// Update CPU data for selected node
		if len(m.instances) > 0 && m.selectedNode < len(m.instances) {
			cpus := m.cache.GetCpu(m.instances[m.selectedNode])

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

	// Calculate dimensions
	nodeListWidth := 30
	if m.width > 0 {
		nodeListWidth = min(nodeListWidth, m.width/4)
	}

	// Render node list
	nodeListHeight := m.height - 4
	var nodeList strings.Builder
	nodeList.WriteString(titleStyle.Render("Nodes") + "\n")

	startIdx := max(0, m.selectedNode-nodeListHeight+2)
	endIdx := min(len(m.instances), startIdx+nodeListHeight-1)

	for i := startIdx; i < endIdx; i++ {
		if i == m.selectedNode {
			nodeList.WriteString(selectedStyle.Render("▶ "+m.instances[i]) + "\n")
		} else {
			nodeList.WriteString("  " + m.instances[i] + "\n")
		}
	}

	nodeListBox := borderStyle.
		Width(nodeListWidth).
		Height(m.height - 2).
		Render(nodeList.String())

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

func Dashboard(cache Cache) {
	m := NewDashboard(cache)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running bubbletea program: %v", err)
	}
}
