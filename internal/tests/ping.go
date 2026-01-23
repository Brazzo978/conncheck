package tests

import (
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type PingStats struct {
	Sent     int
	Received int
	LossPct  int
	MinMs    int
	MaxMs    int
	AvgMs    int
}

var (
	winPingLossRe = regexp.MustCompile(`Lost = (\d+)`) // nolint:lll
	winPingSentRe = regexp.MustCompile(`Sent = (\d+)`) // nolint:lll
	winPingRttRe  = regexp.MustCompile(`Minimum = (\d+)ms, Maximum = (\d+)ms, Average = (\d+)ms`)
	unixPingRttRe = regexp.MustCompile(`= ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)
	unixPingLoss  = regexp.MustCompile(`(\d+)% packet loss`)
	unixPingSent  = regexp.MustCompile(`(\d+) packets transmitted, (\d+) received`)
)

func ParsePing(output string) PingStats {
	stats := PingStats{}
	if runtime.GOOS == "windows" {
		if m := winPingSentRe.FindStringSubmatch(output); len(m) == 2 {
			stats.Sent, _ = strconv.Atoi(m[1])
		}
		if m := winPingLossRe.FindStringSubmatch(output); len(m) == 2 {
			loss, _ := strconv.Atoi(m[1])
			stats.LossPct = loss
			if stats.Sent > 0 {
				stats.Received = stats.Sent - loss
			}
		}
		if m := winPingRttRe.FindStringSubmatch(output); len(m) == 4 {
			stats.MinMs, _ = strconv.Atoi(m[1])
			stats.MaxMs, _ = strconv.Atoi(m[2])
			stats.AvgMs, _ = strconv.Atoi(m[3])
		}
		return stats
	}

	if m := unixPingSent.FindStringSubmatch(output); len(m) == 3 {
		stats.Sent, _ = strconv.Atoi(m[1])
		stats.Received, _ = strconv.Atoi(m[2])
	}
	if m := unixPingLoss.FindStringSubmatch(output); len(m) == 2 {
		stats.LossPct, _ = strconv.Atoi(m[1])
	}
	if m := unixPingRttRe.FindStringSubmatch(output); len(m) == 5 {
		stats.MinMs = toInt(m[1])
		stats.AvgMs = toInt(m[2])
		stats.MaxMs = toInt(m[3])
	}
	return stats
}

func toInt(value string) int {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) == 0 {
		return 0
	}
	v, _ := strconv.Atoi(parts[0])
	return v
}
