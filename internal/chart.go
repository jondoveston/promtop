package promtop

// Chart represents a chart displaying metrics for a specific node
type Chart struct {
	NodeRef    NodeRef
	ChartType  string               // "cpu", "memory", "disk", "network"
	CpuData    map[string][]float64 // CPU name -> time series
	MemoryData map[string]float64   // Memory metrics: total, available, used, used_percent, cached, buffers
}
