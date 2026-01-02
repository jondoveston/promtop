# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`promtop` is a terminal-based dashboard application for monitoring system metrics from Prometheus or directly from node_exporter. Built with Go and termui, it displays real-time CPU, memory, disk, and network metrics in an interactive terminal interface.

## Build & Run

This project uses [Task](https://taskfile.dev) for build automation:

```bash
# Build the application
task build

# Clean build artifacts
task clean

# Format code
task fmt

# Run tests
task test

# Install to /usr/local/bin
task install
```

### Running the Application

The application requires either a Prometheus URL or node_exporter URL:

```bash
# Run with Prometheus backend
PROMTOP_PROMETHEUS_URL=http://prometheus.lan:9090 ./promtop

# Run with node_exporter backend (direct metrics)
PROMTOP_NODE_EXPORTER_URL=http://localhost:9100/metrics ./promtop

# Or use the task shortcuts
task run_prom
task run_node
```

## Architecture

### Data Source Abstraction

The application uses an interface-based design to support multiple metric backends:

- **`Data` interface** (`internal/cache.go`): Defines `GetCpu()` and `GetInstances()` methods
- **`PrometheusData`** (`internal/prometheus_data.go`): Queries Prometheus API using PromQL
- **`NodeExporterData`** (`internal/node_exporter_data.go`): Scrapes node_exporter `/metrics` endpoint directly
- **`Cache`** (`internal/cache.go`): Wraps Data implementations with caching for instance lists

The main function (`main.go`) selects the appropriate backend based on environment variables.

### UI Architecture

The dashboard (`internal/dashboard.go`) uses gizak/termui v3:

- Left panel: scrollable list of instances/nodes
- Right panel: tabbed interface (CPU, Memory, Disk, Network)
- CPU tab displays per-core usage graphs that update every second
- Vim-style keybindings (j/k for scrolling, h/l for tab switching, g/G for top/bottom)

### CPU Metrics Calculation

**PrometheusData**: Uses PromQL query `100 - (avg by (instance,cpu) (rate(node_cpu_seconds_total{mode="idle"}[1m])) * 100)` to calculate CPU usage from idle time.

**NodeExporterData**: Implements custom rate calculation:
- Stores last 60 readings with timestamps
- Handles counter resets by tracking offsets
- Calculates usage as `100 - 100*(idle_delta)/interval`

## Configuration

Configuration is handled via Viper with environment variable support:

- Prefix: `PROMTOP_`
- `PROMTOP_PROMETHEUS_URL`: Prometheus server URL
- `PROMTOP_NODE_EXPORTER_URL`: node_exporter metrics endpoint (default: `http://localhost:9100/metrics`)

At least one URL must be set. If both are set, Prometheus takes precedence.

## Logging

Application logs to `promtop.log` in the current directory. Check this file for debugging connection issues or metric parsing errors.
