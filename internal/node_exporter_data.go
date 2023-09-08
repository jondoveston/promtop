package promtop

import (
	"cmp"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"net/url"

	"github.com/prometheus/common/expfmt"
	// "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/spf13/viper"
)

type NodeExporterData struct {
	cpus       [][]*dto.Metric
	timestamps []time.Time
	instances  map[string]*url.URL
}

func (n *NodeExporterData) instanceToUrl() map[string]*url.URL {
	if n.instances != nil {
		return n.instances
	}

  n.instances = make(map[string]*url.URL)

	for _, raw := range viper.GetStringSlice("node_exporter_url") {
		u, err := url.Parse(raw)
		if err != nil {
			log.Fatalln("Error parsing url:", raw, err)
		}
		n.instances[u.Hostname()] = u
	}
	return n.instances
}

func (n *NodeExporterData) GetInstances() []string {
	keys := make([]string, 0, len(n.instanceToUrl()))
	for k := range n.instanceToUrl() {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func (n *NodeExporterData) GetCpu(instance string) []float64 {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(n.instanceToUrl()[instance].String())
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
  // limit the readings slice to 60 entries
	if len(n.cpus) > 60 {
		n.cpus = n.cpus[1:]
		n.timestamps = n.timestamps[1:]
	}

  // calculate the cpu usage rates
	rates := []float64{}
	if len(n.cpus) < 2 {
		return rates
	}

  // calculate the interval between the first and last reading
	interval := n.timestamps[len(n.timestamps)-1].Sub(n.timestamps[0]).Seconds()
	for cpuIndex := 0; cpuIndex < len(n.cpus[0]); cpuIndex++ {
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
		rates = append(rates, 100-100*(last-first)/interval)
	}

	return rates
}
