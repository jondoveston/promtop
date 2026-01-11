package promtop

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Pane represents a bordered panel in the dashboard.
//
// Example usage:
//
//	pane := NewPane("CPU Metrics", 40, 10).
//	    SetContent("Core 0: 45%\nCore 1: 23%").
//	    SetFocused(true)
//	fmt.Println(pane.Render())
//
// Panes can be composed using layout helpers:
//
//	leftPane := NewPane("Left", 30, 20).SetContent("content")
//	rightPane := NewPane("Right", 50, 20).SetContent("content")
//	dashboard := Horizontal(leftPane, rightPane)
//
// Pane implements tea.Model so it can be used as a standalone Bubble Tea component
type Pane struct {
	title       string
	content     string
	width       int
	height      int
	borderStyle lipgloss.Style
	titleStyle  lipgloss.Style
	focused     bool
}

// NewPane creates a new pane with default styling
func NewPane(title string, width, height int) Pane {
	return Pane{
		title:  title,
		width:  width,
		height: height,
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true),
		focused: false,
	}
}

// SetTitle sets the pane title
func (p Pane) SetTitle(title string) Pane {
	p.title = title
	return p
}

// SetContent sets the pane content
func (p Pane) SetContent(content string) Pane {
	p.content = content
	return p
}

// SetSize sets the pane dimensions
func (p Pane) SetSize(width, height int) Pane {
	p.width = width
	p.height = height
	return p
}

// SetBorderStyle sets the border style
func (p Pane) SetBorderStyle(style lipgloss.Style) Pane {
	p.borderStyle = style
	return p
}

// SetTitleStyle sets the title style
func (p Pane) SetTitleStyle(style lipgloss.Style) Pane {
	p.titleStyle = style
	return p
}

// SetFocused sets the focus state
func (p Pane) SetFocused(focused bool) Pane {
	p.focused = focused
	if focused {
		p.borderStyle = p.borderStyle.BorderForeground(lipgloss.Color("170"))
	} else {
		p.borderStyle = p.borderStyle.BorderForeground(lipgloss.Color("240"))
	}
	return p
}

// Init implements tea.Model
func (p Pane) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (p Pane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return p, nil
}

// View implements tea.Model
func (p Pane) View() string {
	var b strings.Builder

	// Add title if present
	if p.title != "" {
		b.WriteString(p.titleStyle.Render(p.title) + "\n")
	}

	// Add content
	b.WriteString(p.content)

	// Apply border and dimensions
	box := p.borderStyle.
		Width(p.width).
		Height(p.height).
		Render(b.String())

	return box
}

// Render is a convenience method that calls View()
func (p Pane) Render() string {
	return p.View()
}
