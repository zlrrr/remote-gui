//go:build fyne

package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/remote-gui/gui/internal/client"
	"github.com/remote-gui/gui/internal/config"
)

// App is the main GUI application (Fyne build).
type App struct {
	cfg        *config.Config
	client     client.ExecutorClient
	fyneApp    fyne.App
	mainWindow fyne.Window
}

// NewApp creates a new App with the given configuration and executor client.
func NewApp(cfg *config.Config, executorClient client.ExecutorClient) *App {
	a := fyneapp.New()
	w := a.NewWindow("remote-gui")
	w.Resize(fyne.NewSize(480, 560))
	return &App{cfg: cfg, client: executorClient, fyneApp: a, mainWindow: w}
}

// Run starts the Fyne event loop. Blocks until the window is closed.
func (a *App) Run() {
	a.mainWindow.SetContent(a.buildContent())
	a.mainWindow.ShowAndRun()
}

func (a *App) buildContent() fyne.CanvasObject {
	// Operation selector
	opNames := make([]string, len(a.cfg.Operations))
	for i, op := range a.cfg.Operations {
		opNames[i] = op.Alias
	}

	// Result area
	statusLabel := widget.NewLabel("")
	resultArea := widget.NewMultiLineEntry()
	resultArea.Disable()

	// Parameter entries (rebuilt when operation changes)
	paramContainer := container.NewVBox()
	var paramEntries []*widget.Entry

	updateParams := func(opIdx int) {
		paramContainer.Objects = nil
		paramEntries = nil
		if opIdx < 0 || opIdx >= len(a.cfg.Operations) {
			return
		}
		op := a.cfg.Operations[opIdx]
		for _, p := range op.Params {
			label := widget.NewLabel(p.Label + ":")
			entry := widget.NewEntry()
			entry.SetPlaceHolder(p.Placeholder)
			paramContainer.Add(label)
			paramContainer.Add(entry)
			paramEntries = append(paramEntries, entry)
		}
		paramContainer.Refresh()
	}

	selectedOp := 0
	if len(a.cfg.Operations) > 0 {
		updateParams(0)
	}

	opSelect := widget.NewSelect(opNames, func(selected string) {
		for i, op := range a.cfg.Operations {
			if op.Alias == selected {
				selectedOp = i
				updateParams(i)
				return
			}
		}
	})
	if len(opNames) > 0 {
		opSelect.SetSelected(opNames[0])
	}

	// Execute button
	execBtn := widget.NewButton("执行", func() {
		if selectedOp < 0 || selectedOp >= len(a.cfg.Operations) {
			return
		}
		op := a.cfg.Operations[selectedOp]
		params := make(map[string]string)
		for i, p := range op.Params {
			if i < len(paramEntries) {
				params[p.Name] = paramEntries[i].Text
			}
		}

		statusLabel.SetText("执行中...")
		resultArea.SetText("")

		result, err := a.client.Execute(op.Script, params)
		if err != nil {
			statusLabel.SetText("错误: " + err.Error())
			return
		}

		statusLabel.SetText(fmt.Sprintf("状态: %s  退出码: %d  耗时: %dms",
			result.Status, result.ExitCode, result.DurationMs))

		var sb strings.Builder
		if result.Stdout != "" {
			sb.WriteString(result.Stdout)
		}
		if result.Stderr != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n--- stderr ---\n")
			}
			sb.WriteString(result.Stderr)
		}
		resultArea.SetText(sb.String())
	})

	divider := widget.NewSeparator()
	resultLabel := widget.NewLabel("─────────── 结果 ───────────")

	return container.NewVBox(
		widget.NewLabel("操作:"),
		opSelect,
		paramContainer,
		execBtn,
		divider,
		resultLabel,
		statusLabel,
		resultArea,
	)
}
