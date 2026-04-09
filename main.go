package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type AppState struct {
	Config     *Config
	Images     []*ImageInfo
	Tasks      []*TaskInfo
	BatchJobs  []*BatchJob
	ModelLog   *widget.Entry
}

func main() {
	a := app.New()
	w := a.NewWindow("Gemini 2026 Pro Editor (NanoGo)")
	w.Resize(fyne.NewSize(950, 700))

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		cfg = DefaultConfig()
	}

	state := &AppState{
		Config: cfg,
	}

	// Shared Log
	state.ModelLog = widget.NewMultiLineEntry()
	state.ModelLog.SetReadOnly(true)
	state.ModelLog.SetPlaceHolder("To start, drag and drop an image or click 'New Image'...")

	// Tabs
	createTab := state.makeCreateTab()
	batchesTab := makeBatchesTab(state)
	settingsTab := makeSettingsTab(state)
	helpTab := makeHelpTab(state)

	tabs := container.NewAppTabs(
		container.NewTabItem("Create", createTab),
		container.NewTabItem("Batches", batchesTab),
		container.NewTabItem("Settings", settingsTab),
		container.NewTabItem("Help", helpTab),
	)

	content := container.NewBorder(nil, state.ModelLog, nil, nil, tabs)
	w.SetContent(content)

	w.ShowAndRun()
}



func makeHelpTab(state *AppState) fyne.CanvasObject {
	return widget.NewLabel("Nano Banana (Flash, pro and 2) can edit or generate multiple images and do batch.\nImagen is text to image only.")
}

func (s *AppState) Log(msg string) {
	timestamp := time.Now().Format("15:04:05")
	s.ModelLog.SetText(s.ModelLog.Text + "\n[" + timestamp + "] " + msg)
	s.ModelLog.CursorRow = len(s.ModelLog.Text) // Simple scroll to bottom
}
