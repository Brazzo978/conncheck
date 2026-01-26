package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type Speedtest struct {
	outDir string
	cfg    config.Config
}

type speedtestResult struct {
	Ping struct {
		Latency float64 `json:"latency"`
	} `json:"ping"`
	Download struct {
		Bandwidth float64 `json:"bandwidth"`
	} `json:"download"`
	Upload struct {
		Bandwidth float64 `json:"bandwidth"`
	} `json:"upload"`
	Server struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"server"`
}

func NewSpeedtest(outDir string, cfg config.Config) *Speedtest {
	return &Speedtest{outDir: outDir, cfg: cfg}
}

func (s *Speedtest) Name() string {
	return "speedtest"
}

func (s *Speedtest) Run(ctx context.Context) model.TestResult {
	result := baseResult(s.Name())
	result.StartedAt = time.Now()

	binary, err := exec.LookPath("speedtest")
	if err != nil {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "INFO",
			Title:    "speedtest.exe not found",
			Detail:   "Place Ookla Speedtest CLI (speedtest.exe) next to the tool or in PATH.",
		})
		result.EndedAt = time.Now()
		return result
	}

	categories := []struct {
		name  string
		label string
		cfg   config.SpeedtestCategory
	}{
		{name: "local", label: "Local", cfg: s.cfg.Speedtest.Local},
		{name: "national", label: "National", cfg: s.cfg.Speedtest.National},
		{name: "eu", label: "EU", cfg: s.cfg.Speedtest.EU},
		{name: "us", label: "US", cfg: s.cfg.Speedtest.US},
	}

	if allSpeedtestCategoriesEmpty(categories) {
		categories = []struct {
			name  string
			label string
			cfg   config.SpeedtestCategory
		}{
			{name: "local", label: "Local", cfg: config.SpeedtestCategory{ServerIDs: []int{0}, Runs: 1, Weight: 1}},
		}
	}

	totalScore := 0.0
	totalWeight := 0.0
	for _, category := range categories {
		serverIDs := category.cfg.ServerIDs
		if len(serverIDs) == 0 {
			continue
		}
		runs := category.cfg.Runs
		if runs < 1 {
			runs = 1
		}
		weight := distanceWeight(category.name)
		result.Metrics[fmt.Sprintf("%s_runs", category.name)] = fmt.Sprintf("%d", runs)
		result.Metrics[fmt.Sprintf("%s_weight", category.name)] = fmt.Sprintf("%.2f", weight)

		categoryDownSum := 0.0
		categoryUpSum := 0.0
		categoryPingSum := 0.0
		categorySamples := 0

		for _, serverID := range serverIDs {
			for runIndex := 1; runIndex <= runs; runIndex++ {
				args := []string{"--format=json", "--accept-license", "--accept-gdpr"}
				if serverID > 0 {
					args = append(args, fmt.Sprintf("--server-id=%d", serverID))
				}
				output, logPath, err := sys.RunCommand(s.outDir, binary, args...)
				if logPath != "" {
					result.Evidence = append(result.Evidence, model.Evidence{
						Label: "speedtest_raw",
						Path:  logPath,
						Note:  fmt.Sprintf("%s server %d run %d", category.label, serverID, runIndex),
					})
				}
				if err != nil {
					result.Status = StatusWarn
					result.Findings = append(result.Findings, model.Finding{
						Severity: "WARN",
						Title:    "Speedtest failed",
						Detail:   fmt.Sprintf("%s server %d run %d: %s", category.label, serverID, runIndex, err.Error()),
					})
					continue
				}

				var parsed speedtestResult
				if parseErr := json.Unmarshal([]byte(output), &parsed); parseErr == nil {
					downBps := parsed.Download.Bandwidth * 8
					upBps := parsed.Upload.Bandwidth * 8
					categoryDownSum += downBps
					categoryUpSum += upBps
					categoryPingSum += parsed.Ping.Latency
					categorySamples++

					keyPrefix := fmt.Sprintf("%s_server_%d_run_%d", category.name, parsed.Server.ID, runIndex)
					result.Metrics[fmt.Sprintf("%s_ping_ms", keyPrefix)] = fmt.Sprintf("%.2f", parsed.Ping.Latency)
					result.Metrics[fmt.Sprintf("%s_down_bps", keyPrefix)] = fmt.Sprintf("%.0f", downBps)
					result.Metrics[fmt.Sprintf("%s_up_bps", keyPrefix)] = fmt.Sprintf("%.0f", upBps)
					result.Metrics[fmt.Sprintf("%s_name", keyPrefix)] = parsed.Server.Name
				}
			}
		}

		if categorySamples > 0 {
			avgDown := categoryDownSum / float64(categorySamples)
			avgUp := categoryUpSum / float64(categorySamples)
			avgPing := categoryPingSum / float64(categorySamples)
			result.Metrics[fmt.Sprintf("%s_avg_down_bps", category.name)] = fmt.Sprintf("%.0f", avgDown)
			result.Metrics[fmt.Sprintf("%s_avg_up_bps", category.name)] = fmt.Sprintf("%.0f", avgUp)
			result.Metrics[fmt.Sprintf("%s_avg_ping_ms", category.name)] = fmt.Sprintf("%.2f", avgPing)
			result.Metrics[fmt.Sprintf("%s_score_bps", category.name)] = fmt.Sprintf("%.0f", avgDown)

			if weight > 0 {
				totalScore += avgDown * weight
				totalWeight += weight
			}
		}
	}

	if totalWeight > 0 {
		result.Metrics["score_total_bps"] = fmt.Sprintf("%.0f", totalScore/totalWeight)
	}

	if result.Status == StatusSkipped {
		result.Status = StatusOK
	}
	if result.Status == StatusOK && len(result.Metrics) == 0 {
		result.Status = StatusWarn
	}

	result.EndedAt = time.Now()
	return result
}

func allSpeedtestCategoriesEmpty(categories []struct {
	name  string
	label string
	cfg   config.SpeedtestCategory
}) bool {
	for _, category := range categories {
		if len(category.cfg.ServerIDs) > 0 {
			return false
		}
	}
	return true
}

func distanceWeight(category string) float64 {
	switch category {
	case "local":
		return 1
	case "national":
		return 0.7
	case "eu":
		return 0.4
	case "us":
		return 0.2
	default:
		return 0.5
	}
}
