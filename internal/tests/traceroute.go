package tests

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type Traceroute struct {
	outDir string
	cfg    config.Config
}

func NewTraceroute(outDir string, cfg config.Config) *Traceroute {
	return &Traceroute{outDir: outDir, cfg: cfg}
}

func (t *Traceroute) Name() string {
	return "traceroute"
}

func (t *Traceroute) Run(ctx context.Context) model.TestResult {
	result := baseResult(t.Name())
	result.StartedAt = time.Now()

	if len(t.cfg.Targets.Traceroute) == 0 {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "INFO",
			Title:    "No traceroute targets",
			Detail:   "No traceroute targets are configured.",
		})
		result.EndedAt = time.Now()
		return result
	}

	result.Status = StatusOK
	for _, target := range t.cfg.Targets.Traceroute {
		var output, logPath string
		var err error
		if runtime.GOOS == "windows" {
			output, logPath, err = sys.RunCommand(t.outDir, "tracert", target)
		} else {
			output, logPath, err = sys.RunCommand(t.outDir, "traceroute", target)
		}
		if logPath != "" {
			result.Evidence = append(result.Evidence, model.Evidence{Label: fmt.Sprintf("trace_%s", target), Path: logPath})
		}
		if err != nil {
			result.Status = StatusWarn
			result.Findings = append(result.Findings, model.Finding{
				Severity: "WARN",
				Title:    "Traceroute failed",
				Detail:   fmt.Sprintf("Traceroute to %s failed: %s", target, err.Error()),
			})
			continue
		}
		hops := countTracerouteHops(output)
		result.Metrics[fmt.Sprintf("%s_hops", target)] = fmt.Sprintf("%d", hops)
	}

	result.EndedAt = time.Now()
	return result
}

func countTracerouteHops(output string) int {
	lines := strings.Split(output, "\n")
	count := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0][0] >= '0' && fields[0][0] <= '9' {
			count++
		}
	}
	return count
}
