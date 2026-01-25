package promtop

import (
	"log"
	"net/url"
)

// DetectedSource holds a data source and its display name
type DetectedSource struct {
	Data Data
	Name string
}

// TryConnectWithFallbacks tries multiple URL variants to connect to data sources
// Returns all successful connections (can be both Prometheus and node_exporter)
func TryConnectWithFallbacks(baseURL *url.URL) []DetectedSource {
	var detected []DetectedSource

	// Generate URL variants
	variants := generateURLVariants(baseURL)

	// Try Prometheus with all variants
	var promData Data
	var promURL *url.URL
	for _, variant := range variants {
		log.Printf("Trying Prometheus backend: %s", variant)
		pd, err := NewPrometheusData(variant)
		if err != nil {
			log.Printf("Failed to create Prometheus client: %v", err)
			continue
		}
		if err := pd.Check(); err != nil {
			log.Printf("Prometheus check failed: %v", err)
			continue
		}
		log.Printf("✓ Found Prometheus backend at %s", variant)
		promData = pd
		promURL = variant
		break
	}

	// Try node_exporter with all variants
	var nodeData Data
	var nodeURL *url.URL
	for _, variant := range variants {
		log.Printf("Trying node_exporter backend: %s", variant)
		nd, err := NewNodeExporterData([]*url.URL{variant})
		if err != nil {
			log.Printf("Failed to create node_exporter client: %v", err)
			continue
		}
		if err := nd.Check(); err != nil {
			log.Printf("Node exporter check failed: %v", err)
			continue
		}
		log.Printf("✓ Found node_exporter backend at %s", variant)
		nodeData = nd
		nodeURL = variant
		break
	}

	// Add successful connections
	if promData != nil {
		detected = append(detected, DetectedSource{
			Data: promData,
			Name: promURL.Host,
		})
	}

	if nodeData != nil {
		detected = append(detected, DetectedSource{
			Data: nodeData,
			Name: nodeURL.Host,
		})
	}

	return detected
}

// generateURLVariants creates different URL combinations to try
func generateURLVariants(base *url.URL) []*url.URL {
	var variants []*url.URL
	hostname := base.Hostname()
	port := base.Port()
	path := base.Path

	// Schemes to try: prefer HTTPS, fallback to HTTP
	schemes := []string{"https", "http"}
	if base.Scheme == "http" {
		schemes = []string{"http", "https"}
	}

	// Ports to try
	var ports []string
	if port != "" {
		// Use specified port first, then try common ports
		ports = []string{port, "9090", "9100", "443", "80"}
	} else {
		// Try common ports: 9090 (Prometheus), 9100 (node_exporter), 443 (HTTPS), 80 (HTTP)
		ports = []string{"9090", "9100", "443", "80"}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniquePorts := []string{}
	for _, p := range ports {
		if !seen[p] {
			seen[p] = true
			uniquePorts = append(uniquePorts, p)
		}
	}
	ports = uniquePorts

	// Paths to try
	paths := []string{path}
	if path == "" || path == "/" {
		// Try with and without /metrics
		paths = []string{"", "/metrics"}
	}

	// Generate all combinations
	for _, scheme := range schemes {
		for _, p := range ports {
			for _, urlPath := range paths {
				u := &url.URL{
					Scheme: scheme,
					Host:   hostname + ":" + p,
					Path:   urlPath,
				}
				variants = append(variants, u)
			}
		}
	}

	return variants
}
