package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	ModelLog   *widget.Label
	LogScroll  *container.Scroll

	DeleteHandler   func()
	OnImagesUpdated func()
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
	state.ModelLog = widget.NewLabel("To start, drag and drop an image or click 'New Image'...")
	state.ModelLog.Wrapping = fyne.TextWrapBreak

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

	logScroll := container.NewVScroll(state.ModelLog)
	logScroll.SetMinSize(fyne.NewSize(0, 150))
	state.LogScroll = logScroll

	content := container.NewBorder(nil, logScroll, nil, nil, tabs)
	w.SetContent(content)

	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		for _, uri := range uris {
			ext := uri.Extension()
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
				state.AddImages([]string{uri.Path()})
			}
		}
	})

	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyDelete {
			// Don't delete items if user is typing in a text field
			focused := w.Canvas().Focused()
			if _, ok := focused.(*widget.Entry); ok {
				return
			}
			if _, ok := focused.(*TabbableEntry); ok {
				return
			}

			if state.DeleteHandler != nil {
				state.DeleteHandler()
			}
		}
	})

	w.ShowAndRun()
}



func makeHelpTab(state *AppState) fyne.CanvasObject {
	return widget.NewLabel("Nano Banana (Flash, pro and 2) can edit or generate multiple images and do batch.\nImagen is text to image only.")
}

func (s *AppState) Log(msg string) {
	timestamp := time.Now().Format("15:04:05")
	fyne.Do(func() {
		s.ModelLog.SetText(s.ModelLog.Text + "\n[" + timestamp + "] " + msg)
		if s.LogScroll != nil {
			s.LogScroll.ScrollToBottom()
		}
	})
}

func (s *AppState) AddImages(paths []string) {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", len(s.Images)+1),
			FileName: filepath.Base(p),
			FullPath: p,
			SizeMB:   float64(info.Size()) / 1024 / 1024,
		})
	}
	if s.OnImagesUpdated != nil {
		s.OnImagesUpdated()
	}
}
