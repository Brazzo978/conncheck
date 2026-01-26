# conncheck

Conncheck is a Windows-first connectivity diagnostic tool designed for helpdesk and ISP troubleshooting. It runs a suite of checks, collects evidence, and generates structured JSON/XML plus a human-friendly HTML report.

## What this base version includes

- Core data model and reporting pipeline (JSON/XML/HTML).
- Preflight collection of `ipconfig` / `route print` logs.
- LAN gateway ping health check.
- Dual-stack IPv4/IPv6 presence + reachability probe.
- Latency checks for configurable ping targets.
- Speedtest integration (if `speedtest.exe` is present).
- Traceroute parsing for hop counts.
- Placeholder modules for DNS benchmarks, MTU/PMTU, bufferbloat, and HTTP timing.

## Run (Windows)

```powershell
# Build
Go build -o conncheck.exe ./cmd/conncheck

# Run (default output in ./outputs/<timestamp>)
.\conncheck.exe

# Run with config
.\conncheck.exe -config conncheck.yaml
```

## Configuration

Create `conncheck.yaml` before running (configuration is required). A sample file is provided in `conncheck.sample.yaml`.

Key sections:
- `targets`: ping targets, DNS servers, traceroute targets, MTU targets.
- `speedtest`: local/national/EU/US server IDs with per-category runs (weights are derived from distance).
- `thresholds`: warning/fail thresholds for future alerting.

## Outputs

Each run generates:
- `results.json`
- `results.xml`
- `report.html`
- `raw_logs/` with command outputs

## Next steps

This base version focuses on scaffolding. Advanced modules (DNS benchmark, bufferbloat, MTU, HTTP timing) are wired for future implementation.
