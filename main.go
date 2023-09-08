package main

import (
	"log"
	"os"

	ui "github.com/gizak/termui/v3"
	promtop "github.com/jondoveston/promtop/internal"
	"github.com/spf13/viper"
)

func main() {

	logfile, err := os.OpenFile("promtop.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logfile.Close()

	log.SetOutput(logfile)
	log.Println("Starting Promtop")

	viper.SetEnvPrefix("promtop")
	_ = viper.BindEnv("prometheus_url")
	_ = viper.BindEnv("node_exporter_url")
	viper.SetDefault("node_exporter_url", "http://localhost:9100/metrics")

	if viper.Get("prometheus_url") == "" && viper.Get("node_exporter_url") == "" {
		log.Fatalln("prometheus_url or node_exporter_url must be set")
	}

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	var d promtop.Data
	if viper.GetString("prometheus_url") != "" {
		d = &promtop.PrometheusData{}
	} else if viper.GetString("node_exporter_url") != "" {
		d = &promtop.NodeExporterData{}
	}
	promtop.Dashboard(promtop.Cache{Data: d})
}
