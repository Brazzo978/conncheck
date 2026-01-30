package tests

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type Latency struct {
	outDir string
	cfg    config.Config
}

func NewLatency(outDir string, cfg config.Config) *Latency {
	return &Latency{outDir: outDir, cfg: cfg}
}

func (l *Latency) Name() string {
	return "latency"
}

func (l *Latency) Run(ctx context.Context) model.TestResult {
	result := baseResult(l.Name())
	result.StartedAt = time.Now()

	if len(l.cfg.Targets.PingTargets) == 0 {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "INFO",
			Title:    "No ping targets",
			Detail:   "No targets configured for latency checks.",
		})
		result.EndedAt = time.Now()
		return result
	}

	result.Status = StatusOK
	for _, target := range l.cfg.Targets.PingTargets {
		var output, logPath string
		var err error
		if runtime.GOOS == "windows" {
			output, logPath, err = sys.RunCommand(l.outDir, "ping", "-n", "10", target)
		} else {
			output, logPath, err = sys.RunCommand(l.outDir, "ping", "-c", "10", target)
		}
		if logPath != "" {
			result.Evidence = append(result.Evidence, model.Evidence{Label: fmt.Sprintf("ping_%s", target), Path: logPath})
		}
		if err != nil {
			result.Status = StatusWarn
			result.Findings = append(result.Findings, model.Finding{
				Severity: "WARN",
				Title:    "Ping failed",
				Detail:   fmt.Sprintf("Ping to %s failed: %s", target, err.Error()),
			})
			continue
		}
		stats := ParsePing(output)
		result.Metrics[fmt.Sprintf("%s_avg_ms", target)] = fmt.Sprintf("%d", stats.AvgMs)
		result.Metrics[fmt.Sprintf("%s_loss_pct", target)] = fmt.Sprintf("%d", stats.LossPct)
	}

	result.EndedAt = time.Now()
	return result
}
