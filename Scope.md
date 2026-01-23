conncheck — Project Scope (Go, Windows-first)

1) One‑sentence goal

Build a one‑click Windows-native diagnostic tool (single .exe) that runs an in‑depth connectivity test suite, shows a small status window during execution, and outputs machine‑readable results (XML + JSON) plus a human report (HTML) and raw evidence logs.

This tool is meant for helpdesk / ISP troubleshooting and should answer:

Is the issue Wi‑Fi/LAN, last‑mile ISP, routing/peering, DNS, IPv6, MTU/PPPoE clamping, or bufferbloat under load?

2) Non‑goals

No “full app” GUI (no complex settings UI). Only a minimal progress/status window.

No permanent system changes (do not edit registry, DNS, MTU, etc.). Only observe + measure.

No telemetry by default. Results are written locally; sharing is user-driven.

3) Primary deliverables

3.1 Local artifacts

The tool must write an output folder (and optionally a zip bundle) containing:

results.xml — structured, explorable

results.json — structured for web charts

report.html — readable summary with green/yellow/red sections

raw_logs/ — evidence: command outputs, raw speedtest json, traces

bundle.zip — optional, includes everything above

3.2 UX

Default mode: double‑click runs and produces outputs.

Minimal status window:

current step text (e.g., “DNS benchmark…”)

progress bar

scrolling log (last ~50 lines)

optional buttons: “Open report”, “Open output folder”, “Copy summary”

CLI mode also supported (--no-ui).

4) Technology choices

Language: Go

UI: Fyne (minimal window). The tool must still work without UI.

Config: optional YAML (conncheck.yaml) with sane defaults.

Speedtest: uses Ookla Speedtest CLI (speedtest.exe) if present.

5) Principles / quality requirements

5.1 Evidence-first diagnostics

Every finding must be backed by metrics + evidence paths/snippets.

5.2 Fail-soft behavior

If a test cannot run (blocked ICMP, missing speedtest binary, no upload endpoint), it must return SKIPPED with a clear reason.

5.3 Repeatability

Use defined timeouts, retries, sample sizes, and compute stable aggregates (median, p95).

5.4 Privacy-aware

Support privacy modes:

minimal: omit SSID / public IP, reduce raw logs

standard: include typical troubleshooting info

full: include everything (still no secrets)

6) Core test suite (modules)

Each module returns a TestResult with:

status: OK / WARN / FAIL / SKIPPED

metrics: key-value measurements

evidence: file paths, small excerpts

findings: optional per-test messages

6.1 Preflight / Environment snapshot

Goal: understand the measurement context.

Detect active interface and connection type: Ethernet / Wi‑Fi / WWAN

If Wi‑Fi: parse netsh wlan show interfaces (band/channel/signal/link rate)

Detect VPN/proxy hints (virtual adapters, default route anomalies, env proxy)

Collect:

ipconfig /all

route print

DNS servers, MTU per interface

Outputs: environment summary + warnings (e.g., “Wi‑Fi results may vary”).

6.2 LAN health check

Goal: separate LAN/Wi‑Fi issues from ISP issues.

Ping default gateway (e.g., 200 samples)

Compute loss, min/avg/max, jitter (stddev or p95-p50)

If gateway is unstable → flag as LAN/Wi‑Fi root cause.

6.3 Dual‑stack IPv4/IPv6 validation

Goal: detect broken IPv6 causing slow browsing.

Verify IPv6 global address + IPv6 default route

Test reachability:

IPv4-only target

IPv6-only target

dualstack target

Check DNS A/AAAA resolution works as expected

Status mapping:

OK: IPv6 works

ABSENT: no IPv6

BROKEN: IPv6 present but fails

DEGRADED: intermittent/slow IPv6

6.4 DNS benchmark (system vs best)

Goal: identify slow/unreliable resolvers.

Test system DNS servers + preset list:

Cloudflare (1.1.1.1)

Google (8.8.8.8)

Quad9 (9.9.9.9)

optional AdGuard

Measure p50/p95 latency, failure rate

Query mix: A, AAAA, NXDOMAIN

Rank resolvers and recommend best.

6.5 MTU / PMTU discovery (clamping/PPPoE)

Goal: detect path MTU issues when interface says 1500.

Use DF pings (ping -f -l) to estimate PMTU to multiple targets (EU/US)

If PMTU < 1500 while interface MTU is 1500 → warn “possible clamping/PPPoE”.

6.6 Latency & jitter (unloaded)

Goal: baseline latency quality.

Multi-destination pings:

gateway

IT/EU/US targets

ICMP first; fallback to TCP 443 timing if ICMP blocked.

Metrics: loss, min/avg/max, jitter, p95.

6.7 Bufferbloat / loaded latency (download + upload)

Goal: detect latency spikes under load (video calls/gaming problems).

Measure baseline ping to a stable target

Run download load (Hetzner test file or similar) while pinging in parallel

Run upload load (HTTP POST to configured endpoint) while pinging in parallel

Compute delta latency p95, jitter under load, loss under load

Grade:

OK: small delta

WARN: moderate delta

FAIL: large delta

6.8 Speedtest (Ookla CLI)

Goal: measure throughput + latency from controlled servers.

If speedtest.exe exists next to the tool (or in PATH), run:

--format=json (and auto-accept flags if needed)

Use 2–3 server regions (IT / EU / US) and repeat runs (median)

Save raw JSON evidence

If missing binary: SKIPPED with reason.

6.9 Traceroute / path analysis

Goal: pinpoint routing/peering anomalies.

Run tracert to key targets (EU/US)

Parse hop count, detect loops, latency jumps, suspicious detours

Important: do not over-interpret per-hop loss (ICMP rate limiting)

6.10 HTTP “real world” timing checks

Goal: catch cases where speedtest is fine but websites feel slow.

For a few endpoints, measure:

connect time

TLS handshake time

TTFB

download throughput for a medium file

7) Scoring & final report

7.1 Findings

Produce a short list of actionable findings, e.g.:

“You are on 2.4GHz Wi‑Fi with low signal; expect jitter.”

“IPv6 is present but failing (timeouts) — can slow down browsing.”

“Severe bufferbloat on upload (+160ms p95) — calls will stutter; enable SQM/QoS.”

“DNS resolver is slow (p95 180ms). Best found: 1.1.1.1 (p95 35ms).”

“PMTU detected 1492 while interface MTU is 1500 — possible PPPoE clamping.”

7.2 Status colors

Each test gets a color/severity:

Green (OK)

Yellow (WARN)

Red (FAIL)

Grey (SKIPPED)

7.3 Report outputs

HTML report groups results by category and shows the top findings first.

JSON/XML contain the full structure + raw evidence references.

8) Configuration (optional YAML)

Support conncheck.yaml with:

ping targets (IT/EU/US)

DNS preset resolvers

download URL (Hetzner or similar)

upload URL (your endpoint)

speedtest server selection

thresholds for OK/WARN/FAIL

privacy mode defaults

If no config exists, run with baked-in defaults.

9) Repository structure (suggested)

conncheck/
  cmd/conncheck/
    main.go
  internal/
    engine/        # orchestrates pipeline, progress events
    tests/
      preflight/
      lan/
      dualstack/
      dnsbench/
      mtu/
      latency/
      bufferbloat/
      speedtest/
      traceroute/
      httpcheck/
    sys/           # Windows command wrappers + parsers
    report/        # xml/json/html writers
    ui/            # Fyne minimal window (optional)
    config/        # YAML config
  assets/
    report_template.html
  scripts/
    build.ps1
  README.md
  scope.md
  go.mod

10) Implementation roadmap (recommended order)

Core data model (Result, TestResult, Finding) + JSON/XML writer

Preflight snapshot + raw log capture

LAN ping to gateway + unloaded latency module

DNS benchmark module

MTU/PMTU discovery module

Bufferbloat (download + upload) module

Speedtest module (Ookla CLI integration)

Traceroute module + path heuristics

HTTP timing module

HTML report template + Fyne status window polish

Privacy modes + zip bundling

11) Acceptance criteria

Produces results.xml, results.json, report.html reliably on Windows.

Shows a minimal progress/status window by default; can disable UI.

Runs in quick/standard/deep modes.

Handles missing dependencies gracefully (speedtest, upload endpoint).

Reports clear, actionable findings with evidence paths.

