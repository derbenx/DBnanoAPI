package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Window     fyne.Window

	DeleteHandler   func()
	OnImagesUpdated func()
	OnTasksUpdated  func()
	GlobalMode      string
}

func main() {
	a := app.NewWithID("com.nanogo.editor")
	w := a.NewWindow("Gemini 2026 Pro Editor (NanoGo)")
	w.Resize(fyne.NewSize(950, 700))

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		cfg = DefaultConfig()
	}

	state := &AppState{
		Config:     cfg,
		GlobalMode: "Immediate",
		Window:     w,
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
	logScroll.SetMinSize(fyne.NewSize(0, 30))
	state.LogScroll = logScroll

	mainSplit := container.NewVSplit(tabs, logScroll)
	mainSplit.Offset = 0.8
	w.SetContent(mainSplit)

	// Background Monitoring
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			if len(state.BatchJobs) == 0 {
				continue
			}
			state.Log("Checking batch statuses...")
			for _, job := range state.BatchJobs {
				if job.Status == "SUCCEEDED" || job.Status == "FAILED" || job.Status == "CANCELLED" {
					continue
				}
				err := state.CheckBatchStatus(job)
				if err != nil {
					state.Log("Status check error: " + err.Error())
				}
			}
			fyne.Do(func() {
				batchesTab.Refresh()
			})
		}
	}()

	w.SetOnDropped(func(pos fyne.Position, uris []fyne.URI) {
		for _, uri := range uris {
			ext := strings.ToLower(uri.Extension())
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
				state.AddImages([]string{uri.Path()})
			}
		}
		// Refresh UI and selection
		if state.OnImagesUpdated != nil {
			state.OnImagesUpdated()
		}
	})

	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyDelete || k.Name == fyne.KeyBackspace {
			// Don't delete items if user is typing in a text field
			focused := w.Canvas().Focused()
			if focused != nil {
				_, isEntry := focused.(*widget.Entry)
				_, isTabbable := focused.(*TabbableEntry)
				if isEntry || isTabbable {
					return
				}
			}

			if state.DeleteHandler != nil {
				state.DeleteHandler()
			}
		}
	})

	w.CenterOnScreen()
	w.ShowAndRun()
}



func makeHelpTab(state *AppState) fyne.CanvasObject {
	return widget.NewLabel("Nano Banana (Flash, pro and 2) can edit or generate multiple images and do batch.\nImagen is text to image only.")
}

func (s *AppState) Log(msg string) {
	timestamp := time.Now().Format("15:04:05")

	// UI Log
	fyne.Do(func() {
		s.ModelLog.SetText(s.ModelLog.Text + "\n[" + timestamp + "] " + msg)
		if s.LogScroll != nil {
			s.LogScroll.ScrollToBottom()
		}
	})

	// Debug.log file
	if s.Config.Debug {
		s.LogToFile(msg)
	}
}

func (s *AppState) LogToFile(msg string) {
	logPath := "debug.log"
	limit := int64(300 * 1024 * 1024)

	if info, err := os.Stat(logPath); err == nil && info.Size() > limit {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		os.Rename(logPath, "debug_"+timestamp+".log")
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString("[" + timestamp + "] " + msg + "\n")
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
}
