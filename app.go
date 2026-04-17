package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	goruntime "runtime"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx    context.Context
	Mu     sync.RWMutex
	Config *Config

	Images    []*ImageInfo
	Tasks     []*TaskInfo
	BatchJobs []*BatchJob

	HTTPClient *http.Client

	NextImageID int
	NextTaskID  int

	BatchMonitorIndex int
	GlobalMode        string
}

func NewApp() *App {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		cfg = DefaultConfig()
	}

	return &App{
		Config:      cfg,
		NextImageID: 1,
		NextTaskID:  1,
		GlobalMode:  "Immediate",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.LoadJobs()

	// Reset task states on startup
	a.Mu.Lock()
	for _, t := range a.Tasks {
		t.RunningCount = 0
		if t.Status == "Running" {
			t.Status = "Failed"
		}
	}
	a.Mu.Unlock()

	// Background Monitoring
	go func() {
		const interval = 2 * time.Minute
		nextCheck := time.Now().Add(interval)
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			a.Mu.RLock()
			jobsLen := len(a.BatchJobs)
			a.Mu.RUnlock()
			if jobsLen == 0 {
				continue
			}

			remaining := time.Until(nextCheck)
			if remaining <= 0 {
				a.Mu.RLock()
				var activeJobs []*BatchJob
				for _, job := range a.BatchJobs {
					s := job.Status
					if s != "SUCCEEDED" && s != "FAILED" && s != "CANCELLED" && s != "EXPIRED" && s != "Success" && s != "Failed" {
						activeJobs = append(activeJobs, job)
					}
				}
				a.Mu.RUnlock()

				if len(activeJobs) > 0 {
					if a.BatchMonitorIndex >= len(activeJobs) {
						a.BatchMonitorIndex = 0
					}
					target := activeJobs[a.BatchMonitorIndex]
					a.BatchMonitorIndex++

					a.Log(fmt.Sprintf("Checking status for job: %s", target.JobID))
					err := a.CheckBatchStatus(target)
					if err != nil {
						a.Log("Status check error: " + err.Error())
					}
					runtime.EventsEmit(a.ctx, "batch_updated")
				}

				nextCheck = time.Now().Add(interval)
				remaining = interval
			}
			runtime.EventsEmit(a.ctx, "batch_timer", int(remaining.Seconds()))
		}
	}()
}

func (a *App) GetConfig() *Config {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.Config
}

func (a *App) SaveConfig(cfg *Config) error {
	a.Mu.Lock()
	a.Config = cfg
	a.Mu.Unlock()
	return SaveConfig(cfg)
}

func (a *App) GetImages() []*ImageInfo {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.Images
}

func (a *App) GetTasks() []*TaskInfo {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.Tasks
}

func (a *App) Log(msg string) {
	runtime.EventsEmit(a.ctx, "log", msg)

	if a.Config.Debug {
		a.LogToFile(msg)
	}
}

func (a *App) LogToFile(msg string) {
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

func (a *App) LoadJobs() {
	f, err := os.Open("jobs.txt")
	if err != nil {
		return
	}
	defer f.Close()

	a.Mu.Lock()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		id := scanner.Text()
		if id == "" {
			continue
		}
		a.BatchJobs = append(a.BatchJobs, &BatchJob{
			JobID:       id,
			Status:      "Submitted",
			SubmittedAt: time.Now(),
			Progress:    "0%",
		})
	}
	count := len(a.BatchJobs)
	a.Mu.Unlock()
	a.Log(fmt.Sprintf("Loaded %d batch jobs from jobs.txt", count))
}

func (a *App) AddImages(paths []string) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		a.Images = append(a.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", a.NextImageID),
			FileName: filepath.Base(p),
			FullPath: p,
			SizeMB:   float64(info.Size()) / 1024 / 1024,
		})
		a.NextImageID++
	}
	runtime.EventsEmit(a.ctx, "images_updated")
}

func (a *App) GetImageBase64(path string) (string, error) {
	if path == "<GENERATE>" || path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (a *App) SelectAndAddMultipleImages() {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Images",
		Filters: []runtime.FileFilter{
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png"},
		},
	})
	if err != nil {
		a.Log("Error selecting images: " + err.Error())
		return
	}
	if len(files) > 0 {
		a.AddImages(files)
	}
}

func (a *App) CreateNewImage() {
	a.Mu.Lock()
	a.Images = append(a.Images, &ImageInfo{
		ID:       fmt.Sprintf("%d", a.NextImageID),
		FileName: "GENERATE",
		FullPath: "<GENERATE>",
	})
	a.NextImageID++
	a.Mu.Unlock()
	runtime.EventsEmit(a.ctx, "images_updated")
}

func (a *App) SaveSessionUI() {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Session",
		DefaultFilename: "session.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return
	}

	f, err := os.Create(path)
	if err != nil {
		a.Log("Error creating file: " + err.Error())
		return
	}
	defer f.Close()

	if err := a.SaveSession(f); err != nil {
		a.Log("Error saving session: " + err.Error())
	} else {
		a.Log("Session saved to: " + path)
	}
}

func (a *App) LoadSessionUI() {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Load Session",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		a.Log("Error opening file: " + err.Error())
		return
	}
	defer f.Close()

	if err := a.LoadSession(f); err != nil {
		a.Log("Error loading session: " + err.Error())
	} else {
		maxImg := 0
		for _, img := range a.Images {
			var id int
			fmt.Sscanf(img.ID, "%d", &id)
			if id > maxImg {
				maxImg = id
			}
		}
		a.NextImageID = maxImg + 1

		maxTask := 0
		for _, t := range a.Tasks {
			if t.ID > maxTask {
				maxTask = t.ID
			}
		}
		a.NextTaskID = maxTask + 1

		runtime.EventsEmit(a.ctx, "images_updated")
		runtime.EventsEmit(a.ctx, "tasks_updated")
		a.Log("Session loaded from: " + path)
	}
}

func (a *App) AddTask(imgIDs string, agent string, size string, ratio string, prompt string, negPrompt string, paths string) {
	a.Mu.Lock()
	ids := strings.Split(imgIDs, "+")
	for _, id := range ids {
		id = strings.TrimSpace(id)
		for _, img := range a.Images {
			if img.ID == id {
				img.TaskCount++
				break
			}
		}
	}
	newTask := &TaskInfo{
		ID:             a.NextTaskID,
		ImgIDs:         imgIDs,
		Agent:          agent,
		Size:           size,
		Ratio:          ratio,
		Status:         "Pending",
		Cost:           a.CalculateCost(agent, size),
		Prompt:         prompt,
		NegativePrompt: negPrompt,
		SourcePath:     paths,
	}
	a.Tasks = append(a.Tasks, newTask)
	a.NextTaskID++
	a.Mu.Unlock()
	runtime.EventsEmit(a.ctx, "tasks_updated")
}

func (a *App) UpdateTask(task *TaskInfo) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for i, t := range a.Tasks {
		if t.ID == task.ID {
			a.Tasks[i] = task
			break
		}
	}
	runtime.EventsEmit(a.ctx, "tasks_updated")
}

func (a *App) DeleteImage(id string) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	idx := -1
	for i, img := range a.Images {
		if img.ID == id {
			idx = i
			break
		}
	}
	if idx != -1 {
		a.Images = append(a.Images[:idx], a.Images[idx+1:]...)
		newTasks := []*TaskInfo{}
		for _, t := range a.Tasks {
			if !a.isIDInMergedID(id, t.ImgIDs) {
				newTasks = append(newTasks, t)
			}
		}
		a.Tasks = newTasks
		if len(a.Images) == 0 {
			a.NextImageID = 1
		}
		runtime.EventsEmit(a.ctx, "images_updated")
		runtime.EventsEmit(a.ctx, "tasks_updated")
		a.Log("Image " + id + " deleted.")
	}
}

func (a *App) isIDInMergedID(id, mID string) bool {
	if id == mID {
		return true
	}
	for _, p := range strings.Split(mID, "+") {
		if p == id {
			return true
		}
	}
	return false
}

func (a *App) DeleteTask(id int) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	idx := -1
	for i, t := range a.Tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx != -1 {
		deletedTask := a.Tasks[idx]
		a.Tasks = append(a.Tasks[:idx], a.Tasks[idx+1:]...)
		ids := strings.Split(deletedTask.ImgIDs, "+")
		for _, imgID := range ids {
			imgID = strings.TrimSpace(imgID)
			for _, img := range a.Images {
				if img.ID == imgID {
					if img.TaskCount > 0 {
						img.TaskCount--
					}
					break
				}
			}
		}
		if len(a.Tasks) == 0 {
			a.NextTaskID = 1
		}
		runtime.EventsEmit(a.ctx, "images_updated")
		runtime.EventsEmit(a.ctx, "tasks_updated")
		a.Log("Task deleted.")
	}
}

func (a *App) DuplicateTask(id int) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	var original *TaskInfo
	for _, t := range a.Tasks {
		if t.ID == id {
			original = t
			break
		}
	}
	if original != nil {
		newTask := *original
		newTask.ID = a.NextTaskID
		a.NextTaskID++
		newTask.Status = "Pending"
		a.Tasks = append(a.Tasks, &newTask)
		runtime.EventsEmit(a.ctx, "tasks_updated")
		a.Log("Task duplicated.")
	}
}

func (a *App) ToggleTaskDisabled(id int) {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, t := range a.Tasks {
		if t.ID == id {
			t.Disabled = !t.Disabled
			break
		}
	}
	runtime.EventsEmit(a.ctx, "tasks_updated")
}

func (a *App) ChangeImageUI(id string) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Change Image",
		Filters: []runtime.FileFilter{{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png"}},
	})
	if err != nil || path == "" {
		return
	}
	a.Mu.Lock()
	defer a.Mu.Unlock()
	for _, img := range a.Images {
		if img.ID == id {
			img.FullPath = path
			img.FileName = filepath.Base(path)
			info, _ := os.Stat(path)
			img.SizeMB = float64(info.Size()) / 1024 / 1024
			for _, t := range a.Tasks {
				if t.ImgIDs == img.ID {
					t.SourcePath = path
				}
			}
			break
		}
	}
	runtime.EventsEmit(a.ctx, "images_updated")
	runtime.EventsEmit(a.ctx, "tasks_updated")
}

func (a *App) RunTasks() {
	a.Mu.Lock()
	a.GlobalMode = "Immediate"
	var tasksToRun []*TaskInfo
	for _, t := range a.Tasks {
		if !t.Disabled && (t.Status == "Pending" || t.Status == "Failed" || t.Status == "Success" || t.Status == "Running") {
			tasksToRun = append(tasksToRun, t)
		}
	}
	a.Mu.Unlock()

	if len(tasksToRun) == 0 {
		return
	}

	runtime.EventsEmit(a.ctx, "run_started")
	go func() {
		defer runtime.EventsEmit(a.ctx, "run_finished")

		if len(tasksToRun) == 1 {
			// If only one task, we just fire off ONE execution.
			// The frontend button logic ensures we don't exceed 2 concurrent runs.
			a.executeTask(tasksToRun[0])
		} else {
			// Multiple tasks: Run them all sequentially.
			// We only start if no other task is already running (enforced by frontend button).
			for _, task := range tasksToRun {
				if task.Disabled || task.RunningCount > 0 {
					continue
				}
				a.executeTask(task)
			}
		}
	}()
}

func (a *App) executeTask(task *TaskInfo) {
	a.Mu.Lock()
	task.Status = "Running"
	task.RunningCount++
	mode := a.GlobalMode
	a.Mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			a.Log(fmt.Sprintf("Recovered from panic in task %d: %v", task.ID, r))
		}
		a.Mu.Lock()
		task.RunningCount--
		if task.RunningCount < 0 {
			task.RunningCount = 0
		}
		a.Mu.Unlock()
		runtime.EventsEmit(a.ctx, "tasks_updated")
	}()

	runtime.EventsEmit(a.ctx, "tasks_updated")
	a.Log(fmt.Sprintf("Running %s...", task.Agent))
	err := a.RunTask(task, mode)

	a.Mu.Lock()
	if err != nil {
		task.Status = "Failed"
		a.Log(fmt.Sprintf("Task %d failed: %v", task.ID, err))
	} else {
		task.Status = "Success"
	}
	a.Mu.Unlock()
}

func (a *App) RunBatch() {
	a.Mu.Lock()
	a.GlobalMode = "Batch"
	var tasks []*TaskInfo
	for _, t := range a.Tasks {
		if !t.Disabled && t.Status != "Running" {
			tasks = append(tasks, t)
		}
	}
	a.Mu.Unlock()
	if len(tasks) == 0 {
		a.Log("No tasks to batch.")
		return
	}
	runtime.EventsEmit(a.ctx, "run_started")
	go func() {
		defer runtime.EventsEmit(a.ctx, "run_finished")
		err := a.SubmitBatchJob(tasks)
		if err != nil {
			a.Log("Batch failed: " + err.Error())
			a.Mu.Lock()
			for _, t := range tasks { t.Status = "Failed" }
			a.Mu.Unlock()
		} else {
			a.Mu.Lock()
			for _, t := range tasks { t.Status = "Submitted" }
			a.Mu.Unlock()
		}
		runtime.EventsEmit(a.ctx, "tasks_updated")
	}()
}

func (a *App) GetBatchJobs() []*BatchJob {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return a.BatchJobs
}

func (a *App) ClearFinishedJobs() {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	var active []*BatchJob
	for _, j := range a.BatchJobs {
		s := j.Status
		if s != "SUCCEEDED" && s != "FAILED" && s != "CANCELLED" && s != "EXPIRED" && s != "Success" && s != "Failed" {
			active = append(active, j)
		}
	}
	a.BatchJobs = active
	a.CleanupJobsFile()
	runtime.EventsEmit(a.ctx, "batch_updated")
}

func (a *App) OpenImageFolder() {
	out := a.Config.OutputDir
	if out == "" {
		out = "img"
	}
	abs, _ := filepath.Abs(out)

	// Ensure the directory exists
	os.MkdirAll(abs, 0755)

	a.Log("Opening folder: " + abs)

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", abs)
	case "darwin":
		cmd = exec.Command("open", abs)
	default: // linux, etc
		cmd = exec.Command("xdg-open", abs)
	}

	if err := cmd.Start(); err != nil {
		a.Log("Error opening folder: " + err.Error())
		// Fallback to Wails BrowserOpenURL
		runtime.BrowserOpenURL(a.ctx, abs)
	}
}

func (a *App) GetLastGeneratedImage(taskID int) string {
	lastFile := a.getLastGeneratedImagePath(taskID)
	if lastFile == "" {
		return ""
	}
	b64, _ := a.GetImageBase64(lastFile)
	return b64
}

func (a *App) HasGeneratedImage(taskID int) bool {
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	for _, t := range a.Tasks {
		if t.ID == taskID {
			return t.LastSavedPath != ""
		}
	}
	return false
}

func (a *App) getLastGeneratedImagePath(taskID int) string {
	out := a.Config.OutputDir
	if out == "" {
		out = "img"
	}
	files, err := os.ReadDir(out)
	if err != nil {
		return ""
	}
	var lastFile string
	var lastTime time.Time
	prefix := fmt.Sprintf("GoTask_%d_", taskID)
	batchPrefix := fmt.Sprintf("Batch_task_%d_", taskID)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) || strings.HasPrefix(f.Name(), batchPrefix) {
			info, _ := f.Info()
			if info.ModTime().After(lastTime) {
				lastTime = info.ModTime()
				lastFile = filepath.Join(out, f.Name())
			}
		}
	}
	return lastFile
}

func (a *App) GetCost(agent, size, mode string) float64 {
	a.Mu.Lock()
	oldMode := a.GlobalMode
	a.GlobalMode = mode
	a.Mu.Unlock()

	cost := a.CalculateCost(agent, size)

	a.Mu.Lock()
	a.GlobalMode = oldMode
	a.Mu.Unlock()

	return cost
}

func (a *App) TestConnection() {
	go func() {
		runtime.EventsEmit(a.ctx, "test_api_started")
		if err := a.TestAPI(); err != nil {
			a.Log("API Test Failed: " + err.Error())
			runtime.EventsEmit(a.ctx, "test_api_finished", false, err.Error())
		} else {
			runtime.EventsEmit(a.ctx, "test_api_finished", true, "Success")
		}
	}()
}
