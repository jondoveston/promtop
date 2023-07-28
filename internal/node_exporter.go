package promtop

import (
  // "fmt"
  "log"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/spf13/viper"
	"github.com/prometheus/common/expfmt"
)

func NodeExporterDashboard() {
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	ui.Render(grid)

	previousKey := ""
	uiEvents := ui.PollEvents()
	ticker := time.NewTicker(time.Second).C
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
				ui.Render(grid)
			}
			if previousKey == "g" {
				previousKey = ""
			} else {
				previousKey = e.ID
			}
		case <-ticker:
			ui.Render(grid)
		}
	}
}

func getLoad() []float64 {
  exporterURL := viper.GetString("prometheus_url")

  client := http.Client{
		Timeout: 5 * time.Second,
	}
  resp, err := client.Get(exporterURL)
	if err != nil {
		log.Fatalln("Error querying node exporter:", err)
	}
	defer resp.Body.Close()

  body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Failed to read response body:", err)
	}
  parser := expfmt.TextParser{}
  _, err = parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		log.Fatalln("Failed to parse metrics:", err)
	}
  return nil
}
