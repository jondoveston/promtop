package main

import (
	"log"
	"os"

	promtop "github.com/jondoveston/promtop/internal"
	"github.com/spf13/viper"
)

var version = "dev"

func main() {
	log.SetOutput(os.Stderr)
	log.Printf("Starting Promtop %s", version)

	viper.SetEnvPrefix("promtop")
	if err := viper.BindEnv("prometheus_url"); err != nil {
		log.Fatalf("failed to bind prometheus_url: %v", err)
	}
	if err := viper.BindEnv("node_exporter_url"); err != nil {
		log.Fatalf("failed to bind node_exporter_url: %v", err)
	}
	viper.SetDefault("node_exporter_url", "http://localhost:9100/metrics")

	if viper.Get("prometheus_url") == "" && viper.Get("node_exporter_url") == "" {
		log.Fatalln("prometheus_url or node_exporter_url must be set")
	}

	var d promtop.Data
	if viper.GetString("prometheus_url") != "" {
		d = &promtop.PrometheusData{}
	} else if viper.GetString("node_exporter_url") != "" {
		d = &promtop.NodeExporterData{}
	}
	promtop.Dashboard(promtop.Cache{Data: d})
}
