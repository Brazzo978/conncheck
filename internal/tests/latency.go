package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/sys"
)

var pingLatencyMatchers = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(time|tempo)[=<]\s*([\d.,]+)\s*ms`),
}

type Latency struct {
	outDir string
	cfg    config.Config
}

func NewLatency(outDir string, cfg config.Config) *Latency {
	return &Latency{outDir: outDir, cfg: cfg}
}

func (l *Latency) Name() string {
	return "latency"
}

func (l *Latency) Run(ctx context.Context) model.TestResult {
	result := baseResult(l.Name())
	result.StartedAt = time.Now()

	if len(l.cfg.Targets.PingTargets) == 0 {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "INFO",
			Title:    "No ping targets",
			Detail:   "No targets configured for latency checks.",
		})
		result.EndedAt = time.Now()
		return result
	}

	result.Status = StatusOK
	type targetResult struct {
		target  string
		samples []latencySample
		summary latencySummary
		err     error
	}

	results := make([]targetResult, 0, len(l.cfg.Targets.PingTargets))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, target := range l.cfg.Targets.PingTargets {
		target := target
		wg.Add(1)
		go func() {
			defer wg.Done()
			samples, summary, err := runLatencySeries(ctx, target)
			mu.Lock()
			results = append(results, targetResult{
				target:  target,
				samples: samples,
				summary: summary,
				err:     err,
			})
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, entry := range results {
		if entry.err != nil {
			result.Status = StatusWarn
			result.Findings = append(result.Findings, model.Finding{
				Severity: "WARN",
				Title:    "Latency sampling failed",
				Detail:   fmt.Sprintf("Latency sampling for %s failed: %s", entry.target, entry.err.Error()),
			})
			continue
		}

		seriesJSON, err := json.Marshal(entry.samples)
		if err == nil {
			result.Metrics[fmt.Sprintf("latency_series.%s", entry.target)] = string(seriesJSON)
		}
		result.Metrics[fmt.Sprintf("%s_avg_ms", entry.target)] = fmt.Sprintf("%d", entry.summary.AvgMs)
		result.Metrics[fmt.Sprintf("%s_min_ms", entry.target)] = fmt.Sprintf("%d", entry.summary.MinMs)
		result.Metrics[fmt.Sprintf("%s_max_ms", entry.target)] = fmt.Sprintf("%d", entry.summary.MaxMs)
		result.Metrics[fmt.Sprintf("%s_loss_pct", entry.target)] = fmt.Sprintf("%d", entry.summary.LossPct)
		if entry.summary.LossPct > 0 {
			result.Status = StatusWarn
		}
	}

	result.EndedAt = time.Now()
	return result
}

const (
	latencyProbeDuration = time.Minute
	latencyProbeInterval = 100 * time.Millisecond
)

type latencySample struct {
	OffsetMs  int  `json:"t"`
	LatencyMs int  `json:"latency"`
	Loss      bool `json:"loss"`
}

type latencySummary struct {
	AvgMs   int
	MinMs   int
	MaxMs   int
	LossPct int
}

func runLatencySeries(ctx context.Context, target string) ([]latencySample, latencySummary, error) {
	sampleCount := int(latencyProbeDuration / latencyProbeInterval)
	if sampleCount < 1 {
		sampleCount = 1
	}

	samples := make([]latencySample, 0, sampleCount)
	start := time.Now()
	var (
		successCount int
		sumLatency   int
		minLatency   int
		maxLatency   int
		lossCount    int
	)

	for i := 0; i < sampleCount; i++ {
		select {
		case <-ctx.Done():
			return samples, summarizeLatency(successCount, sumLatency, minLatency, maxLatency, lossCount, len(samples)), ctx.Err()
		default:
		}

		iterStart := time.Now()
		latencyMs, lost := pingSample(ctx, target)
		offsetMs := int(time.Since(start).Milliseconds())
		samples = append(samples, latencySample{
			OffsetMs:  offsetMs,
			LatencyMs: latencyMs,
			Loss:      lost,
		})

		if lost || latencyMs < 0 {
			lossCount++
		} else {
			successCount++
			sumLatency += latencyMs
			if minLatency == 0 || latencyMs < minLatency {
				minLatency = latencyMs
			}
			if latencyMs > maxLatency {
				maxLatency = latencyMs
			}
		}

		elapsed := time.Since(iterStart)
		if sleep := latencyProbeInterval - elapsed; sleep > 0 {
			time.Sleep(sleep)
		}
	}

	return samples, summarizeLatency(successCount, sumLatency, minLatency, maxLatency, lossCount, len(samples)), nil
}

func summarizeLatency(successCount, sumLatency, minLatency, maxLatency, lossCount, total int) latencySummary {
	avgMs := 0
	if successCount > 0 {
		avgMs = int(float64(sumLatency) / float64(successCount))
	}
	lossPct := 0
	if total > 0 {
		lossPct = int(float64(lossCount) / float64(total) * 100)
	}
	return latencySummary{
		AvgMs:   avgMs,
		MinMs:   minLatency,
		MaxMs:   maxLatency,
		LossPct: lossPct,
	}
}

func pingSample(ctx context.Context, target string) (int, bool) {
	timeout := 1200 * time.Millisecond
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := pingArgs(target)
	output, err := sys.RunCommandNoLog(pingCtx, "ping", args...)
	if latencyMs, ok := parsePingLatency(output); ok {
		return latencyMs, false
	}
	stats := ParsePing(output)
	if stats.Sent > 0 && stats.LossPct < 100 {
		return stats.AvgMs, false
	}
	if err != nil {
		return -1, true
	}
	return -1, true
}

func pingArgs(target string) []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"-n", "1", "-w", "1000", target}
	case "darwin":
		return []string{"-c", "1", "-W", "1000", target}
	default:
		return []string{"-c", "1", "-W", "1", target}
	}
}

func parsePingLatency(output string) (int, bool) {
	for _, matcher := range pingLatencyMatchers {
		if match := matcher.FindStringSubmatch(output); len(match) == 3 {
			return parseLatencyValue(match[2]), true
		}
	}
	return 0, false
}

func parseLatencyValue(value string) int {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", ".")
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int(parsed)
}
