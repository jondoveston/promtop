package main

import (
	"log"

	ui "github.com/gizak/termui/v3"
	"github.com/jondoveston/promtop/internal"
	"github.com/spf13/viper"
  "github.com/Kerrigan29a/drawille-go"
)

func main() {
	viper.SetEnvPrefix("promtop")
	viper.BindEnv("prometheus_url")
	viper.BindEnv("node_exporter_url")
	viper.SetDefault("node_exporter_url", "http://localhost:9100/metrics")

	if viper.Get("prometheus_url") == "" && viper.Get("node_exporter_url") == "" {
		log.Fatalf("prometheus_url or node_exporter_url must be set")
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	if viper.GetString("prometheus_url") != "" {
		promtop.PrometheusDashboard()
		log.Printf(viper.GetString("prometheus_url"))
	} else if viper.GetString("node_exporter_url") != "" {
		promtop.NodeExporterDashboard()
	}
}
