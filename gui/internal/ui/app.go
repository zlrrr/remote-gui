//go:build !fyne

package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/remote-gui/gui/internal/client"
	"github.com/remote-gui/gui/internal/config"
)

// App is the main GUI application.
type App struct {
	cfg    *config.Config
	client client.ExecutorClient
}

// NewApp creates a new App with the given configuration and executor client.
func NewApp(cfg *config.Config, executorClient client.ExecutorClient) *App {
	return &App{cfg: cfg, client: executorClient}
}

// Run starts the application.
// In non-fyne mode this is a simple interactive CLI.
// Build with -tags fyne for the full Fyne desktop UI.
func (a *App) Run() {
	fmt.Println("remote-gui (CLI mode — build with -tags fyne for desktop UI)")
	fmt.Println("Available operations:")
	for i, op := range a.cfg.Operations {
		fmt.Printf("  [%d] %s → %s\n", i+1, op.Alias, op.Script)
	}
	fmt.Println()

	// Simple interactive loop
	for {
		fmt.Print("Enter operation number (or 'q' to quit): ")
		var input string
		fmt.Scanln(&input)
		if strings.TrimSpace(input) == "q" {
			fmt.Println("Goodbye.")
			os.Exit(0)
		}

		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(a.cfg.Operations) {
			fmt.Println("Invalid selection.")
			continue
		}

		op := a.cfg.Operations[idx-1]
		params := make(map[string]string)

		for _, p := range op.Params {
			fmt.Printf("  %s [%s]: ", p.Label, p.Placeholder)
			var val string
			fmt.Scanln(&val)
			params[p.Name] = val
		}

		fmt.Printf("Executing '%s'...\n", op.Alias)
		result, err := a.client.Execute(op.Script, params)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Printf("Status: %s  ExitCode: %d  Duration: %dms\n", result.Status, result.ExitCode, result.DurationMs)
		if result.Stdout != "" {
			fmt.Println("--- stdout ---")
			fmt.Println(result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Println("--- stderr ---")
			fmt.Println(result.Stderr)
		}
	}
}
