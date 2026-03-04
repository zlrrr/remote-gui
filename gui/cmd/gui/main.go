package main

import (
	"flag"
	"log"

	"github.com/remote-gui/gui/internal/client"
	"github.com/remote-gui/gui/internal/config"
	"github.com/remote-gui/gui/internal/ui"
)

func main() {
	configPath := flag.String("config", "configs/gui.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	executorClient, err := client.NewExecutorClient(client.ExecutorClientConfig{
		Endpoint:   cfg.Executor.Endpoint,
		CACert:     cfg.Executor.TLS.CACert,
		ClientCert: cfg.Executor.TLS.ClientCert,
		ClientKey:  cfg.Executor.TLS.ClientKey,
	})
	if err != nil {
		log.Fatalf("failed to create executor client: %v", err)
	}

	app := ui.NewApp(cfg, executorClient)
	app.Run()
}
