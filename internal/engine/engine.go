package engine

import (
	"context"
	"fmt"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/tests"
)

type Logger interface {
	Printf(format string, args ...any)
}

type Engine struct {
	Cfg    config.Config
	Logger Logger
	OutDir string
}

func (e *Engine) Run(ctx context.Context) (model.Result, error) {
	result := model.Result{
		Version:   model.Version,
		StartedAt: time.Now(),
		Summary: model.Summary{
			StatusCounts: model.IntMap{},
		},
	}

	testList := []tests.Runner{
		tests.NewPreflight(e.OutDir),
		tests.NewLAN(e.OutDir),
		tests.NewDualStack(e.OutDir),
		tests.NewDNSBench(e.OutDir, e.Cfg),
		tests.NewMTU(e.OutDir, e.Cfg),
		tests.NewLatency(e.OutDir, e.Cfg),
		tests.NewBufferbloat(e.OutDir, e.Cfg),
		tests.NewSpeedtest(e.OutDir, e.Cfg),
		tests.NewTraceroute(e.OutDir, e.Cfg),
		tests.NewHTTPCheck(e.OutDir, e.Cfg),
	}

	for _, test := range testList {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		if !e.Cfg.Tests.IsEnabled(test.Name()) {
			e.log("Skipping %s (disabled in config).", test.Name())
			continue
		}

		e.log("Running %s...", test.Name())
		res := test.Run(ctx)
		result.Tests = append(result.Tests, res)
		result.Summary.StatusCounts[res.Status] = result.Summary.StatusCounts[res.Status] + 1
		result.Findings = append(result.Findings, res.Findings...)
	}

	result.Environment = tests.CollectEnvironment()
	result.FinishedAt = time.Now()

	if len(result.Tests) == 0 {
		return result, fmt.Errorf("no tests executed")
	}

	return result, nil
}

func (e *Engine) log(format string, args ...any) {
	if e.Logger == nil {
		return
	}
	e.Logger.Printf(format, args...)
}
