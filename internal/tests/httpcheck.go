package tests

import (
	"context"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
)

type HTTPCheck struct {
	outDir string
	cfg    config.Config
}

func NewHTTPCheck(outDir string, cfg config.Config) *HTTPCheck {
	return &HTTPCheck{outDir: outDir, cfg: cfg}
}

func (h *HTTPCheck) Name() string {
	return "http_check"
}

func (h *HTTPCheck) Run(ctx context.Context) model.TestResult {
	result := baseResult(h.Name())
	result.StartedAt = time.Now()
	result.Status = StatusSkipped
	result.Metrics["endpoints"] = joinList(h.cfg.HTTP.Endpoints)
	result.Findings = append(result.Findings, model.Finding{
		Severity: "INFO",
		Title:    "HTTP timing checks pending",
		Detail:   "Endpoints are configured; timing probes will be added in a future version.",
	})
	result.EndedAt = time.Now()
	return result
}
