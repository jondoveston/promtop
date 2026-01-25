package promtop

import (
	"fmt"
	"math"
	"time"
)

const (
	// CPU_RATE_INTERVAL is the time window in seconds used to calculate CPU usage rates
	CPU_RATE_INTERVAL = 60

	// UPDATE_INTERVAL is the time between data updates in seconds
	UPDATE_INTERVAL = 1
)

// MaxCPURecords returns the maximum number of CPU readings to store
// Calculated as CPU_RATE_INTERVAL / UPDATE_INTERVAL, rounded
func MaxCPURecords() int {
	return int(math.Round(float64(CPU_RATE_INTERVAL) / float64(UPDATE_INTERVAL)))
}

// UpdateDuration returns the update interval as a time.Duration
func UpdateDuration() time.Duration {
	return time.Duration(UPDATE_INTERVAL) * time.Second
}

// CPURateIntervalString returns the CPU rate interval formatted for Prometheus queries (e.g., "60s")
func CPURateIntervalString() string {
	return fmt.Sprintf("%ds", CPU_RATE_INTERVAL)
}
