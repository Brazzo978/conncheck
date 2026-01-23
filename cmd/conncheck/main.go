package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"conncheck/internal/config"
	"conncheck/internal/engine"
	"conncheck/internal/report"
)

func main() {
	var (
		configPath string
		outDir     string
		noUI       bool
	)
	flag.StringVar(&configPath, "config", "", "Path to conncheck.yaml")
	flag.StringVar(&outDir, "out", "", "Output directory (default: ./outputs/<timestamp>)")
	flag.BoolVar(&noUI, "no-ui", false, "Disable UI (CLI only)")
	flag.Parse()

	logger := log.New(os.Stdout, "conncheck: ", log.LstdFlags)
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatalf("config load failed: %v", err)
	}
	if outDir == "" {
		outDir = filepath.Join("outputs", time.Now().Format("20060102-150405"))
	}
	if cfg.OutputDir != "" {
		outDir = cfg.OutputDir
	}

	if err := os.MkdirAll(filepath.Join(outDir, "raw_logs"), 0o755); err != nil {
		logger.Fatalf("failed to create output dir: %v", err)
	}

	if !noUI {
		logger.Println("UI mode placeholder: status window will be added later")
	}

	engine := engine.Engine{Cfg: cfg, Logger: logger, OutDir: outDir}
	ctx := context.Background()
	result, err := engine.Run(ctx)
	if err != nil {
		logger.Fatalf("run failed: %v", err)
	}

	jsonPath, err := report.WriteJSON(outDir, result)
	if err != nil {
		logger.Fatalf("write json failed: %v", err)
	}
	xmlPath, err := report.WriteXML(outDir, result)
	if err != nil {
		logger.Fatalf("write xml failed: %v", err)
	}
	htmlPath, err := report.WriteHTML(outDir, result)
	if err != nil {
		logger.Fatalf("write html failed: %v", err)
	}

	logger.Println("Outputs generated:")
	logger.Printf("- %s\n- %s\n- %s", jsonPath, xmlPath, htmlPath)
	logger.Printf("Summary: %s", report.FormatSummary(result))
	logger.Println("Done.")

	fmt.Println()
}
