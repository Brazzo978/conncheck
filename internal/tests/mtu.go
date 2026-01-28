package tests

import (
	"context"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
)

type MTU struct {
	outDir string
	cfg    config.Config
}

func NewMTU(outDir string, cfg config.Config) *MTU {
	return &MTU{outDir: outDir, cfg: cfg}
}

func (m *MTU) Name() string {
	return "mtu_pmtu"
}

func (m *MTU) Run(ctx context.Context) model.TestResult {
	result := baseResult(m.Name())
	result.StartedAt = time.Now()
	result.Status = StatusSkipped
	result.Metrics["targets"] = joinList(m.cfg.Targets.MTUTargets)
	result.Findings = append(result.Findings, model.Finding{
		Severity: "INFO",
		Title:    "MTU discovery pending",
		Detail:   "Base version reserves MTU/PMTU checks for future implementation.",
	})
	result.EndedAt = time.Now()
	return result
}
