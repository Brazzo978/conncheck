package tests

import (
	"context"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
)

type Bufferbloat struct {
	outDir string
	cfg    config.Config
}

func NewBufferbloat(outDir string, cfg config.Config) *Bufferbloat {
	return &Bufferbloat{outDir: outDir, cfg: cfg}
}

func (b *Bufferbloat) Name() string {
	return "bufferbloat"
}

func (b *Bufferbloat) Run(ctx context.Context) model.TestResult {
	result := baseResult(b.Name())
	result.StartedAt = time.Now()
	result.Status = StatusSkipped
	result.Metrics["download_url"] = b.cfg.Bufferbloat.DownloadURL
	result.Metrics["upload_url"] = b.cfg.Bufferbloat.UploadURL
	result.Findings = append(result.Findings, model.Finding{
		Severity: "INFO",
		Title:    "Bufferbloat test pending",
		Detail:   "Base version includes configuration for load tests; measurements will be added later.",
	})
	result.EndedAt = time.Now()
	return result
}
