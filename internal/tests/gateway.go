package tests

import (
	"runtime"
	"strings"

	"conncheck/internal/sys"
)

func DetectDefaultGateway(outDir string) (string, string, error) {
	if runtime.GOOS == "windows" {
		output, logPath, err := sys.RunCommand(outDir, "route", "print", "-4")
		if err != nil {
			return "", logPath, err
		}
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
				return fields[2], logPath, nil
			}
		}
		return "", logPath, nil
	}
	output, logPath, err := sys.RunCommand(outDir, "ip", "route")
	if err != nil {
		return "", logPath, err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "default" && fields[1] == "via" {
			return fields[2], logPath, nil
		}
	}
	return "", logPath, nil
}
