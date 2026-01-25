# promtop
A WIP terminal dashboard app that reads system metrics from Prometheus

## Core usage calculation

### The Core Concept

The `node_cpu_seconds_total{mode="idle"}` metric is a **cumulative counter** that tracks how many seconds a CPU core has spent in idle state since boot.

### How Rate of Idle Time Works

**Key insight**: In real time, time passes at exactly 1 second per second. A CPU core can either spend that time doing work OR being idle.

**If a core is completely idle:**
- For every 1 second of real time, the idle counter increases by 1 second
- Rate of idle time = **1.0 seconds/second** (100% of time is idle)
- CPU usage = `100 - (1.0 * 100) = 0%`

**If a core is completely busy:**
- For every 1 second of real time, the idle counter increases by 0 seconds (it's doing work)
- Rate of idle time = **0.0 seconds/second** (0% of time is idle)
- CPU usage = `100 - (0.0 * 100) = 100%`

**If a core is 50% busy:**
- For every 1 second of real time, the core spends 0.5s working and 0.5s idle
- The idle counter increases by 0.5 seconds
- Rate of idle time = **0.5 seconds/second** (50% of time is idle)
- CPU usage = `100 - (0.5 * 100) = 50%`

### In the Code

**Prometheus version:**
```go
100 - (avg by (instance,cpu) (rate(node_cpu_seconds_total{mode="idle"}[1m])) * 100)
```
- `rate(...[1m])` calculates: (idle_now - idle_1min_ago) / 60 seconds
- This gives seconds of idle per second of real time
- Multiply by 100 to get percentage idle
- Subtract from 100 to get percentage busy

**node_exporter version:**
```go
rates[cpuName] = 100 - 100*(last-first)/interval
```
- `(last-first)` = total idle seconds accumulated over the measurement window
- `interval` = total real-time seconds in the measurement window
- `(last-first)/interval` = fraction of time spent idle
- `100 - 100*fraction_idle` = percentage of time spent NOT idle (i.e., busy)

### Example Calculation

Over a 60-second window:
- **Scenario**: Core is 75% busy
- Idle counter increases from 1000.0s to 1015.0s (gained 15 seconds of idle time)
- Real time elapsed: 60 seconds
- Rate: 15 / 60 = 0.25 seconds idle per second
- CPU usage: 100 - (0.25 * 100) = **75% busy**

The calculation works because **idle time + busy time must equal real time**. If we know the idle fraction, we can deduce the busy fraction.

## Sampling interval

### Compromises of Using Long Intervals (60 seconds)

#### 1. Slower Response to Changes
- **Problem**: CPU spikes or drops take up to 60 seconds to fully reflect in the display
- **Example**: If a process suddenly uses 100% CPU, you won't see the full impact for a full minute
- **Impact**: Makes the dashboard less useful for real-time monitoring of sudden events

#### 2. Averaging Out Short Bursts
- **Problem**: Brief high-CPU events get diluted across the 60-second window
- **Example**:
  - A core runs at 100% for 6 seconds, then 0% for 54 seconds
  - 60-second window shows: 10% average usage
  - 1-second window would show: 100%, 100%, 100%, 100%, 100%, 100%, 0%, 0%, ...
- **Impact**: You completely miss short-duration CPU spikes that might be important

#### 3. Delayed Initial Reading
- **Problem**: Need to wait the full window duration before getting any data
- **Prometheus**: Needs 60 seconds of data history before `rate(...[1m])` returns meaningful results
- **node_exporter code**: Currently requires 2 readings, but rate accuracy improves as it approaches 60 readings
- **Impact**: 60+ second delay before dashboard shows anything useful after startup

#### 4. Poor for Bursty Workloads
- **Problem**: Many workloads are bursty (e.g., web servers, batch jobs, cron tasks)
- **Example**: A batch job runs every 5 minutes for 30 seconds
  - Short window: Clearly shows 30-second bursts
  - 60-second window: Smears it out, making patterns harder to see
- **Impact**: Harder to identify patterns or correlate with application behavior

#### 5. Latency in Troubleshooting
- **Problem**: When investigating a performance issue, 60-second lag makes it hard to correlate cause and effect
- **Example**: You run a command and want to see its CPU impact immediately
- **Impact**: By the time you see the CPU change, you've forgotten what you did

### The Trade-off

**Longer windows (60s):**
- ✅ Smoother, more stable graphs
- ✅ Less noise from brief fluctuations
- ✅ Better for capacity planning and trend analysis
- ❌ Hides short-term spikes
- ❌ Slow to respond to changes
- ❌ Poor for real-time troubleshooting

**Shorter windows (1-5s):**
- ✅ Immediate response to changes
- ✅ Shows all spikes and bursts
- ✅ Better for real-time monitoring
- ❌ Noisier data
- ❌ Harder to see long-term trends
- ❌ More susceptible to measurement artifacts

### Common Practice

Most monitoring tools offer **configurable windows** or use different windows for different purposes:
- **htop/top**: ~1-3 seconds (real-time monitoring)
- **Grafana dashboards**: Often 1-5 minutes (trend analysis)
- **Alerting**: 5-15 minutes (avoid alert fatigue from brief spikes)

Your current 60-second window is quite long for an interactive TUI. A 5-15 second window might be a better balance for `promtop`.
