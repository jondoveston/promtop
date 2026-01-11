package main

import (
	"fmt"
	"log"
	"net/url"
	"os"

	promtop "github.com/jondoveston/promtop/internal"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "promtop <url> [url...]",
	Short: "Terminal-based dashboard for Prometheus/node_exporter metrics",
	Long: `promtop displays real-time CPU, memory, disk, and network metrics
from Prometheus or node_exporter in an interactive terminal interface.

Each URL can point to either a Prometheus server or a node_exporter endpoint.
promtop will automatically detect which backend to use for each URL by trying
Prometheus first, then falling back to node_exporter if needed.

When multiple URLs are provided, nodes from all sources are displayed together
with source prefixes (e.g., "prometheus.lan:9090:hostname").

Examples:
  promtop http://prometheus.lan:9090
  promtop http://localhost:9100/metrics
  promtop http://prom1:9090 http://prom2:9090
  promtop http://prometheus.lan:9090 http://localhost:9100/metrics`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow no args if --version flag is set
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			return nil
		}
		return cobra.MinimumNArgs(1)(cmd, args)
	},
	Run: run,
}

func init() {
	// Define version flag
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")
}

func parseAndValidateURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Validate scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https, got: %s", u.Scheme)
	}

	// Validate host
	if u.Host == "" {
		return nil, fmt.Errorf("URL must have a host")
	}

	return u, nil
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

	var sources []promtop.Data
	var sourceNames []string

	// Process each URL
	for _, rawURL := range args {
		// Parse and validate
		targetURL, err := parseAndValidateURL(rawURL)
		if err != nil {
			log.Fatalf("Invalid URL '%s': %v", rawURL, err)
		}

		// Derive source name from URL (hostname:port or just hostname)
		sourceName := targetURL.Host

		// Try Prometheus first, then node_exporter
		var d promtop.Data

		log.Printf("Trying Prometheus backend: %s", targetURL)
		promData, err := promtop.NewPrometheusData(targetURL)
		if err != nil {
			log.Printf("Failed to create Prometheus client: %v", err)
		} else if err := promData.Check(); err != nil {
			log.Printf("Prometheus check failed: %v", err)
		} else {
			log.Printf("Using Prometheus backend for %s", sourceName)
			d = promData
		}

		// Fallback to node_exporter if Prometheus failed
		if d == nil {
			log.Printf("Trying node_exporter backend: %s", targetURL)
			nodeData, err := promtop.NewNodeExporterData([]*url.URL{targetURL})
			if err != nil {
				log.Fatalf("Failed to create node_exporter client for %s: %v", sourceName, err)
			}
			if err := nodeData.Check(); err != nil {
				log.Fatalf("Node exporter check failed for %s: %v", sourceName, err)
			}
			log.Printf("Using node_exporter backend for %s", sourceName)
			d = nodeData
		}

		sources = append(sources, d)
		sourceNames = append(sourceNames, sourceName)
	}

	// Wrap each source in a Cache for node list caching
	var caches []promtop.Cache
	for _, source := range sources {
		caches = append(caches, promtop.Cache{Data: source})
	}

	// Start dashboard with multiple sources
	promtop.Dashboard(caches, sourceNames)
}
