# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## Project Overview

`promtop` is a terminal-based dashboard application for monitoring system
metrics from Prometheus or directly from node_exporter. Built with Go and
Bubble Tea, it displays real-time CPU, memory, disk, and network metrics in an
interactive terminal interface.

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

The application accepts one or more URLs as positional arguments. Each URL can
point to either a Prometheus server or a node_exporter endpoint. The backend
type is automatically detected for each URL.

```bash
# Single Prometheus source
./promtop http://prometheus.lan:9090

# Single node_exporter source
./promtop http://localhost:9100/metrics

# Multiple sources (mixed types supported)
./promtop http://prometheus.lan:9090 http://localhost:9100/metrics

# Or use the task shortcuts
task run_prom    # Single Prometheus
task run_node    # Single node_exporter
task run         # Multiple sources
```

URLs must use HTTP or HTTPS scheme and include a host. The application will
attempt to connect to each URL, trying Prometheus first, then falling back to
node_exporter if Prometheus detection fails.

## Architecture

### Data Source Abstraction

The application uses an interface-based design to support multiple metric
backends:

- **`Data` interface** (`internal/cache.go`): Defines `GetCpu()`, `GetNodes()`,
  `Check()`, and `GetType()` methods
- **`PrometheusData`** (`internal/prometheus_data.go`): Queries Prometheus API
  using PromQL, returns type "prometheus"
- **`NodeExporterData`** (`internal/node_exporter_data.go`): Scrapes
  node_exporter `/metrics` endpoint directly, returns type "node_exporter"
- **`Cache`** (`internal/cache.go`): Wraps Data implementations with caching for
  node lists

The main function (`main.go`) uses Cobra for CLI argument parsing. For each
provided URL, it:

1. Parses and validates the URL (scheme, host)
2. Creates a Prometheus client and calls `Check()`
3. Falls back to node_exporter if Prometheus check fails
4. Wraps each successful Data source in a Cache
5. Passes all sources to the Dashboard

### Multiple Source Support

When multiple URLs are provided, nodes from all sources are displayed in a
unified interface:

- Prometheus sources show as selectable orange headers with nodes listed beneath
- Node_exporter sources show as single-line entries with orange styling
- Each node tracks its source via a `NodeRef` structure containing `Type`,
  `SourceIndex`, `SourceName`, `NodeName`, and `DisplayName`
- Node types: `"prometheus"` (source header), `"prometheus_node"` (individual
  Prometheus target), `"node_exporter"` (direct node_exporter)

### UI Architecture

The dashboard (`internal/dashboard.go`) uses Bubble Tea and Lipgloss:

- Left panel: scrollable list of sources and nodes with color-coded headers
- Right panel: tabbed interface (CPU, Memory, Disk, Network)
- CPU tab displays per-core usage with sparkline graphs that update every second
- Selecting a Prometheus source header shows a blank panel
- Vim-style keybindings:
  - `j`/`k` or arrows: navigate nodes
  - `h`/`l` or arrows: switch tabs
  - `g`/`G`: jump to top/bottom
  - `Ctrl+d`/`Ctrl+u`: scroll by 5 items
  - `q` or `Ctrl+c`: quit

### CPU Metrics Calculation

**PrometheusData**: Uses PromQL query
`100 - (avg by (instance,cpu) (rate(node_cpu_seconds_total{mode="idle"}[1m])) * 100)`
to calculate CPU usage from idle time.

**NodeExporterData**: Implements custom rate calculation:

- Stores last 60 readings with timestamps
- Handles counter resets by tracking offsets
- Calculates usage as `100 - 100*(idle_delta)/interval`

## CLI Interface

Command-line interface built with Cobra:

```
Usage:
  promtop <url> [url...]

Flags:
  -v, --version   Print version information
```

URLs are validated before connection attempts. Invalid URLs (missing scheme,
missing host, non-HTTP/HTTPS) cause immediate error exit.

## Logging

Application logs to `promtop.log` in the current directory via stderr
redirection (see Taskfile). Check this file for debugging connection issues,
metric parsing errors, or backend detection results.
