package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"github.com/moody2real/merc-vcc-gen/internal/cli"
	"github.com/moody2real/merc-vcc-gen/internal/config"
	"github.com/moody2real/merc-vcc-gen/internal/fluzcli"
	"github.com/moody2real/merc-vcc-gen/internal/setup"
	"github.com/moody2real/merc-vcc-gen/pkg/fluz"
)

const configPath = "config.json"

type runner interface {
	EnsureSession() error
	Run() error
}

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Level:           log.InfoLevel,
	})
	log.SetDefault(logger)

	cfg, err := loadOrSetup()
	if err != nil {
		log.Fatal("config error", "err", err)
	}
	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	outputPath := filepath.Join("data", "cards_"+time.Now().Format("20060102-150405")+".txt")
	if err := os.MkdirAll("data", 0o700); err != nil {
		log.Fatal("create data dir", "err", err)
	}
	log.Info("output file for this run", "path", outputPath)

	app, err := build(cfg, outputPath)
	if err != nil {
		log.Fatal("startup error", "err", err)
	}

	if err := app.EnsureSession(); err != nil {
		log.Fatal("session error", "err", err)
	}
	if err := app.Run(); err != nil {
		log.Fatal("fatal", "err", err)
	}

	fmt.Println("Goodbye.")
}

func loadOrSetup() (*config.Config, error) {
	if setup.Needed(configPath) {
		cfg, err := setup.Run(configPath)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func build(cfg *config.Config, outputPath string) (runner, error) {
	switch strings.ToLower(cfg.Provider) {
	case "mercury":
		return cli.New(cfg, "session.json", outputPath), nil
	case "fluz":
		client := fluz.New(fluz.Config{
			APIKey:    cfg.Fluz.APIKey,
			UserID:    cfg.Fluz.UserID,
			AccountID: cfg.Fluz.AccountID,
			SeatID:    cfg.Fluz.SeatID,
		})
		return fluzcli.New(cfg, client, outputPath), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}
