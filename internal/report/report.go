package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

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

func WriteHTML(outDir string, result model.Result) (string, error) {
	tpl := template.Must(template.New("report").Parse(htmlTemplate))
	path := filepath.Join(outDir, "report.html")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := tpl.Execute(file, result); err != nil {
		return "", err
	}
	return path, nil
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
