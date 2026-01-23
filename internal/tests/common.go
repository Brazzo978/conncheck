package tests

import "conncheck/internal/model"

const (
	StatusOK      = "OK"
	StatusWarn    = "WARN"
	StatusFail    = "FAIL"
	StatusSkipped = "SKIPPED"
)

func baseResult(name string) model.TestResult {
	return model.TestResult{
		Name:     name,
		Status:   StatusSkipped,
		Metrics:  map[string]string{},
		Findings: []model.Finding{},
		Evidence: []model.Evidence{},
	}
}
