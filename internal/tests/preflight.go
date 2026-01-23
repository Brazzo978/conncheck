package tests

import (
	"context"
	"runtime"
	"strconv"
	"time"

	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type Preflight struct {
	outDir string
}

func NewPreflight(outDir string) *Preflight {
	return &Preflight{outDir: outDir}
}

func (p *Preflight) Name() string {
	return "preflight"
}

func (p *Preflight) Run(ctx context.Context) model.TestResult {
	result := baseResult(p.Name())
	result.StartedAt = time.Now()

	if runtime.GOOS == "windows" {
		output, logPath, err := sys.RunCommand(p.outDir, "ipconfig", "/all")
		if err == nil {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "ipconfig", Path: logPath})
			result.Metrics["ipconfig_bytes"] = strconv.Itoa(len(output))
		} else {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "ipconfig", Path: logPath, Note: err.Error()})
		}
		if _, routeLog, err := sys.RunCommand(p.outDir, "route", "print"); err == nil {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "route_print", Path: routeLog})
		} else {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "route_print", Path: routeLog, Note: err.Error()})
		}
		result.Status = StatusOK
	} else {
		if _, logPath, err := sys.RunCommand(p.outDir, "ip", "addr"); err == nil {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "ip_addr", Path: logPath})
		}
		if _, logPath, err := sys.RunCommand(p.outDir, "ip", "route"); err == nil {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "ip_route", Path: logPath})
		}
		result.Status = StatusOK
	}

	result.EndedAt = time.Now()
	return result
}
