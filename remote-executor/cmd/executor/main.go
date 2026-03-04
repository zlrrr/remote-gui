package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/remote-gui/remote-executor/internal/config"
	"github.com/remote-gui/remote-executor/internal/server"
)

func main() {
	configPath := flag.String("config", "configs/executor.yaml", "path to config file")
	scriptsDir := flag.String("scripts-dir", "scripts", "path to scripts directory")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	_ = cfg
	_ = scriptsDir

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	if addr == ":" {
		addr = ":8443"
	}

	srv := server.New(server.Config{
		Addr:       addr,
		CACert:     cfg.TLS.CACert,
		ServerCert: cfg.TLS.ServerCert,
		ServerKey:  cfg.TLS.ServerKey,
	})

	log.Printf("starting remote-executor on %s", addr)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
