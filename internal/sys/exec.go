package sys

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func RunCommand(outDir, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	logName := fmt.Sprintf("%s_%d.log", filepath.Base(name), start.UnixNano())
	logPath := filepath.Join(outDir, "raw_logs", logName)
	_ = os.WriteFile(logPath, append(stdout.Bytes(), stderr.Bytes()...), 0o644)
	return stdout.String(), logPath, wrapErr(err, stderr.String(), duration)
}

func RunCommandNoLog(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	return stdout.String(), wrapErr(err, stderr.String(), duration)
}

func wrapErr(err error, stderr string, duration time.Duration) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("command failed after %s: %w (stderr: %s)", duration, err, stderr)
}
