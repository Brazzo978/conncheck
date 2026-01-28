package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"conncheck/internal/config"
	"conncheck/internal/model"
)

func WriteJSON(outDir string, result model.Result) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(outDir, "results.json")
	return path, os.WriteFile(path, data, 0o644)
}

func WriteXML(outDir string, result model.Result) (string, error) {
	data, err := xml.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(outDir, "results.xml")
	return path, os.WriteFile(path, data, 0o644)
}

func WriteHTML(outDir string, result model.Result, cfg config.Config) (string, error) {
	tpl := template.Must(template.New("report").Parse(htmlTemplate))
	path := filepath.Join(outDir, "report.html")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	view := reportView{
		Result:    result,
		Speedtest: buildSpeedtestView(result, cfg.SpeedtestUI),
		DNS:       buildDNSView(result),
	}
	if err := tpl.Execute(file, view); err != nil {
		return "", err
	}
	return path, nil
}

type reportView struct {
	model.Result
	Speedtest *speedtestView
	DNS       *dnsBenchView
}

type speedtestView struct {
	Available            bool
	LocalDownloadMbps    float64
	LocalUploadMbps      float64
	DownloadMaxMbps      float64
	UploadMaxMbps        float64
	DownloadScale        []speedtestScaleView
	UploadScale          []speedtestScaleView
	DownloadCurrentScale *speedtestScaleView
	UploadCurrentScale   *speedtestScaleView
	Comparisons          []speedtestComparisonView
}

type speedtestScaleView struct {
	MinMbps     float64
	MaxMbps     float64
	Label       string
	Description string
}

type speedtestComparisonView struct {
	Label     string
	Percent   float64
	LossPct   float64
	SpeedMbps float64
}

type dnsBenchView struct {
	Available bool
	Domains   []string
	Servers   []dnsServerView
	MaxAvgMs  float64
	Summary   string
}

type dnsServerView struct {
	Server  string
	AvgMs   float64
	Percent float64
	Success int
	Fail    int
}

func buildSpeedtestView(result model.Result, cfg config.SpeedtestUI) *speedtestView {
	var speedtestResult *model.TestResult
	for _, test := range result.Tests {
		if test.Name == "speedtest" {
			speedtestResult = &test
			break
		}
	}
	if speedtestResult == nil {
		return nil
	}

	localDownBps, hasLocalDown := metricFloat(speedtestResult.Metrics, "local_avg_down_bps")
	localUpBps, hasLocalUp := metricFloat(speedtestResult.Metrics, "local_avg_up_bps")
	localDownMbps := localDownBps / 1_000_000
	localUpMbps := localUpBps / 1_000_000
	view := &speedtestView{
		Available:         hasLocalDown || hasLocalUp,
		LocalDownloadMbps: localDownMbps,
		LocalUploadMbps:   localUpMbps,
	}

	view.DownloadScale = toScaleView(cfg.DownloadScale)
	view.UploadScale = toScaleView(cfg.UploadScale)
	view.DownloadMaxMbps = maxScaleValue(view.DownloadScale, localDownMbps)
	view.UploadMaxMbps = maxScaleValue(view.UploadScale, localUpMbps)
	if hasLocalDown {
		view.DownloadCurrentScale = matchScale(view.DownloadScale, localDownMbps)
	}
	if hasLocalUp {
		view.UploadCurrentScale = matchScale(view.UploadScale, localUpMbps)
	}

	if hasLocalDown {
		view.Comparisons = buildComparisons(localDownMbps, speedtestResult.Metrics, cfg.Comparisons)
	}

	return view
}

func metricFloat(metrics model.StringMap, key string) (float64, bool) {
	value, ok := metrics[key]
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func toScaleView(scales []config.SpeedtestScale) []speedtestScaleView {
	views := make([]speedtestScaleView, 0, len(scales))
	for _, scale := range scales {
		views = append(views, speedtestScaleView{
			MinMbps:     scale.MinMbps,
			MaxMbps:     scale.MaxMbps,
			Label:       scale.Label,
			Description: scale.Description,
		})
	}
	return views
}

func maxScaleValue(scales []speedtestScaleView, current float64) float64 {
	maxValue := current
	for _, scale := range scales {
		if scale.MaxMbps > maxValue {
			maxValue = scale.MaxMbps
		}
		if scale.MaxMbps == 0 && scale.MinMbps > maxValue {
			maxValue = scale.MinMbps
		}
	}
	if maxValue <= 0 {
		return 100
	}
	return math.Ceil(maxValue)
}

func matchScale(scales []speedtestScaleView, value float64) *speedtestScaleView {
	for _, scale := range scales {
		if value >= scale.MinMbps && (scale.MaxMbps == 0 || value < scale.MaxMbps) {
			match := scale
			return &match
		}
	}
	if len(scales) > 0 {
		match := scales[len(scales)-1]
		return &match
	}
	return nil
}

func buildComparisons(localMbps float64, metrics model.StringMap, cfg config.SpeedtestCompare) []speedtestComparisonView {
	comparisons := []struct {
		label       string
		metricKey   string
		fallbackPct float64
	}{
		{label: "Nazionali", metricKey: "national_avg_down_bps", fallbackPct: cfg.NationalPct},
		{label: "Europei", metricKey: "eu_avg_down_bps", fallbackPct: cfg.EUPct},
		{label: "USA", metricKey: "us_avg_down_bps", fallbackPct: cfg.USPct},
	}
	views := []speedtestComparisonView{}
	for _, comparison := range comparisons {
		percent := 0.0
		if localMbps > 0 {
			if downBps, ok := metricFloat(metrics, comparison.metricKey); ok {
				percent = (downBps / 1_000_000) / localMbps * 100
			} else if comparison.fallbackPct > 0 {
				percent = comparison.fallbackPct
			}
		}
		if percent <= 0 {
			continue
		}
		views = append(views, speedtestComparisonView{
			Label:     comparison.label,
			Percent:   math.Round(percent),
			LossPct:   math.Round(100 - percent),
			SpeedMbps: localMbps * percent / 100,
		})
	}
	return views
}

func buildDNSView(result model.Result) *dnsBenchView {
	var dnsResult *model.TestResult
	for _, test := range result.Tests {
		if test.Name == "dns_benchmark" {
			dnsResult = &test
			break
		}
	}
	if dnsResult == nil {
		return nil
	}

	view := &dnsBenchView{}
	if domains, ok := dnsResult.Metrics["dns_domains"]; ok && domains != "" {
		view.Domains = strings.Split(domains, ",")
	}
	configuredServers := splitList(dnsResult.Metrics["dhcp_dns_servers"])

	avgValues := map[string]float64{}
	successValues := map[string]int{}
	failValues := map[string]int{}
	for key, value := range dnsResult.Metrics {
		if strings.HasPrefix(key, "dns_avg_ms.") {
			server := strings.TrimPrefix(key, "dns_avg_ms.")
			if parsed, err := strconv.ParseFloat(value, 64); err == nil {
				avgValues[server] = parsed
			}
		}
		if strings.HasPrefix(key, "dns_success.") {
			server := strings.TrimPrefix(key, "dns_success.")
			if parsed, err := strconv.Atoi(value); err == nil {
				successValues[server] = parsed
			}
		}
		if strings.HasPrefix(key, "dns_fail.") {
			server := strings.TrimPrefix(key, "dns_fail.")
			if parsed, err := strconv.Atoi(value); err == nil {
				failValues[server] = parsed
			}
		}
	}

	for server, avg := range avgValues {
		if avg > view.MaxAvgMs {
			view.MaxAvgMs = avg
		}
		view.Servers = append(view.Servers, dnsServerView{
			Server:  server,
			AvgMs:   avg,
			Success: successValues[server],
			Fail:    failValues[server],
		})
	}

	if len(view.Servers) > 1 {
		sort.Slice(view.Servers, func(i, j int) bool {
			return view.Servers[i].AvgMs < view.Servers[j].AvgMs
		})
	}

	if view.MaxAvgMs > 0 {
		for i := range view.Servers {
			view.Servers[i].Percent = view.Servers[i].AvgMs / view.MaxAvgMs * 100
		}
	}

	if len(view.Servers) > 0 {
		view.Available = true
	}

	view.Summary = buildDNSSummary(configuredServers, view.Servers)
	return view
}

func buildDNSSummary(configuredServers []string, servers []dnsServerView) string {
	if len(configuredServers) == 0 || len(servers) == 0 {
		return ""
	}

	configuredSet := map[string]bool{}
	for _, server := range configuredServers {
		if server == "" {
			continue
		}
		configuredSet[server] = true
	}

	configuredTotal := 0.0
	configuredCount := 0
	for _, server := range servers {
		if configuredSet[server.Server] {
			configuredTotal += server.AvgMs
			configuredCount++
		}
	}
	if configuredCount == 0 {
		return ""
	}
	configuredAvg := configuredTotal / float64(configuredCount)

	faster := []dnsServerView{}
	bestAvg := configuredAvg
	for _, server := range servers {
		if server.AvgMs < bestAvg {
			bestAvg = server.AvgMs
		}
		if server.AvgMs < configuredAvg && !configuredSet[server.Server] {
			faster = append(faster, server)
		}
	}

	if len(faster) == 0 {
		return "Complimenti: hai il server DNS migliore già configurato."
	}

	sort.Slice(faster, func(i, j int) bool {
		return faster[i].AvgMs < faster[j].AvgMs
	})
	alternativeNames := make([]string, 0, len(faster))
	for _, server := range faster {
		alternativeNames = append(alternativeNames, server.Server)
	}

	diffPct := (configuredAvg - bestAvg) / configuredAvg * 100
	if diffPct < 0 {
		diffPct = 0
	}
	return fmt.Sprintf(
		"Il DNS configurato attualmente in media è %.0f%% meno veloce di queste alternative misurate: %s.",
		math.Round(diffPct),
		strings.Join(alternativeNames, ", "),
	)
}

func splitList(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		results = append(results, trimmed)
	}
	return results
}

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>Conncheck Report</title>
<style>
body { font-family: "Segoe UI", sans-serif; margin: 24px; background: #f7f9fc; }
header { display: flex; justify-content: space-between; align-items: center; }
.badge { padding: 6px 12px; border-radius: 12px; background: #1f2937; color: #fff; }
section { background: #fff; padding: 16px; margin-top: 16px; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.05); }
.status-OK { color: #16a34a; }
.status-WARN { color: #d97706; }
.status-FAIL { color: #dc2626; }
.status-SKIPPED { color: #6b7280; }
.grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 12px; }
.card { background: #f9fafb; padding: 12px; border-radius: 8px; border: 1px solid #e5e7eb; }
.slider-block { margin-top: 8px; }
.slider-block input[type=range] { width: 100%; accent-color: #2563eb; }
.slider-value { font-weight: 600; margin-top: 6px; }
.slider-desc { margin: 4px 0 8px; color: #4b5563; }
.scale-list { list-style: none; padding-left: 0; margin: 8px 0 0; }
.scale-list li { margin-bottom: 8px; }
.comparison { margin-top: 12px; }
.pulse { animation: pulse 2s ease-in-out infinite; }
.dns-row { display: flex; align-items: center; gap: 12px; margin-top: 8px; }
.dns-label { width: 180px; font-weight: 600; }
.dns-bar { position: relative; flex: 1; height: 24px; background: #e5e7eb; border-radius: 999px; overflow: hidden; }
.dns-bar-fill { height: 100%; background: #60a5fa; border-radius: 999px; }
.dns-bar span { position: absolute; left: 10px; top: 3px; font-size: 12px; color: #111827; }
@keyframes pulse { 0% { transform: scale(1); } 50% { transform: scale(1.02); } 100% { transform: scale(1); } }
small { color: #6b7280; }
</style>
</head>
<body>
<header>
  <h1>Conncheck Report</h1>
  <div class="badge">Version {{ .Version }}</div>
</header>
<section>
  <h2>Environment</h2>
  <div class="grid">
    <div class="card"><strong>OS:</strong> {{ .Environment.OS }}<br/><small>{{ .Environment.Arch }}</small></div>
    <div class="card"><strong>Hostname:</strong> {{ .Environment.Hostname }}</div>
    <div class="card"><strong>Timezone:</strong> {{ .Environment.Timezone }}</div>
  </div>
</section>
<section>
  <h2>Findings</h2>
  {{ if .Findings }}
  <ul>
    {{ range .Findings }}
    <li><strong>{{ .Severity }}:</strong> {{ .Title }} — {{ .Detail }}</li>
    {{ end }}
  </ul>
  {{ else }}
  <p>No findings were recorded.</p>
  {{ end }}
</section>
{{ if .Speedtest }}
<section>
  <h2>Scala Speedtest (medie server locali)</h2>
  {{ if .Speedtest.Available }}
  <div class="grid">
    <div class="card">
      <h3>Download medio</h3>
      {{ if .Speedtest.DownloadCurrentScale }}
      <div class="slider-block">
        <input type="range" min="0" max="{{ printf "%.0f" .Speedtest.DownloadMaxMbps }}" value="{{ printf "%.0f" .Speedtest.LocalDownloadMbps }}" disabled />
        <div class="slider-value pulse">{{ printf "%.1f" .Speedtest.LocalDownloadMbps }} Mbps — {{ .Speedtest.DownloadCurrentScale.Label }}</div>
        <div class="slider-desc">{{ .Speedtest.DownloadCurrentScale.Description }}</div>
      </div>
      {{ end }}
      <ul class="scale-list">
        {{ range .Speedtest.DownloadScale }}
        <li><strong>{{ printf "%.0f" .MinMbps }}{{ if gt .MaxMbps 0.0 }}–{{ printf "%.0f" .MaxMbps }}{{ else }}+{{ end }} Mbps</strong> — {{ .Label }}<br/><small>{{ .Description }}</small></li>
        {{ end }}
      </ul>
    </div>
    <div class="card">
      <h3>Upload medio</h3>
      {{ if .Speedtest.UploadCurrentScale }}
      <div class="slider-block">
        <input type="range" min="0" max="{{ printf "%.0f" .Speedtest.UploadMaxMbps }}" value="{{ printf "%.0f" .Speedtest.LocalUploadMbps }}" disabled />
        <div class="slider-value pulse">{{ printf "%.1f" .Speedtest.LocalUploadMbps }} Mbps — {{ .Speedtest.UploadCurrentScale.Label }}</div>
        <div class="slider-desc">{{ .Speedtest.UploadCurrentScale.Description }}</div>
      </div>
      {{ end }}
      <ul class="scale-list">
        {{ range .Speedtest.UploadScale }}
        <li><strong>{{ printf "%.0f" .MinMbps }}{{ if gt .MaxMbps 0.0 }}–{{ printf "%.0f" .MaxMbps }}{{ else }}+{{ end }} Mbps</strong> — {{ .Label }}<br/><small>{{ .Description }}</small></li>
        {{ end }}
      </ul>
    </div>
  </div>
  {{ if .Speedtest.Comparisons }}
  <div class="card" style="margin-top: 12px;">
    <h3>Variazioni rispetto ai locali</h3>
    {{ range .Speedtest.Comparisons }}
    <div class="comparison">
      <div><strong>{{ .Label }}</strong>: {{ printf "%.0f" .Percent }}% ({{ printf "%.0f" .SpeedMbps }} Mbps, perdita {{ printf "%.0f" .LossPct }}%)</div>
      <input type="range" min="0" max="100" value="{{ printf "%.0f" .Percent }}" disabled />
    </div>
    {{ end }}
  </div>
  {{ end }}
  {{ else }}
  <p>Speedtest non disponibile o senza dati locali.</p>
  {{ end }}
</section>
{{ end }}
{{ if .DNS }}
<section>
  <h2>DNS Benchmark</h2>
  {{ if .DNS.Available }}
  {{ if .DNS.Summary }}
  <p><strong>{{ .DNS.Summary }}</strong></p>
  {{ end }}
  {{ if .DNS.Domains }}
  <p><small>Domains: {{ range $index, $domain := .DNS.Domains }}{{ if $index }}, {{ end }}{{ $domain }}{{ end }}</small></p>
  {{ end }}
  <div>
    {{ range .DNS.Servers }}
    <div class="dns-row">
      <div class="dns-label">{{ .Server }}</div>
      <div class="dns-bar">
        <div class="dns-bar-fill" style="width: {{ printf "%.0f" .Percent }}%;"></div>
        <span>{{ printf "%.1f" .AvgMs }} ms ({{ .Success }} ok, {{ .Fail }} fail)</span>
      </div>
    </div>
    {{ end }}
  </div>
  {{ else }}
  <p>DNS benchmark non disponibile o senza dati sufficienti.</p>
  {{ end }}
</section>
{{ end }}
<section>
  <h2>Test Results</h2>
  {{ range .Tests }}
  <div class="card">
    <h3>{{ .Name }} <span class="status-{{ .Status }}">({{ .Status }})</span></h3>
    <p><small>{{ .StartedAt }} → {{ .EndedAt }}</small></p>
    {{ if .Metrics }}
      <ul>
        {{ range $key, $value := .Metrics }}
        <li>{{ $key }}: {{ $value }}</li>
        {{ end }}
      </ul>
    {{ end }}
    {{ if .Evidence }}
      <p><strong>Evidence</strong></p>
      <ul>
        {{ range .Evidence }}
        <li>{{ .Label }}: {{ .Path }} {{ if .Note }}({{ .Note }}){{ end }}</li>
        {{ end }}
      </ul>
    {{ end }}
  </div>
  {{ end }}
</section>
<footer>
  <p><small>Generated at {{ .FinishedAt }}</small></p>
</footer>
</body>
</html>`

func FormatSummary(result model.Result) string {
	return fmt.Sprintf("Tests: %d, OK: %d, WARN: %d, FAIL: %d, SKIPPED: %d",
		len(result.Tests),
		result.Summary.StatusCounts["OK"],
		result.Summary.StatusCounts["WARN"],
		result.Summary.StatusCounts["FAIL"],
		result.Summary.StatusCounts["SKIPPED"],
	)
}
