package tests

import (
	"context"
	"runtime"
	"strings"
	"time"

	"conncheck/internal/model"
	"conncheck/internal/sys"
)

type DualStack struct {
	outDir string
}

func NewDualStack(outDir string) *DualStack {
	return &DualStack{outDir: outDir}
}

func (d *DualStack) Name() string {
	return "dualstack"
}

func (d *DualStack) Run(ctx context.Context) model.TestResult {
	result := baseResult(d.Name())
	result.StartedAt = time.Now()

	ipv6Present := false
	if runtime.GOOS == "windows" {
		output, logPath, err := sys.RunCommand(d.outDir, "ipconfig")
		if logPath != "" {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "ipconfig", Path: logPath})
		}
		if err == nil && strings.Contains(output, "IPv6") {
			ipv6Present = true
		}
	} else {
		output, logPath, err := sys.RunCommand(d.outDir, "ip", "-6", "addr")
		if logPath != "" {
			result.Evidence = append(result.Evidence, model.Evidence{Label: "ip_addr_v6", Path: logPath})
		}
		if err == nil && strings.Contains(output, "inet6") {
			ipv6Present = true
		}
	}

	result.Metrics["ipv6_present"] = boolString(ipv6Present)
	if !ipv6Present {
		result.Status = StatusSkipped
		result.Findings = append(result.Findings, model.Finding{
			Severity: "INFO",
			Title:    "IPv6 not detected",
			Detail:   "No IPv6 addresses were found on active interfaces.",
		})
		result.EndedAt = time.Now()
		return result
	}

	ipv4OK := pingOnce(d.outDir, "1.1.1.1")
	ipv6OK := pingOnce(d.outDir, "2606:4700:4700::1111")
	dualOK := pingOnce(d.outDir, "google.com")

	result.Metrics["ipv4_reach"] = boolString(ipv4OK)
	result.Metrics["ipv6_reach"] = boolString(ipv6OK)
	result.Metrics["dualstack_reach"] = boolString(dualOK)

	result.Status = StatusOK
	if ipv6Present && !ipv6OK {
		result.Status = StatusWarn
		result.Findings = append(result.Findings, model.Finding{
			Severity: "WARN",
			Title:    "IPv6 appears broken",
			Detail:   "IPv6 is present but connectivity tests failed.",
		})
	}

	result.EndedAt = time.Now()
	return result
}

func pingOnce(outDir, target string) bool {
	var output string
	var err error
	if runtime.GOOS == "windows" {
		output, _, err = sys.RunCommand(outDir, "ping", "-n", "2", target)
	} else {
		output, _, err = sys.RunCommand(outDir, "ping", "-c", "2", target)
	}
	if err != nil {
		return false
	}
	return !strings.Contains(output, "Request timed out") && !strings.Contains(output, "100% packet loss")
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
