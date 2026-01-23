package tests

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type LAN struct {
	outDir string
}

func NewLAN(outDir string) *LAN {
	return &LAN{outDir: outDir}
}

func (l *LAN) Name() string {
	return "lan_health"
}

func (l *LAN) Run(ctx context.Context) model.TestResult {
	result := baseResult(l.Name())
	result.StartedAt = time.Now()

	gateway, routeLog, err := DetectDefaultGateway(l.outDir)
	if routeLog != "" {
		result.Evidence = append(result.Evidence, model.Evidence{Label: "route_print", Path: routeLog})
	}
	if err != nil {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Gateway detection failed",
			Detail:   err.Error(),
		})
		result.EndedAt = time.Now()
		return result
	}
	if gateway == "" {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Gateway not found",
			Detail:   "Unable to locate default gateway from routing table.",
		})
		result.EndedAt = time.Now()
		return result
	}

	var output string
	var pingLog string
	if runtime.GOOS == "windows" {
		output, pingLog, err = sys.RunCommand(l.outDir, "ping", "-n", "20", gateway)
	} else {
		output, pingLog, err = sys.RunCommand(l.outDir, "ping", "-c", "20", gateway)
	}
	if pingLog != "" {
		result.Evidence = append(result.Evidence, model.Evidence{Label: "gateway_ping", Path: pingLog})
	}
	if err != nil {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Gateway ping failed",
			Detail:   err.Error(),
		})
		result.EndedAt = time.Now()
		return result
	}

	stats := ParsePing(output)
	result.Metrics["gateway"] = gateway
	result.Metrics["loss_pct"] = fmt.Sprintf("%d", stats.LossPct)
	result.Metrics["min_ms"] = fmt.Sprintf("%d", stats.MinMs)
	result.Metrics["avg_ms"] = fmt.Sprintf("%d", stats.AvgMs)
	result.Metrics["max_ms"] = fmt.Sprintf("%d", stats.MaxMs)

	result.Status = StatusOK
	if stats.LossPct > 0 || stats.AvgMs > 20 {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "Gateway latency or loss",
			Detail:   "The local gateway shows packet loss or elevated latency. Wi-Fi/LAN may be unstable.",
		})
	}

	result.EndedAt = time.Now()
	return result
}
