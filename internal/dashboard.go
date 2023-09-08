package promtop

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"log"
	"time"
)

func Dashboard(d Cache) {
	termWidth, termHeight := ui.TerminalDimensions()

	ls := widgets.NewList()
	ls.Title = "Nodes"
	ls.Rows = d.GetInstances()
	ls.Border = true
	ls.TextStyle = ui.NewStyle(ui.ColorYellow)
	ls.WrapText = false
	ls.SetRect(0, 0, max(d.MaxInstanceNameLen()+2, 9), min(d.NumberOfInstances()+2, termHeight))

	tabpane := widgets.NewTabPane("CPU", "Memory", "Disk", "Network")
	tabpane.SetRect(ls.GetRect().Max.X, 0, termWidth, termHeight)
	tabpane.Border = true

	cpuGrid := ui.NewGrid()
	cpuGrid.SetRect(tabpane.Inner.Min.X, tabpane.Inner.Min.Y+1, tabpane.Inner.Max.X, tabpane.Inner.Max.Y)

	p0 := widgets.NewPlot()
	p0.Title = "Cores"
	p0.Data = make([][]float64, 0)
	p0.DrawDirection = widgets.DrawLeft
	p0.AxesColor = ui.ColorWhite
	p0.LineColors[0] = ui.ColorGreen

	cpuGrid.Set(ui.NewRow(0.25, p0))

	render := func() {
		ui.Render(ls, tabpane)
		switch tabpane.ActiveTabIndex {
		case 0:
			ui.Render(cpuGrid)
		}
	}

	render()

	previousKey := ""
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "j", "<Down>":
				ls.ScrollDown()
			case "k", "<Up>":
				ls.ScrollUp()
			case "h", "<Left>":
				tabpane.FocusLeft()
			case "l", "<Right>":
				tabpane.FocusRight()
			case "<C-d>":
				ls.ScrollHalfPageDown()
			case "<C-u>":
				ls.ScrollHalfPageUp()
			case "<C-f>":
				ls.ScrollPageDown()
			case "<C-b>":
				ls.ScrollPageUp()
			case "g":
				if previousKey == "g" {
					ls.ScrollTop()
				}
			case "<Home>":
				ls.ScrollTop()
			case "G", "<End>":
				ls.ScrollBottom()
			case "<Resize>":
				// payload := e.Payload.(ui.Resize)
				// tabpane.SetRect(maxInstanceNameLen+2, 0, payload.Width, payload.Height)
				// cpuGrid.SetRect(maxInstanceNameLen+3, 2, payload.Width-1, payload.Height-1)
				ui.Clear()
				render()
			}
			if previousKey == "g" {
				previousKey = ""
			} else {
				previousKey = e.ID
			}
			render()
		case <-ticker:
			ls.Rows = d.GetInstances()
			cpus := d.GetCpu(ls.Rows[ls.SelectedRow])
			if len(p0.Data) != len(cpus) {
				p0.Data = make([][]float64, len(cpus))
				for i := range p0.Data {
					p0.Data[i] = []float64{0.0}
				}
			}
			for i, c := range cpus {
				p0.Data[i] = append(p0.Data[i], c)
        p0.Data[i] = p0.Data[i][max(0, len(p0.Data[i]) - p0.Inner.Dx() + 5):]
			}
			if len(p0.Data) > 0 {
				log.Println("Inner", p0.Inner.Dx(), "Data", len(p0.Data[0]))
			}
			render()
		}
	}
}
