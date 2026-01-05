package main

import (
	"fmt"
	"log"
	"os"

	promtop "github.com/jondoveston/promtop/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "promtop [prometheus-url]",
	Short: "Terminal-based dashboard for Prometheus/node_exporter metrics",
	Long: `promtop displays real-time CPU, memory, disk, and network metrics
from Prometheus or node_exporter in an interactive terminal interface.

Examples:
  promtop http://prometheus.lan:9090
  promtop --prometheus-url http://prometheus.lan:9090
  promtop --node-exporter-url http://localhost:9100/metrics
  PROMTOP_PROMETHEUS_URL=http://prometheus.lan:9090 promtop`,
	Args: cobra.MaximumNArgs(1),
	Run:  run,
}

func init() {
	// Define flags
	rootCmd.Flags().String("prometheus-url", "", "Prometheus server URL")
	rootCmd.Flags().String("node-exporter-url", "", "node_exporter metrics endpoint URL")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	// Bind flags to Viper keys (note: dashes in flags become underscores in viper)
	viper.BindPFlag("prometheus_url", rootCmd.Flags().Lookup("prometheus-url"))
	viper.BindPFlag("node_exporter_url", rootCmd.Flags().Lookup("node-exporter-url"))

	// Configure Viper for environment variables
	viper.SetEnvPrefix("promtop")
	viper.AutomaticEnv()

	// Explicitly bind environment variables (ensures they take precedence)
	if err := viper.BindEnv("prometheus_url"); err != nil {
		log.Fatalf("failed to bind prometheus_url: %v", err)
	}
	if err := viper.BindEnv("node_exporter_url"); err != nil {
		log.Fatalf("failed to bind node_exporter_url: %v", err)
	}

	// Set defaults
	viper.SetDefault("node_exporter_url", "http://localhost:9100/metrics")
}

func run(cmd *cobra.Command, args []string) {
	// Handle --version flag first
	versionFlag, _ := cmd.Flags().GetBool("version")
	if versionFlag {
		fmt.Printf("promtop version %s\n", version)
		return
	}

	// Set up logging
	log.SetOutput(os.Stderr)
	log.Printf("Starting Promtop %s", version)

	// Environment variables take precedence - explicitly override flags if env vars are set
	if envPrometheusURL := os.Getenv("PROMTOP_PROMETHEUS_URL"); envPrometheusURL != "" {
		viper.Set("prometheus_url", envPrometheusURL)
	}
	if envNodeExporterURL := os.Getenv("PROMTOP_NODE_EXPORTER_URL"); envNodeExporterURL != "" {
		viper.Set("node_exporter_url", envNodeExporterURL)
	}

	// Handle positional argument (only if prometheus_url not already set by env var or flag)
	if len(args) == 1 {
		// Only use positional arg if not already set
		if viper.GetString("prometheus_url") == "" {
			viper.Set("prometheus_url", args[0])
		}
	}

	// Validation: at least one URL must be set
	prometheusURL := viper.GetString("prometheus_url")
	nodeExporterURL := viper.GetString("node_exporter_url")

	if prometheusURL == "" && nodeExporterURL == "" {
		log.Fatalln("Error: prometheus_url or node_exporter_url must be set")
	}

	// Backend selection
	var d promtop.Data
	if prometheusURL != "" {
		log.Printf("Using Prometheus backend: %s", prometheusURL)
		d = &promtop.PrometheusData{}
	} else {
		log.Printf("Using node_exporter backend: %s", nodeExporterURL)
		d = &promtop.NodeExporterData{}
	}

	// Start dashboard
	promtop.Dashboard(promtop.Cache{Data: d})
}
