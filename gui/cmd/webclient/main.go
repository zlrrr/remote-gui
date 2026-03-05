package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/remote-gui/gui/internal/webui"
)

func main() {
	configPath := flag.String("config", "configs/gui.yaml", "path to gui.yaml config file")
	port := flag.Int("port", 0, "port to listen on (0 = random)")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	mux := webui.NewServer(*configPath)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		slog.Error("failed to start listener", "error", err)
		os.Exit(1)
	}

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
	fmt.Printf("Remote GUI Web Client\n")
	fmt.Printf("Serving at %s\n", url)
	fmt.Printf("Press Ctrl+C to quit.\n\n")

	openBrowser(url)

	if err := http.Serve(listener, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		slog.Warn("could not open browser automatically", "url", url, "error", err)
	}
}
