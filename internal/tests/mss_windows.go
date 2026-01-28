//go:build windows

package tests

import (
	"context"
	"errors"
)

func observeMSSWithSocket(ctx context.Context, target, network string) (int, error) {
	return 0, errors.New("MSS observation not supported on Windows")
}
