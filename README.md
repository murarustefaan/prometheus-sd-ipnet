# Network Prometheus Service Discovery

A Go service that scans an IP range for a specific port and exposes discovered targets in Prometheus HTTP Service Discovery format.

## Features

- CIDR notation support for IP ranges
- Concurrent port scanning with configurable timeout
- Periodic rescanning to detect new/removed targets
- Prometheus HTTP SD compatible JSON output
- Health check endpoint
- Configurable via command-line flags

## Installation

```bash
go build -o network-prometheus-sd
```

## Usage

```bash
./network-prometheus-sd -ip-range 192.168.1.0/24 -port 9100
```

### Command-line Flags

- `-ip-range`: IP range in CIDR notation (required, e.g., `192.168.1.0/24`)
- `-port`: Target port to scan (default: `80`)
- `-scan-timeout`: Timeout for individual port scans (default: `2s`)
- `-rescan-interval`: Interval between rescans (default: `5m`)
- `-listen`: HTTP server listen address (default: `:8080`)

### Examples

Scan for node exporters on a local network:
```bash
./network-prometheus-sd -ip-range 192.168.1.0/24 -port 9100 -rescan-interval 2m
```

Scan a single host:
```bash
./network-prometheus-sd -ip-range 192.168.1.10 -port 8080
```

Scan a larger subnet:
```bash
./network-prometheus-sd -ip-range 10.0.0.0/16 -port 9090 -scan-timeout 1s
```

## Endpoints

- `GET /sd` - Prometheus service discovery endpoint (returns JSON)
- `GET /health` - Health check endpoint

## Prometheus Configuration

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'network-discovered'
    http_sd_configs:
      - url: http://localhost:8080/sd
        refresh_interval: 30s
```

## Output Format

The service returns targets in Prometheus HTTP SD format:

```json
[
  {
    "targets": ["192.168.1.10:9100"],
    "labels": {
      "job": "network-scan",
      "source": "network-prometheus-sd",
      "ip": "192.168.1.10"
    }
  },
  {
    "targets": ["192.168.1.15:9100"],
    "labels": {
      "job": "network-scan",
      "source": "network-prometheus-sd",
      "ip": "192.168.1.15"
    }
  }
]
```

## How It Works

1. The service parses the provided CIDR range into individual IP addresses
2. It concurrently scans each IP for the specified port (max 100 concurrent workers)
3. Successfully connected IPs are collected and formatted as Prometheus targets
4. The results are exposed via HTTP at `/sd` endpoint
5. The scan repeats at the configured interval to detect changes

## Performance Considerations

- The service uses a worker pool (max 100 concurrent connections) to prevent overwhelming the network
- Adjust `-scan-timeout` based on your network latency
- For large IP ranges, consider increasing `-rescan-interval` to reduce network load
- A /24 subnet (254 IPs) typically completes in 5-10 seconds with default settings