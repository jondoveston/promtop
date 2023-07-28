package promtop

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/viper"
)

func PrometheusDashboard() {
  instancesNames := GetInstances()
  maxInstanceNameLen := 0
  for _, name := range instancesNames {
    if len(name) > maxInstanceNameLen {
      maxInstanceNameLen = len(name)
    }
  }
	termWidth, termHeight := ui.TerminalDimensions()

	ls := widgets.NewList()
	ls.Title = "Nodes"
	ls.Rows = instancesNames
	ls.Border = true
	ls.TextStyle = ui.NewStyle(ui.ColorYellow)
	ls.WrapText = false
  ls.SetRect(0, 0, maxInstanceNameLen+2, termHeight)

  tabpane := widgets.NewTabPane("CPU", "Memory", "Disk", "Network")
	tabpane.SetRect(maxInstanceNameLen+2, 0, termWidth, termHeight)
	tabpane.Border = true

	cpuGrid := ui.NewGrid()
	cpuGrid.SetRect(maxInstanceNameLen+3, 2, termWidth-1, termHeight-1)

	cpu_slg := widgets.NewSparklineGroup(widgets.NewSparkline())
	cpu_slg.Title = "CPU"

	cpuGrid.Set(ui.NewRow(1.0, ui.NewCol(1.0, ui.NewRow(1.0, cpu_slg))))

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
				payload := e.Payload.(ui.Resize)
        tabpane.SetRect(maxInstanceNameLen+2, 0, payload.Width, payload.Height)
        cpuGrid.SetRect(maxInstanceNameLen+3, 2, payload.Width-1, payload.Height-1)
				ui.Clear()
        render()
			case "<Enter>":
				cpus := GetCpu(ls.Rows[ls.SelectedRow])
				sls := make([]*widgets.Sparkline, 0, len(cpus))
				for i, cpu := range cpus {
					sl := widgets.NewSparkline()
					sl.MaxVal = 100.0
					sl.Title = fmt.Sprintf("CPU %d", i)
					sl.LineColor = ui.ColorCyan
					sl.TitleStyle.Fg = ui.ColorWhite
					sl.Data = []float64{cpu}
					sls = append(sls, sl)
				}
				cpu_slg.Sparklines = sls
			}
			if previousKey == "g" {
				previousKey = ""
			} else {
				previousKey = e.ID
			}
      render()
		case <-ticker:
			ls.Rows = GetInstances()
			cpus := GetCpu(ls.Rows[ls.SelectedRow])
			for i, cpu := range cpus {
				if i < len(cpu_slg.Sparklines) {
					cpu_slg.Sparklines[i].Data = append(cpu_slg.Sparklines[i].Data, cpu)
				}
			}
      render()
		}
	}
}

func getClient() api.Client {
	client, err := api.NewClient(api.Config{
		Address: viper.GetString("prometheus_url"),
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	return client
}

func GetInstances() []string {
	client := getClient()

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, "up{job=\"node_exporter\"}", time.Now())
	if err != nil {
		log.Fatalf("Error querying Prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Fatalf("Warnings: %v\n", warnings)
	}

	instances := make([]string, 0, result.(model.Vector).Len())
	for _, val := range result.(model.Vector) {
		if val.Value == 1 {
			instances = append(instances, string(val.Metric["instance"]))
		}
	}

	return instances
}

func GetCpu(instance string) []float64 {
	client := getClient()

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, "100 - (avg by (instance,cpu) (rate(node_cpu_seconds_total{instance=\""+instance+"\",job=\"node_exporter\",mode=\"idle\"}[1m])) * 100)", time.Now())
	if err != nil {
		log.Fatalf("Error querying Prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Fatalf("Warnings: %v\n", warnings)
	}

	sort.Slice(result.(model.Vector), func(i, j int) bool {
		cpu_i, _ := strconv.Atoi(string(result.(model.Vector)[i].Metric["cpu"]))
		cpu_j, _ := strconv.Atoi(string(result.(model.Vector)[j].Metric["cpu"]))
		return cpu_i < cpu_j
	})

	cpus := make([]float64, 0, result.(model.Vector).Len())
	for _, val := range result.(model.Vector) {
		// percent, err := strconv.ParseFloat(string(val.Value), 64)
		cpus = append(cpus, float64(val.Value))
	}

	return cpus
}
