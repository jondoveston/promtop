package promtop

import (
	"context"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/viper"
)

type PrometheusData struct {
}

func (p *PrometheusData) getClient() api.Client {
	client, err := api.NewClient(api.Config{
		Address: viper.GetString("prometheus_url"),
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	return client
}

func (p *PrometheusData) GetNodes() []string {
	client := p.getClient()

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

	nodes := make([]string, 0, result.(model.Vector).Len())
	for _, val := range result.(model.Vector) {
		if val.Value == 1 {
			nodes = append(nodes, string(val.Metric["instance"]))
		}
	}

	return nodes
}

func (p *PrometheusData) GetCpu(node string) []float64 {
	client := p.getClient()

	v1api := v1.NewAPI(client)
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
