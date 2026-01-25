package promtop

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusData struct {
	client api.Client
	url    *url.URL
}

func NewPrometheusData(prometheusURL *url.URL) (*PrometheusData, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusData{
		client: client,
		url:    prometheusURL,
	}, nil
}

func (p *PrometheusData) Check() error {
	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test basic Prometheus API connectivity
	result, warnings, err := v1api.Query(ctx, "up", time.Now())
	if err != nil {
		return fmt.Errorf("prometheus API query failed: %w", err)
	}
	if len(warnings) > 0 {
		log.Printf("Prometheus warnings: %v", warnings)
	}

	// Verify node_exporter job exists
	result, _, err = v1api.Query(ctx, "up{job=\"node_exporter\"}", time.Now())
	if err != nil {
		return fmt.Errorf("node_exporter job query failed: %w", err)
	}
	if result.(model.Vector).Len() == 0 {
		return fmt.Errorf("no node_exporter targets found in prometheus")
	}

	return nil
}

func (p *PrometheusData) GetNodes() []string {
	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, "up{job=\"node_exporter\"}", time.Now())
	if err != nil {
		log.Fatalf("Error querying Prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Fatalf("Warnings: %v\n", warnings)
	}

	nodes := make([]string, 0, result.(model.Vector).Len())
	for _, val := range result.(model.Vector) {
		if val.Value == 1 {
			nodes = append(nodes, string(val.Metric["instance"]))
		}
	}

	return nodes
}

func (p *PrometheusData) GetCpu(node string) map[string]float64 {
	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf(
		"100 - (avg by (instance,cpu) (rate(node_cpu_seconds_total{instance=\"%s\",job=\"node_exporter\",mode=\"idle\"}[%s])) * 100)",
		node,
		CPURateIntervalString(),
	)

	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		log.Fatalf("Error querying Prometheus: %v", err)
	}
	if len(warnings) > 0 {
		log.Fatalf("Warnings: %v\n", warnings)
	}

	cpus := make(map[string]float64)
	for _, val := range result.(model.Vector) {
		cpuName := string(val.Metric["cpu"])
		cpus[cpuName] = float64(val.Value)
	}
	return cpus
}

func (p *PrometheusData) GetMemory(node string) map[string]float64 {
	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	memory := make(map[string]float64)

	// Get total memory (Linux: MemTotal_bytes, macOS: total_bytes)
	result, _, err := v1api.Query(ctx, fmt.Sprintf("node_memory_MemTotal_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["total"] = float64(result.(model.Vector)[0].Value)
	} else {
		// Try macOS naming
		result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_total_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
		if err == nil && result.(model.Vector).Len() > 0 {
			memory["total"] = float64(result.(model.Vector)[0].Value)
		}
	}

	// Get available memory (Linux only - MemAvailable_bytes)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_MemAvailable_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["available"] = float64(result.(model.Vector)[0].Value)
	}

	// Get free memory (both Linux and macOS have this)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_MemFree_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["free"] = float64(result.(model.Vector)[0].Value)
	} else {
		// Try macOS naming
		result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_free_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
		if err == nil && result.(model.Vector).Len() > 0 {
			memory["free"] = float64(result.(model.Vector)[0].Value)
		}
	}

	// Get cached memory (Linux: Cached_bytes)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_Cached_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["cached"] = float64(result.(model.Vector)[0].Value)
	}

	// Get buffer memory (Linux: Buffers_bytes)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_Buffers_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["buffers"] = float64(result.(model.Vector)[0].Value)
	}

	// macOS-specific metrics
	// Get active memory (macOS)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_active_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["active"] = float64(result.(model.Vector)[0].Value)
	}

	// Get inactive memory (macOS)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_inactive_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["inactive"] = float64(result.(model.Vector)[0].Value)
	}

	// Get wired memory (macOS)
	result, _, err = v1api.Query(ctx, fmt.Sprintf("node_memory_wired_bytes{instance=\"%s\",job=\"node_exporter\"}", node), time.Now())
	if err == nil && result.(model.Vector).Len() > 0 {
		memory["wired"] = float64(result.(model.Vector)[0].Value)
	}

	// Calculate used memory and percentage
	if total, ok := memory["total"]; ok && total > 0 {
		// Linux: use available if present
		if available, ok := memory["available"]; ok {
			used := total - available
			memory["used"] = used
			memory["used_percent"] = (used / total) * 100
		} else if free, ok := memory["free"]; ok {
			// macOS: calculate from active + wired (or total - free as fallback)
			if active, hasActive := memory["active"]; hasActive {
				if wired, hasWired := memory["wired"]; hasWired {
					used := active + wired
					memory["used"] = used
					memory["used_percent"] = (used / total) * 100
				}
			} else {
				// Fallback: total - free
				used := total - free
				memory["used"] = used
				memory["used_percent"] = (used / total) * 100
			}
		}
	}

	return memory
}

func (p *PrometheusData) GetType() string {
	return "prometheus"
}
