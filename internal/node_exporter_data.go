package promtop

import (
	"cmp"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type NodeExporterData struct {
	cpus       [][]*dto.Metric
	timestamps []time.Time
	nodes      map[string]*url.URL
}

func NewNodeExporterData(urls []*url.URL) (*NodeExporterData, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("at least one node_exporter URL required")
	}

	nodes := make(map[string]*url.URL)
	for _, u := range urls {
		hostname := u.Hostname()
		if hostname == "" {
			return nil, fmt.Errorf("URL missing hostname: %s", u.String())
		}
		// Include port in hostname if present
		if u.Port() != "" {
			hostname = hostname + ":" + u.Port()
		}
		nodes[hostname] = u
	}

	return &NodeExporterData{
		nodes: nodes,
	}, nil
}

func (n *NodeExporterData) Check() error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Test at least one node_exporter endpoint
	var lastErr error
	for hostname, u := range n.nodes {
		resp, err := client.Get(u.String())
		if err != nil {
			lastErr = fmt.Errorf("failed to connect to %s: %w", hostname, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("%s returned status %d", hostname, resp.StatusCode)
			continue
		}

		// Try to parse metrics to validate it's a real node_exporter endpoint
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response from %s: %w", hostname, err)
			continue
		}

		parser := expfmt.TextParser{}
		data, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
		if err != nil {
			lastErr = fmt.Errorf("failed to parse metrics from %s: %w", hostname, err)
			continue
		}

		// Validate it has CPU metrics (basic check)
		if _, ok := data["node_cpu_seconds_total"]; !ok {
			lastErr = fmt.Errorf("%s does not provide node_cpu_seconds_total metric", hostname)
			continue
		}

		// At least one node works
		return nil
	}

	// All nodes failed
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no node_exporter endpoints available")
}

func (n *NodeExporterData) GetNodes() []string {
	keys := make([]string, 0, len(n.nodes))
	for k := range n.nodes {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func (n *NodeExporterData) GetCpu(node string) map[string]float64 {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(n.nodes[node].String())
	if err != nil {
		log.Fatalln("Error querying node exporter:", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Failed to read response body:", err)
	}
	parser := expfmt.TextParser{}
	data, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		log.Fatalln("Failed to parse metrics:", err)
	}

	// extract cpu idle time metrics
	currentCpu := slices.DeleteFunc(data["node_cpu_seconds_total"].GetMetric(), func(metric *dto.Metric) bool {
		var cpu, mode string
		for _, label := range metric.GetLabel() {
			if label.GetName() == "cpu" {
				cpu = label.GetValue()
			}
			if label.GetName() == "mode" {
				mode = label.GetValue()
			}
		}
		_, err := strconv.Atoi(cpu)
		return mode != "idle" || err != nil
	})
	// clear the slice to free memory
	clear(currentCpu[len(currentCpu):cap(currentCpu)])

	// sort by cpu number
	slices.SortFunc(currentCpu, func(a, b *dto.Metric) int {
		return cmp.Compare(a.GetCounter().GetValue(), b.GetCounter().GetValue())
	})

	// append the new reading to the readings slice
	n.cpus = append(n.cpus, currentCpu)
	n.timestamps = append(n.timestamps, time.Now())
	// limit the readings slice to MaxCPURecords entries
	maxRecords := MaxCPURecords()
	if len(n.cpus) > maxRecords {
		n.cpus = n.cpus[1:]
		n.timestamps = n.timestamps[1:]
	}

	// calculate the cpu usage rates
	rates := make(map[string]float64)
	if len(n.cpus) < 2 {
		return rates
	}

	// calculate the interval between the first and last reading
	interval := n.timestamps[len(n.timestamps)-1].Sub(n.timestamps[0]).Seconds()
	for cpuIndex := 0; cpuIndex < len(n.cpus[0]); cpuIndex++ {
		// Get the CPU name from the metric labels
		var cpuName string
		for _, label := range n.cpus[0][cpuIndex].GetLabel() {
			if label.GetName() == "cpu" {
				cpuName = label.GetValue()
				break
			}
		}

		// each cpu counter might have been reset so we need to calculate an offset
		offset := 0.0
		for readingIndex, reading := range n.cpus {
			if readingIndex > 0 && reading[cpuIndex].GetCounter().GetValue() < n.cpus[readingIndex-1][cpuIndex].GetCounter().GetValue() {
				offset += n.cpus[readingIndex-1][cpuIndex].GetCounter().GetValue()
			}
		}
		// use the first nad last reading to calculate the cpu usage rate
		first := n.cpus[0][cpuIndex].GetCounter().GetValue() + offset
		last := n.cpus[len(n.cpus)-1][cpuIndex].GetCounter().GetValue() + offset
		// because the times should add up to the interval
		// we can calculate the cpu usage rate by subtracting the idle time from 100%
		rates[cpuName] = 100 - 100*(last-first)/interval
	}

	return rates
}

func (n *NodeExporterData) GetMemory(node string) map[string]float64 {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(n.nodes[node].String())
	if err != nil {
		log.Fatalln("Error querying node exporter:", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Failed to read response body:", err)
	}
	parser := expfmt.TextParser{}
	data, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		log.Fatalln("Failed to parse metrics:", err)
	}

	memory := make(map[string]float64)

	// Extract memory metrics
	if memTotal, ok := data["node_memory_MemTotal_bytes"]; ok && len(memTotal.GetMetric()) > 0 {
		memory["total"] = memTotal.GetMetric()[0].GetGauge().GetValue()
	}

	if memAvailable, ok := data["node_memory_MemAvailable_bytes"]; ok && len(memAvailable.GetMetric()) > 0 {
		memory["available"] = memAvailable.GetMetric()[0].GetGauge().GetValue()
	}

	// Calculate used memory and percentage
	if total, ok := memory["total"]; ok && total > 0 {
		if available, ok := memory["available"]; ok {
			used := total - available
			memory["used"] = used
			memory["used_percent"] = (used / total) * 100
		}
	}

	if memCached, ok := data["node_memory_Cached_bytes"]; ok && len(memCached.GetMetric()) > 0 {
		memory["cached"] = memCached.GetMetric()[0].GetGauge().GetValue()
	}

	if memBuffers, ok := data["node_memory_Buffers_bytes"]; ok && len(memBuffers.GetMetric()) > 0 {
		memory["buffers"] = memBuffers.GetMetric()[0].GetGauge().GetValue()
	}

	return memory
}

func (n *NodeExporterData) GetType() string {
	return "node_exporter"
}
