package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/remote-gui/remote-executor/internal/config"
	"github.com/remote-gui/remote-executor/internal/record"
	"github.com/remote-gui/remote-executor/internal/runner"
	"github.com/remote-gui/remote-executor/internal/script"
	"github.com/remote-gui/remote-executor/internal/server"
)

func main() {
	configPath := flag.String("config", "configs/executor.yaml", "path to config file")
	scriptsDir := flag.String("scripts-dir", "", "override scripts directory from config")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	scriptsPath := cfg.Scripts.Dir
	if *scriptsDir != "" {
		scriptsPath = *scriptsDir
	}

	registry, err := script.LoadScripts(scriptsPath)
	if err != nil {
		log.Fatalf("failed to load scripts from %q: %v", scriptsPath, err)
	}
	log.Printf("loaded %d scripts from %q", len(registry), scriptsPath)

	store := record.NewFileStore(cfg.Records.Dir)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	if addr == ":" {
		addr = ":8443"
	}

	srv := server.New(server.Config{
		Addr:       addr,
		CACert:     cfg.TLS.CACert,
		ServerCert: cfg.TLS.ServerCert,
		ServerKey:  cfg.TLS.ServerKey,
		Registry:   registry,
		Runner:     runner.NewRunner(),
		Store:      store,
	})

	log.Printf("starting remote-executor on %s", addr)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
