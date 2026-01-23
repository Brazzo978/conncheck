package tests

import (
	"os"
	"runtime"
	"time"

	"conncheck/internal/model"
)

func CollectEnvironment() model.Environment {
	hostname, _ := os.Hostname()
	return model.Environment{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname,
		Timezone: time.Now().Format("MST"),
	}
}
