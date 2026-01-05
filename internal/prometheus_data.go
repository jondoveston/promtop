package promtop

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
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

func (p *PrometheusData) GetCpu(node string) []float64 {
	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := v1api.Query(ctx, "100 - (avg by (instance,cpu) (rate(node_cpu_seconds_total{instance=\""+node+"\",job=\"node_exporter\",mode=\"idle\"}[1m])) * 100)", time.Now())
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
