package config

import (
	"fmt"
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
	Local    SpeedtestCategory `yaml:"local"`
	National SpeedtestCategory `yaml:"national"`
	EU       SpeedtestCategory `yaml:"eu"`
	US       SpeedtestCategory `yaml:"us"`
}

type SpeedtestCategory struct {
	ServerIDs []int   `yaml:"server_ids"`
	Runs      int     `yaml:"runs"`
	Weight    float64 `yaml:"weight"`
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
			Local: SpeedtestCategory{
				ServerIDs: []int{},
				Runs:      2,
				Weight:    1,
			},
			National: SpeedtestCategory{
				ServerIDs: []int{},
				Runs:      2,
				Weight:    1,
			},
			EU: SpeedtestCategory{
				ServerIDs: []int{},
				Runs:      1,
				Weight:    1,
			},
			US: SpeedtestCategory{
				ServerIDs: []int{},
				Runs:      1,
				Weight:    1,
			},
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
	if path == "" {
		path = DefaultConfigFilename
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config file not found: %s", path)
		}
		return Config{}, err
	}
	cfg := Config{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
