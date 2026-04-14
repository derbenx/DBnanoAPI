package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type AppState struct {
	Config     *Config
	Images     []*ImageInfo
	Tasks      []*TaskInfo
	BatchJobs  []*BatchJob
	ModelLog   *widget.List
	LogLines   []string
	LogScroll  *container.Scroll
	Window     fyne.Window

	BatchProgressBar *widget.ProgressBar
	BatchStatusLabel *widget.Label
	BatchList        *widget.List

	NextImageID int
	NextTaskID  int

	DeleteHandler   func()
	OnImagesUpdated func()
	OnTasksUpdated  func()
	GlobalMode      string

	BatchMonitorIndex int

	MainSplit *container.Split
	LeftSplit *container.Split
	TopSplit  *container.Split
	LogSplit  *container.Split
}

func main() {
	a := app.NewWithID("com.nanogo.editor")
	w := a.NewWindow("Gemini 2026 Pro Editor (NanoGo)")

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		cfg = DefaultConfig()
	}

	state := &AppState{
		Config:      cfg,
		GlobalMode:  "Immediate",
		Window:      w,
		NextImageID: 1,
		NextTaskID:  1,
	}

	// Shared Log
	state.LogLines = []string{"To start, drag and drop an image or click 'New Image'..."}
	state.ModelLog = widget.NewList(
		func() int { return len(state.LogLines) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Wrapping = fyne.TextWrapBreak
			return l
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(state.LogLines[id])
		},
	)
	state.LoadJobs()

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
	tabs.OnSelected = func(t *container.TabItem) {
		// Update app state or just let the index handle it
	}

	logSplit := container.NewVSplit(tabs, state.ModelLog)
	logSplit.Offset = state.Config.LogSplitOffset
	state.LogSplit = logSplit
	w.SetContent(logSplit)

	w.Resize(fyne.NewSize(state.Config.WindowWidth, state.Config.WindowHeight))
	if state.Config.IsMaximized {
		w.Maximize()
	} else {
		w.CenterOnScreen()
	}

	// Background Monitoring
	go func() {
		const interval = 2 * time.Minute
		nextCheck := time.Now().Add(interval)
		ticker := time.NewTicker(1 * time.Second)
		lastJobsLen := -1
		for range ticker.C {
			jobsLen := len(state.BatchJobs)
			if jobsLen == 0 {
				if state.BatchProgressBar != nil && (lastJobsLen != 0) {
					state.BatchProgressBar.SetValue(0)
					state.BatchStatusLabel.SetText("No active batch jobs.")
					lastJobsLen = 0
				}
				continue
			}
			lastJobsLen = jobsLen

			remaining := time.Until(nextCheck)
			if remaining <= 0 {
				// 1. Identify active jobs
				var activeJobs []*BatchJob
				for _, job := range state.BatchJobs {
					s := job.Status
					if s != "SUCCEEDED" && s != "FAILED" && s != "CANCELLED" && s != "EXPIRED" && s != "Success" && s != "Failed" {
						activeJobs = append(activeJobs, job)
					}
				}

				if len(activeJobs) > 0 {
					// 2. Cycle to the next job
					if state.BatchMonitorIndex >= len(activeJobs) {
						state.BatchMonitorIndex = 0
					}
					target := activeJobs[state.BatchMonitorIndex]
					state.BatchMonitorIndex++

					// 3. Check only that one job
					state.Log(fmt.Sprintf("Checking status for job: %s", target.JobID))
					err := state.CheckBatchStatus(target)
					if err != nil {
						state.Log("Status check error: " + err.Error())
					}

					if state.BatchList != nil {
						state.BatchList.Refresh()
					}
				}

				nextCheck = time.Now().Add(interval)
				remaining = interval
			}

			if state.BatchProgressBar != nil {
				elapsed := interval - remaining
				state.BatchProgressBar.SetValue(float64(elapsed) / float64(interval))
				newStatus := fmt.Sprintf("Next status check in %d seconds...", int(remaining.Seconds()))
				if state.BatchStatusLabel.Text != newStatus {
					state.BatchStatusLabel.SetText(newStatus)
				}
			}
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

	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		current := tabs.SelectedIndex()
		current++
		if current >= len(tabs.Items) {
			current = 0
		}
		tabs.SelectIndex(current)
	})

	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyTab, Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift}, func(shortcut fyne.Shortcut) {
		current := tabs.SelectedIndex()
		current--
		if current < 0 {
			current = len(tabs.Items) - 1
		}
		tabs.SelectIndex(current)
	})

	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {

		focused := w.Canvas().Focused()

		// List Arrow Navigation
		if k.Name == fyne.KeyUp || k.Name == fyne.KeyDown {
			if list, ok := focused.(*widget.List); ok {
				selected := list.Selected
				if len(selected) > 0 {
					current := selected[0]
					if k.Name == fyne.KeyUp && current > 0 {
						list.Select(current - 1)
					} else if k.Name == fyne.KeyDown && current < list.Length()-1 {
						list.Select(current + 1)
					}
					return
				}
			}
		}

		if k.Name == fyne.KeyDelete || k.Name == fyne.KeyBackspace {
			// Don't delete items if user is typing in a text field
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

	// Capture GUI state on close without overwriting business settings
	w.SetOnClosed(func() {
		diskCfg, err := LoadConfig()
		if err != nil {
			diskCfg = state.Config
		}

		diskCfg.IsMaximized = w.IsMaximized()
		if !diskCfg.IsMaximized {
			diskCfg.WindowWidth = w.Canvas().Size().Width
			diskCfg.WindowHeight = w.Canvas().Size().Height
		}
		diskCfg.LogSplitOffset = state.LogSplit.Offset
		if state.MainSplit != nil {
			diskCfg.SplitOffsetMain = state.MainSplit.Offset
		}
		if state.LeftSplit != nil {
			diskCfg.SplitOffsetLeft = state.LeftSplit.Offset
		}
		if state.TopSplit != nil {
			diskCfg.SplitOffsetTop = state.TopSplit.Offset
		}
		SaveConfig(diskCfg)
	})

	w.ShowAndRun()
}



func makeHelpTab(state *AppState) fyne.CanvasObject {
	return widget.NewLabel("Nano Banana (Flash, pro and 2) can edit or generate multiple images and do batch.\nImagen is text to image only.")
}

func (s *AppState) Log(msg string) {
	timestamp := time.Now().Format("15:04:05")
	entry := "[" + timestamp + "] " + msg

	// UI Log
	s.LogLines = append(s.LogLines, entry)
	if len(s.LogLines) > 300 {
		s.LogLines = s.LogLines[len(s.LogLines)-300:]
	}
	if s.ModelLog != nil {
		s.ModelLog.Refresh()
		s.ModelLog.ScrollToBottom()
	}

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

func (s *AppState) LoadJobs() {
	f, err := os.Open("jobs.txt")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id := scanner.Text()
		if id == "" {
			continue
		}
		s.BatchJobs = append(s.BatchJobs, &BatchJob{
			JobID:       id,
			Status:      "Submitted",
			SubmittedAt: time.Now(),
			Progress:    "0%",
		})
	}
	s.Log(fmt.Sprintf("Loaded %d batch jobs from jobs.txt", len(s.BatchJobs)))
}

func (s *AppState) AddImages(paths []string) {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", s.NextImageID),
			FileName: filepath.Base(p),
			FullPath: p,
			SizeMB:   float64(info.Size()) / 1024 / 1024,
		})
		s.NextImageID++
	}
}
