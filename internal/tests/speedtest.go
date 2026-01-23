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

	serverIDs := s.cfg.Speedtest.ServerIDs
	if len(serverIDs) == 0 {
		serverIDs = []int{0}
	}

	for _, serverID := range serverIDs {
		args := []string{"--format=json", "--accept-license", "--accept-gdpr"}
		if serverID > 0 {
			args = append(args, fmt.Sprintf("--server-id=%d", serverID))
		}
		output, logPath, err := sys.RunCommand(s.outDir, binary, args...)
		if logPath != "" {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "speedtest_raw", Path: logPath})
		}
		if err != nil {
			result.Status = StatusWarn
			result.Findings = append(result.Findings, model.Finding{
				Severity: "WARN",
				Title:    "Speedtest failed",
				Detail:   err.Error(),
			})
			continue
		}

		var parsed speedtestResult
		if parseErr := json.Unmarshal([]byte(output), &parsed); parseErr == nil {
			result.Metrics[fmt.Sprintf("server_%d_ping_ms", parsed.Server.ID)] = fmt.Sprintf("%.2f", parsed.Ping.Latency)
			result.Metrics[fmt.Sprintf("server_%d_down_bps", parsed.Server.ID)] = fmt.Sprintf("%.0f", parsed.Download.Bandwidth*8)
			result.Metrics[fmt.Sprintf("server_%d_up_bps", parsed.Server.ID)] = fmt.Sprintf("%.0f", parsed.Upload.Bandwidth*8)
		}
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
