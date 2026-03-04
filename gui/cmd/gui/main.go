package main

import (
	"flag"
	"log"

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
	_ = cfg

	app := ui.NewApp()
	app.Run()
}
