package tests

import (
	"context"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
)

type DNSBench struct {
	outDir string
	cfg    config.Config
}

func NewDNSBench(outDir string, cfg config.Config) *DNSBench {
	return &DNSBench{outDir: outDir, cfg: cfg}
}

func (d *DNSBench) Name() string {
	return "dns_benchmark"
}

func (d *DNSBench) Run(ctx context.Context) model.TestResult {
	result := baseResult(d.Name())
	result.StartedAt = time.Now()
	result.Status = StatusSkipped
	result.Metrics["servers"] = joinList(d.cfg.Targets.DNSServers)
	result.Findings = append(result.Findings, model.Finding{
		Severity: "INFO",
		Title:    "DNS benchmark not yet implemented",
		Detail:   "Base version includes configuration placeholders for DNS benchmarking.",
	})
	result.EndedAt = time.Now()
	return result
}
