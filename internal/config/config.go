package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFilename = "conncheck.yaml"

// Config holds user-tunable settings.
type Config struct {
	Mode        string        `yaml:"mode"`
	Privacy     string        `yaml:"privacy"`
	OutputDir   string        `yaml:"output_dir"`
	Targets     TargetsConfig `yaml:"targets"`
	Thresholds  Thresholds    `yaml:"thresholds"`
	Speedtest   Speedtest     `yaml:"speedtest"`
	HTTP        HTTPChecks    `yaml:"http"`
	Bufferbloat Bufferbloat   `yaml:"bufferbloat"`
}

type TargetsConfig struct {
	PingTargets []string `yaml:"ping_targets"`
	DNSServers  []string `yaml:"dns_servers"`
	Traceroute  []string `yaml:"traceroute_targets"`
	MTUTargets  []string `yaml:"mtu_targets"`
}

type Thresholds struct {
	PingWarnMs        int `yaml:"ping_warn_ms"`
	PingFailMs        int `yaml:"ping_fail_ms"`
	PacketLossWarnPct int `yaml:"packet_loss_warn_pct"`
	PacketLossFailPct int `yaml:"packet_loss_fail_pct"`
	BufferbloatWarnMs int `yaml:"bufferbloat_warn_ms"`
	BufferbloatFailMs int `yaml:"bufferbloat_fail_ms"`
}

type Speedtest struct {
	ServerIDs []int `yaml:"server_ids"`
}

type HTTPChecks struct {
	Endpoints []string `yaml:"endpoints"`
}

type Bufferbloat struct {
	DownloadURL string `yaml:"download_url"`
	UploadURL   string `yaml:"upload_url"`
}

func Default() Config {
	return Config{
		Mode:      "standard",
		Privacy:   "standard",
		OutputDir: "",
		Targets: TargetsConfig{
			PingTargets: []string{"1.1.1.1", "8.8.8.8"},
			DNSServers:  []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
			Traceroute:  []string{"1.1.1.1", "8.8.8.8"},
			MTUTargets:  []string{"1.1.1.1"},
		},
		Thresholds: Thresholds{
			PingWarnMs:        50,
			PingFailMs:        100,
			PacketLossWarnPct: 2,
			PacketLossFailPct: 5,
			BufferbloatWarnMs: 40,
			BufferbloatFailMs: 80,
		},
		Speedtest: Speedtest{
			ServerIDs: []int{},
		},
		HTTP: HTTPChecks{
			Endpoints: []string{"https://www.cloudflare.com"},
		},
		Bufferbloat: Bufferbloat{
			DownloadURL: "https://speed.hetzner.de/100MB.bin",
			UploadURL:   "",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = DefaultConfigFilename
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
