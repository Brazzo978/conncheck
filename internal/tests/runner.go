package tests

import (
	"context"

	"conncheck/internal/model"
)

type Runner interface {
	Name() string
	Run(ctx context.Context) model.TestResult
}
