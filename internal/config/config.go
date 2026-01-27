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
	SpeedtestUI SpeedtestUI   `yaml:"speedtest_ui"`
	HTTP        HTTPChecks    `yaml:"http"`
	Bufferbloat Bufferbloat   `yaml:"bufferbloat"`
}

type TargetsConfig struct {
	PingTargets []string `yaml:"ping_targets"`
	DNSServers  []string `yaml:"dns_servers"`
	DNSDomains  []string `yaml:"dns_domains"`
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

type SpeedtestUI struct {
	DownloadScale []SpeedtestScale `yaml:"download_scale"`
	UploadScale   []SpeedtestScale `yaml:"upload_scale"`
	Comparisons   SpeedtestCompare `yaml:"comparisons"`
}

type SpeedtestScale struct {
	MinMbps     float64 `yaml:"min_mbps"`
	MaxMbps     float64 `yaml:"max_mbps"`
	Label       string  `yaml:"label"`
	Description string  `yaml:"description"`
}

type SpeedtestCompare struct {
	NationalPct float64 `yaml:"national_pct"`
	EUPct       float64 `yaml:"eu_pct"`
	USPct       float64 `yaml:"us_pct"`
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
			DNSDomains:  []string{"www.google.com", "www.cloudflare.com", "www.wikipedia.org"},
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
		SpeedtestUI: SpeedtestUI{
			DownloadScale: []SpeedtestScale{
				{
					MinMbps:     0,
					MaxMbps:     20,
					Label:       "Sofferenza 🐢",
					Description: "ADSL / 3G / sat base / FWA molto distante",
				},
				{
					MinMbps:     20,
					MaxMbps:     200,
					Label:       "Si vive 🚶‍♂️",
					Description: "VDSL / 4G medio / FWA ok / sat decente",
				},
				{
					MinMbps:     200,
					MaxMbps:     600,
					Label:       "IDEALE ⭐🚀",
					Description: "Uso normale + streaming + call + gaming senza pensieri",
				},
				{
					MinMbps:     600,
					MaxMbps:     1000,
					Label:       "Scheggia ⚡",
					Description: "5G serio / FTTH buona",
				},
				{
					MinMbps:     1000,
					MaxMbps:     2300,
					Label:       "Fibra Ottima 👑",
					Description: "FTTH 1–2.5G",
				},
				{
					MinMbps:     2300,
					MaxMbps:     0,
					Label:       "Fibra Divina 🔱✨",
					Description: "FTTH 2.5G+ / roba da datacenter a casa",
				},
			},
			UploadScale: []SpeedtestScale{
				{
					MinMbps:     0,
					MaxMbps:     2,
					Label:       "Sofferenza 🐢",
					Description: "ADSL / sat base / 4G scarso / FWA lontana",
				},
				{
					MinMbps:     2,
					MaxMbps:     20,
					Label:       "Si vive 🚶‍♂️",
					Description: "VDSL / 4G medio / FWA ok / sat decente",
				},
				{
					MinMbps:     20,
					MaxMbps:     100,
					Label:       "Spinge 🚗💨",
					Description: "4G top / 5G meh / Starlink / FWA buona",
				},
				{
					MinMbps:     100,
					MaxMbps:     300,
					Label:       "IDEALE ⭐🚀",
					Description: "Upload adatto ad ogni uso",
				},
				{
					MinMbps:     300,
					MaxMbps:     500,
					Label:       "FTTH Buona ⚡",
					Description: "Profili fibra moderni (300/500 up)",
				},
				{
					MinMbps:     500,
					MaxMbps:     1000,
					Label:       "FTTH Ottima 👑",
					Description: "500/1000 up",
				},
				{
					MinMbps:     1000,
					MaxMbps:     2500,
					Label:       "FTTH Esagerata 🦾",
					Description: "1–2.5G up",
				},
				{
					MinMbps:     2500,
					MaxMbps:     0,
					Label:       "FTTH Divina 🔱✨",
					Description: "10G (fino a ~10.000 up)",
				},
			},
			Comparisons: SpeedtestCompare{
				NationalPct: 93,
				EUPct:       55,
				USPct:       35,
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
