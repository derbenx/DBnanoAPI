package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func parseRatio(str string) float32 {
	if str == "Default" || str == "" {
		return 0
	}
	parts := strings.Split(str, ":")
	if len(parts) != 2 {
		return 0
	}
	w, _ := strconv.ParseFloat(parts[0], 32)
	h, _ := strconv.ParseFloat(parts[1], 32)
	if h == 0 {
		return 0
	}
	return float32(w / h)
}

func getImageDimensions(path string) (int, int, error) {
	if path == "<GENERATE>" {
		return 1024, 1024, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func GetClosestRatio(w, h int) string {
	target := float64(w) / float64(h)
	ratios := []string{"1:8", "1:4", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9", "4:1", "8:1"}
	bestMatch := "1:1"
	minDiff := 999.0

	for _, str := range ratios {
		parts := strings.Split(str, ":")
		rw, _ := strconv.ParseFloat(parts[0], 64)
		rh, _ := strconv.ParseFloat(parts[1], 64)
		ratioVal := rw / rh
		diff := math.Abs(target - ratioVal)

		if diff < minDiff {
			minDiff = diff
			bestMatch = str
		}
	}
	return bestMatch
}

type ratioOverlayLayout struct {
	targetRatio float32
}

func (r *ratioOverlayLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 5 {
		return
	}

	preview := objects[0]
	preview.Resize(size)
	preview.Move(fyne.NewPos(0, 0))

	if r.targetRatio <= 0 {
		for i := 1; i < 5; i++ {
			objects[i].Hide()
		}
		return
	}

	pw, ph := size.Width, size.Height
	if pw == 0 || ph == 0 {
		return
	}

	var bw, bh float32
	if pw/ph > r.targetRatio {
		bh = ph
		bw = bh * r.targetRatio
	} else {
		bw = pw
		bh = bw / r.targetRatio
	}

	bx := (pw - bw) / 2
	by := (ph - bh) / 2
	thick := float32(4)

	// Top
	objects[1].Show()
	objects[1].Resize(fyne.NewSize(bw, thick))
	objects[1].Move(fyne.NewPos(bx, by))

	// Bottom
	objects[2].Show()
	objects[2].Resize(fyne.NewSize(bw, thick))
	objects[2].Move(fyne.NewPos(bx, by+bh-thick))

	// Left
	objects[3].Show()
	objects[3].Resize(fyne.NewSize(thick, bh))
	objects[3].Move(fyne.NewPos(bx, by))

	// Right
	objects[4].Show()
	objects[4].Resize(fyne.NewSize(thick, bh))
	objects[4].Move(fyne.NewPos(bx+bw-thick, by))
}

func (r *ratioOverlayLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(300, 200)
	}
	return objects[0].MinSize()
}


func (s *AppState) makeCreateTab() fyne.CanvasObject {
	var selectedImgID string
	var selectedTaskID int = -1
	// In place of the closure, use a real layout later if needed, but for now
	// let's simplify the stack to reduce object count.

	// Pre-declare components
	var imageTable *widget.Table
	var taskTable *widget.Table
	var preview *canvas.Image
	var promptEntry *TabbableEntry
	var negPromptEntry *TabbableEntry
	var tierSelect *widget.Select
	var ratioSelect *widget.Select
	var modeSelect *widget.RadioGroup
	var sourceIDsEntry *widget.Entry
	var costLabel *widget.Label
	var runBtn *widget.Button
	var overlayLayout *ratioOverlayLayout
	var previewContainer *fyne.Container
	var debounceTimer *time.Timer
	var debounceLock sync.Mutex

	updateRatioOverlay := func(ratioStr string) {
		if overlayLayout == nil || previewContainer == nil {
			return
		}
		overlayLayout.targetRatio = parseRatio(ratioStr)
		previewContainer.Refresh()
	}

	updateRunBtnLabel := func() {
		if runBtn == nil || modeSelect == nil {
			return
		}
		if modeSelect.Selected == "Batch" {
			runBtn.SetText("RUN BATCH")
		} else {
			runBtn.SetText("RUN IMMEDIATE")
		}
	}

	// --- Component Initialization ---

	preview = canvas.NewImageFromResource(nil)
	preview.FillMode = canvas.ImageFillContain
	preview.SetMinSize(fyne.NewSize(50, 50))

	overlayT := canvas.NewRectangle(color.RGBA{R: 255, A: 255})
	overlayB := canvas.NewRectangle(color.RGBA{R: 255, A: 255})
	overlayL := canvas.NewRectangle(color.RGBA{R: 255, A: 255})
	overlayR := canvas.NewRectangle(color.RGBA{R: 255, A: 255})
	overlayT.Hide()
	overlayB.Hide()
	overlayL.Hide()
	overlayR.Hide()

	overlayLayout = &ratioOverlayLayout{targetRatio: 0}
	previewContainer = container.New(overlayLayout, preview, overlayT, overlayB, overlayL, overlayR)

	// Image Table
	imgCols := []float32{40, 40, 60, 50, 400}
	imageTable = NewRightClickTable(
		func() (int, int) {
			s.Mu.RLock()
			defer s.Mu.RUnlock()
			return len(s.Images) + 1, 5
		},
		func(t *widget.Table) fyne.CanvasObject {
			l := NewRightClickLabel("", t)
			l.OnRightClick = func(id widget.TableCellID, pos fyne.Position) {
				if id.Row == 0 {
					return
				}
				row := id.Row - 1
				s.Mu.RLock()
				img := s.Images[row]
				s.Mu.RUnlock()

				menu := fyne.NewMenu("",
					fyne.NewMenuItem("Change Image", func() {
						fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
							if err == nil && reader != nil {
								p := reader.URI().Path()
								s.Mu.Lock()
								img.FullPath = p
								img.FileName = filepath.Base(p)
								info, _ := os.Stat(p)
								img.SizeMB = float64(info.Size()) / 1024 / 1024

								for _, t := range s.Tasks {
									if t.ImgIDs == img.ID {
										t.SourcePath = p
									}
								}
								s.Mu.Unlock()
								imageTable.Refresh()
								s.Log("Image changed to: " + p)
							}
						}, s.Window)

						cwd, _ := os.Getwd()
						if lister, err := storage.ListerForURI(storage.NewFileURI(cwd)); err == nil {
							fd.SetLocation(lister)
						}
						fd.Resize(fyne.NewSize(800, 550))
						fd.Show()
					}),
					fyne.NewMenuItem("Delete", func() {
						selectedTaskID = -1
						selectedImgID = img.ID
						s.DeleteHandler()
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(l), pos)
			}
			return l
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			l := cell.(*RightClickLabel)
			l.id = id
			l.TextStyle = fyne.TextStyle{}
			l.Alignment = fyne.TextAlignLeading

			if id.Row == 0 {
				headers := []string{"Sel", "#", "MBs", "Tasks", "Image"}
				l.SetText(headers[id.Col])
				l.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			s.Mu.RLock()
			if id.Row-1 >= len(s.Images) {
				s.Mu.RUnlock()
				return
			}
			img := s.Images[id.Row-1]
			s.Mu.RUnlock()

			switch id.Col {
			case 0:
				if img.Selected {
					l.SetText("[X]")
				} else {
					l.SetText("[ ]")
				}
			case 1:
				l.SetText(img.ID)
			case 2:
				l.SetText(fmt.Sprintf("%.2f", img.SizeMB))
			case 3:
				l.SetText(fmt.Sprintf("%d", img.TaskCount))
			case 4:
				l.SetText(img.FileName)
			}
		},
	)
	for i, w := range imgCols {
		imageTable.SetColumnWidth(i, w)
	}
	imageTable.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			imageTable.Unselect(id)
			return
		}
		row := id.Row - 1
		s.Mu.RLock()
		if row >= len(s.Images) {
			s.Mu.RUnlock()
			return
		}
		img := s.Images[row]
		s.Mu.RUnlock()

		if id.Col == 0 {
			img.Selected = !img.Selected
			imageTable.Refresh()
			imageTable.Unselect(id)
			return
		}

		selectedImgID = img.ID
		selectedTaskID = -1
		taskTable.UnselectAll()
		s.Window.Canvas().Focus(imageTable)

		if img.FullPath != "<GENERATE>" {
			preview.File = img.FullPath
		} else {
			preview.File = ""
		}
		preview.Refresh()
		updateRatioOverlay("Default")
	}

	// Task Table
	taskCols := []float32{60, 120, 60, 100, 80, 500}
	taskTable = NewRightClickTable(
		func() (int, int) {
			s.Mu.RLock()
			defer s.Mu.RUnlock()
			return len(s.Tasks) + 1, 6
		},
		func(t *widget.Table) fyne.CanvasObject {
			l := NewRightClickLabel("", t)
			l.OnRightClick = func(id widget.TableCellID, pos fyne.Position) {
				if id.Row == 0 {
					return
				}
				row := id.Row - 1
				s.Mu.RLock()
				task := s.Tasks[row]
				s.Mu.RUnlock()

				toggleText := "Disable"
				if task.Disabled {
					toggleText = "Enable"
				}
				menu := fyne.NewMenu("",
					fyne.NewMenuItem(toggleText, func() {
						task.Disabled = !task.Disabled
						taskTable.Refresh()
					}),
					fyne.NewMenuItem("Duplicate Task", func() {
						s.Mu.Lock()
						newTask := *task
						newTask.ID = s.NextTaskID
						s.NextTaskID++
						newTask.Status = "Pending"
						s.Tasks = append(s.Tasks, &newTask)
						s.Mu.Unlock()
						taskTable.Refresh()
						if s.OnTasksUpdated != nil {
							s.OnTasksUpdated()
						}
					}),
					fyne.NewMenuItem("Delete", func() {
						selectedImgID = ""
						selectedTaskID = task.ID
						s.DeleteHandler()
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(l), pos)
			}
			return l
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			l := cell.(*RightClickLabel)
			l.id = id
			l.TextStyle = fyne.TextStyle{}

			if id.Row == 0 {
				headers := []string{"Img", "Tier", "Ratio", "Status", "Cost", "Prompt"}
				l.SetText(headers[id.Col])
				l.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			s.Mu.RLock()
			if id.Row-1 >= len(s.Tasks) {
				s.Mu.RUnlock()
				return
			}
			task := s.Tasks[id.Row-1]
			s.Mu.RUnlock()

			switch id.Col {
			case 0:
				l.SetText(task.ImgIDs)
			case 1:
				l.SetText(task.Agent + " " + task.Size)
			case 2:
				l.SetText(task.Ratio)
			case 3:
				stat := task.Status
				if task.Disabled {
					stat += " (Off)"
				}
				l.SetText(stat)
			case 4:
				l.SetText(fmt.Sprintf("$%.4f", task.Cost))
			case 5:
				l.SetText(task.Prompt)
			}
		},
	)
	for i, w := range taskCols {
		taskTable.SetColumnWidth(i, w)
	}

	clearEditor := func() {
		// Block OnChanged listeners
		oldP := promptEntry.OnChanged
		oldN := negPromptEntry.OnChanged
		oldT := tierSelect.OnChanged
		oldR := ratioSelect.OnChanged
		oldIDs := sourceIDsEntry.OnChanged

		promptEntry.OnChanged = nil
		negPromptEntry.OnChanged = nil
		tierSelect.OnChanged = nil
		ratioSelect.OnChanged = nil
		sourceIDsEntry.OnChanged = nil

		sourceIDsEntry.SetText("")
		promptEntry.SetText("")
		negPromptEntry.SetText("")
		tierSelect.ClearSelected()
		ratioSelect.ClearSelected()
		costLabel.SetText("$0.00")

		promptEntry.OnChanged = oldP
		negPromptEntry.OnChanged = oldN
		tierSelect.OnChanged = oldT
		ratioSelect.OnChanged = oldR
		sourceIDsEntry.OnChanged = oldIDs
	}

	taskTable.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			taskTable.Unselect(id)
			return
		}
		row := id.Row - 1
		s.Mu.RLock()
		if row >= len(s.Tasks) {
			s.Mu.RUnlock()
			return
		}
		task := s.Tasks[row]
		s.Mu.RUnlock()
		selectedTaskID = task.ID
		selectedImgID = ""
		imageTable.UnselectAll()
		s.Window.Canvas().Focus(taskTable)

		// Block OnChanged listeners
		oldP := promptEntry.OnChanged
		oldN := negPromptEntry.OnChanged
		oldT := tierSelect.OnChanged
		oldR := ratioSelect.OnChanged
		oldIDs := sourceIDsEntry.OnChanged

		promptEntry.OnChanged = nil
		negPromptEntry.OnChanged = nil
		tierSelect.OnChanged = nil
		ratioSelect.OnChanged = nil
		sourceIDsEntry.OnChanged = nil

		sourceIDsEntry.SetText(task.ImgIDs)
		promptEntry.SetText(task.Prompt)
		negPromptEntry.SetText(task.NegativePrompt)
		tierSelect.SetSelected(task.Agent + " " + task.Size)
		ratioSelect.SetSelected(task.Ratio)
		costLabel.SetText(fmt.Sprintf("$%.4f", task.Cost))
		updateRatioOverlay(task.Ratio)

		promptEntry.OnChanged = oldP
		negPromptEntry.OnChanged = oldN
		tierSelect.OnChanged = oldT
		ratioSelect.OnChanged = oldR
		sourceIDsEntry.OnChanged = oldIDs
	}

	// Task Editor Fields
	sourceIDsEntry = widget.NewEntry()

	promptEntry = NewTabbableEntry()
	promptEntry.SetText(s.Config.DefaultPrompt)
	promptEntry.Wrapping = fyne.TextWrapWord

	negPromptEntry = NewTabbableEntry()
	negPromptEntry.SetText(s.Config.DefaultNegPrompt)
	negPromptEntry.Wrapping = fyne.TextWrapWord

	tierOptions := []string{"Nano Flash 1K", "Nano Pro 1K", "Nano Pro 2K", "Nano Pro 4K", "Nano 2 1K", "Nano 2 2K", "Nano 2 4K", "Imagen 2K", "Imagen Ultra 2K"}
	tierSelect = widget.NewSelect(tierOptions, nil)

	ratioOptions := []string{"Default", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9"}
	ratioOptionsExt := []string{"Default", "1:8", "1:4", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9", "4:1", "8:1"}
	ratioSelect = widget.NewSelect(ratioOptions, nil)
	ratioSelect.SetSelected("1:1")

	costLabel = widget.NewLabel("$0.00")

	updateTaskFromUI := func() {
		s.Mu.Lock()
		var task *TaskInfo
		if selectedTaskID != -1 {
			for _, t := range s.Tasks {
				if t.ID == selectedTaskID {
					task = t
					break
				}
			}
		}

		if task != nil {
			if task.ImgIDs != sourceIDsEntry.Text {
				// 1. Decrement current image task counts
				oldIDs := strings.Split(task.ImgIDs, "+")
				for _, id := range oldIDs {
					id = strings.TrimSpace(id)
					for _, img := range s.Images {
						if img.ID == id {
							if img.TaskCount > 0 {
								img.TaskCount--
							}
							break
						}
					}
				}

				// 2. Validate and filter new IDs, then increment their task counts
				var validIDs []string
				var newPaths []string
				ids := strings.Split(sourceIDsEntry.Text, "+")
				for _, id := range ids {
					id = strings.TrimSpace(id)
					if id == "" {
						continue
					}
					for _, img := range s.Images {
						if img.ID == id {
							validIDs = append(validIDs, id)
							newPaths = append(newPaths, img.FullPath)
							img.TaskCount++
							break
						}
					}
				}
				task.ImgIDs = strings.Join(validIDs, "+")
				task.SourcePath = strings.Join(newPaths, "|")

				// Update entry if we filtered anything
				if task.ImgIDs != sourceIDsEntry.Text {
					sourceIDsEntry.SetText(task.ImgIDs)
				}
				s.Mu.Unlock()
				imageTable.Refresh()
				s.Mu.Lock()
			}
			task.Prompt = promptEntry.Text
			task.NegativePrompt = negPromptEntry.Text

			parts := strings.Split(tierSelect.Selected, " ")
			if len(parts) >= 3 {
				task.Agent = strings.Join(parts[:len(parts)-1], " ")
				task.Size = parts[len(parts)-1]
			} else if len(parts) == 2 {
				task.Agent = parts[0]
				task.Size = parts[1]
			}

			task.Ratio = ratioSelect.Selected
			task.Cost = s.CalculateCost(task.Agent, task.Size)
			costLabel.SetText(fmt.Sprintf("$%.4f", task.Cost))
			updateRatioOverlay(task.Ratio)

			s.Mu.Unlock()
			taskTable.Refresh()
			if s.OnTasksUpdated != nil {
				s.OnTasksUpdated()
			}
		} else {
			s.Mu.Unlock()
		}
	}

	debounceUpdate := func(delay time.Duration) {
		debounceLock.Lock()
		defer debounceLock.Unlock()

		if debounceTimer != nil {
			debounceTimer.Stop()
		}

		debounceTimer = time.AfterFunc(delay, func() {
			updateTaskFromUI()
		})
	}

	sourceIDsEntry.OnChanged = func(string) { debounceUpdate(2000 * time.Millisecond) }
	promptEntry.OnChanged = func(string) { debounceUpdate(500 * time.Millisecond) }
	negPromptEntry.OnChanged = func(string) { debounceUpdate(500 * time.Millisecond) }
	tierSelect.OnChanged = func(str string) {
		if strings.Contains(str, "Nano 2") {
			ratioSelect.Options = ratioOptionsExt
		} else {
			ratioSelect.Options = ratioOptions
		}
		ratioSelect.Refresh()
		updateTaskFromUI()
	}
	ratioSelect.OnChanged = func(string) { updateTaskFromUI() }
	tierSelect.SetSelected("Nano Flash 1K")

	modeSelect = widget.NewRadioGroup([]string{"Immediate", "Batch"}, func(m string) {
		s.Mu.Lock()
		s.GlobalMode = m
		s.Mu.Unlock()
		s.Log("Mode switched to: " + m)

		updateRunBtnLabel()

		s.Mu.Lock()
		for _, t := range s.Tasks {
			t.Cost = s.CalculateCost(t.Agent, t.Size)
		}

		var task *TaskInfo
		if selectedTaskID != -1 {
			for _, t := range s.Tasks {
				if t.ID == selectedTaskID {
					task = t
					break
				}
			}
		}
		if task != nil {
			costLabel.SetText(fmt.Sprintf("$%.4f", task.Cost))
		}
		s.Mu.Unlock()
		taskTable.Refresh()
		if s.OnTasksUpdated != nil {
			s.OnTasksUpdated()
		}
	})
	modeSelect.SetSelected("Immediate")
	modeSelect.Horizontal = true

	// --- Buttons ---

	newImgBtn := widget.NewButton("New Image", func() {
		s.Mu.Lock()
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", len(s.Images)+1),
			FileName: "GENERATE",
			FullPath: "<GENERATE>",
		})
		s.Mu.Unlock()
		imageTable.Refresh()
	})

	addTaskBtn := widget.NewButton("Add Task", func() {
		s.Mu.Lock()
		var imgIDs []string
		var paths []string

		for _, img := range s.Images {
			if img.Selected {
				imgIDs = append(imgIDs, img.ID)
				paths = append(paths, img.FullPath)
				img.TaskCount++
			}
		}

		if len(paths) == 0 && selectedImgID != "" {
			for _, img := range s.Images {
				if img.ID == selectedImgID {
					imgIDs = append(imgIDs, img.ID)
					paths = append(paths, img.FullPath)
					img.TaskCount++
					break
				}
			}
		}

		if len(paths) == 0 {
			s.Mu.Unlock()
			s.Log("Error: Select images (checkbox) first!")
			return
		}

		// Detect ratio from first selected image
		detectedRatio := "1:1"
		if len(paths) > 0 {
			w, h, err := getImageDimensions(paths[0])
			if err == nil {
				detectedRatio = GetClosestRatio(w, h)
			}
		}

		var agent, size string
		parts := strings.Split(tierSelect.Selected, " ")
		if len(parts) >= 3 {
			agent = strings.Join(parts[:len(parts)-1], " ")
			size = parts[len(parts)-1]
		} else if len(parts) == 2 {
			agent = parts[0]
			size = parts[1]
		}

		newTask := &TaskInfo{
			ID:             s.NextTaskID,
			ImgIDs:         strings.Join(imgIDs, "+"),
			Agent:          agent,
			Size:           size,
			Ratio:          detectedRatio,
			Status:         "Pending",
			Cost:           s.CalculateCost(agent, size),
			Prompt:         s.Config.DefaultPrompt,
			NegativePrompt: s.Config.DefaultNegPrompt,
			SourcePath:     strings.Join(paths, "|"),
		}
		s.Tasks = append(s.Tasks, newTask)
		s.NextTaskID++
		s.Mu.Unlock()

		imageTable.Refresh()
		taskTable.Refresh()
		if s.OnTasksUpdated != nil {
			s.OnTasksUpdated()
		}
		// Select the new task
		taskTable.Select(widget.TableCellID{Row: len(s.Tasks), Col: 1})
	})

	runBtn = widget.NewButton("RUN IMMEDIATE", func() {
		runBtn.Disable()
		go func() {
			defer runBtn.Enable()
			processedCount := 0
			batchTasks := make(map[string][]*TaskInfo)

			s.Mu.RLock()
			tasksCopy := make([]*TaskInfo, len(s.Tasks))
			copy(tasksCopy, s.Tasks)
			s.Mu.RUnlock()

			for i, task := range tasksCopy {
				if task.Disabled {
					continue
				}
				// Allow re-running Pending, Failed, Success, and Submitted tasks
				// Basically anything not currently "Running"
				if task.Status == "Running" {
					continue
				}

				if s.GlobalMode == "Batch" {
					if strings.Contains(task.Agent, "Imagen") {
						s.Log(fmt.Sprintf("[%d] Warning: Imagen doesn't support batch. Using Immediate mode.", i+1))
					} else {
						batchTasks[task.Agent] = append(batchTasks[task.Agent], task)
						continue
					}
				}

				processedCount++
				s.Mu.Lock()
				task.Status = "Running"
				s.Mu.Unlock()
				taskTable.Refresh()

				s.Log(fmt.Sprintf("[%d] Running %s...", i+1, task.Agent))
				err := s.RunTask(task)
				s.Mu.Lock()
				if err != nil {
					task.Status = "Failed"
					s.Mu.Unlock()
					s.Log(fmt.Sprintf("Task %d failed: %v", task.ID, err))
				} else {
					task.Status = "Success"
					s.Mu.Unlock()
				}
				taskTable.Refresh()
				if s.OnTasksUpdated != nil {
					s.OnTasksUpdated()
				}
			}

			for agent, tasks := range batchTasks {
				processedCount++
				s.Log(fmt.Sprintf("Submitting batch for %s (%d tasks)...", agent, len(tasks)))
				err := s.SubmitBatchJob(tasks)
				status := "Submitted"
				if err != nil {
					status = "Failed"
					s.Log("Batch Submission Error: " + err.Error())
				}
				s.Mu.Lock()
				for _, t := range tasks {
					t.Status = status
				}
				s.Mu.Unlock()
				taskTable.Refresh()
			}

			if processedCount == 0 {
				s.Log("No tasks to run.")
			} else {
				s.Log(fmt.Sprintf("Finished processing %d jobs.", processedCount))
			}
		}()
	})
	updateRunBtnLabel()

	// --- Layout Assembly ---

	totalCostLabel := widget.NewLabel("Total: $0.0000")
	s.OnTasksUpdated = func() {
		total := 0.0
		s.Mu.RLock()
		for _, t := range s.Tasks {
			total += t.Cost
		}
		s.Mu.RUnlock()
		totalCostLabel.SetText(fmt.Sprintf("Total: $%.4f", total))
	}

	saveBtn := widget.NewButton("Save Session", func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err == nil && writer != nil {
				defer writer.Close()
				err = s.SaveSession(writer)
				if err != nil {
					s.Log("Save failed: " + err.Error())
				} else {
					s.Log("Session saved to: " + writer.URI().String())
				}
			}
		}, s.Window)

		// Set default directory and filename
		cwd, _ := os.Getwd()
		if lister, err := storage.ListerForURI(storage.NewFileURI(cwd)); err == nil {
			fd.SetLocation(lister)
		}
		fd.SetFileName("session.json")
		fd.Resize(fyne.NewSize(800, 550))
		fd.Show()
	})

	loadBtn := widget.NewButton("Load Session", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				defer reader.Close()
				err = s.LoadSession(reader)
				if err != nil {
					s.Log("Load failed: " + err.Error())
				} else {
					// Update IDs to prevent collisions
					maxImg := 0
					for _, img := range s.Images {
						id := 0
						fmt.Sscanf(img.ID, "%d", &id)
						if id > maxImg {
							maxImg = id
						}
					}
					s.NextImageID = maxImg + 1

					maxTask := 0
					for _, t := range s.Tasks {
						if t.ID > maxTask {
							maxTask = t.ID
						}
					}
					s.NextTaskID = maxTask + 1

					imageTable.Refresh()
					taskTable.Refresh()
					if s.OnTasksUpdated != nil {
						s.OnTasksUpdated()
					}
					s.Log("Session loaded from: " + reader.URI().String())
				}
			}
		}, s.Window)

		// Set default directory
		cwd, _ := os.Getwd()
		if lister, err := storage.ListerForURI(storage.NewFileURI(cwd)); err == nil {
			fd.SetLocation(lister)
		}
		fd.Resize(fyne.NewSize(800, 550))
		fd.Show()
	})

	btnLine1 := container.NewHBox(newImgBtn, addTaskBtn, totalCostLabel, modeSelect)
	btnLine2 := container.NewHBox(loadBtn, saveBtn, runBtn)
	btnBox := container.NewVBox(btnLine1, btnLine2)

	promptEntry.SetMinRowsVisible(8)

	taskEditor := container.NewVBox(
		widget.NewLabel("Source IDs:"),
		sourceIDsEntry,
		widget.NewLabel("Positive Prompt:"),
		promptEntry,
		widget.NewLabel("Negative Prompt:"),
		negPromptEntry,
		container.New(layout.NewFormLayout(),
			widget.NewLabel("Tier:"), tierSelect,
			widget.NewLabel("Aspect Ratio:"), ratioSelect,
			widget.NewLabel("Cost:"), costLabel,
		),
	)

	imageScroll := container.NewScroll(imageTable)
	imageScroll.SetMinSize(fyne.NewSize(50, 50))

	topHalf := container.NewHSplit(imageScroll, previewContainer)
	topHalf.Offset = s.Config.SplitOffsetTop
	s.TopSplit = topHalf

	taskScroll := container.NewScroll(taskTable)
	taskScroll.SetMinSize(fyne.NewSize(50, 50))

	leftBottom := container.NewBorder(nil, btnBox, nil, nil, taskScroll)
	right := container.NewVScroll(taskEditor)

	s.OnImagesUpdated = func() {
		imageTable.Refresh()
		s.Mu.RLock()
		defer s.Mu.RUnlock()
		if len(s.Images) > 0 {
			// If no selection, or if we want to snap to the latest addition
			// we select the LAST added image (most natural for drops)
			lastIdx := len(s.Images) - 1
			img := s.Images[lastIdx]
			if img.FullPath != "<GENERATE>" {
				preview.File = img.FullPath
			} else {
				preview.File = ""
			}
			preview.Refresh()
			imageTable.Select(widget.TableCellID{Row: lastIdx + 1, Col: 1})
		} else {
			preview.File = ""
			preview.Resource = nil
			preview.Refresh()
		}
	}

	isIDInMergedID := func(id, mID string) bool {
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

	s.DeleteHandler = func() {
		s.Mu.Lock()
		if selectedTaskID != -1 {
			var idx = -1
			for i, t := range s.Tasks {
				if t.ID == selectedTaskID {
					idx = i
					break
				}
			}
			if idx != -1 {
				deletedTask := s.Tasks[idx]
				s.Tasks = append(s.Tasks[:idx], s.Tasks[idx+1:]...)

				ids := strings.Split(deletedTask.ImgIDs, "+")
				for _, id := range ids {
					id = strings.TrimSpace(id)
					for _, img := range s.Images {
						if img.ID == id {
							if img.TaskCount > 0 {
								img.TaskCount--
							}
							break
						}
					}
				}

				selectedTaskID = -1
				clearEditor()
				s.Mu.Unlock()
				taskTable.Refresh()
				imageTable.Refresh()
				if s.OnTasksUpdated != nil {
					s.OnTasksUpdated()
				}
				s.Log("Task deleted.")

				s.Mu.Lock()
				if len(s.Tasks) == 0 {
					s.NextTaskID = 1
				}
				s.Mu.Unlock()
			} else {
				s.Mu.Unlock()
			}
			return
		}

		if selectedImgID != "" {
			var idx = -1
			for i, img := range s.Images {
				if img.ID == selectedImgID {
					idx = i
					break
				}
			}
			if idx != -1 {
				s.Images = append(s.Images[:idx], s.Images[idx+1:]...)

				newTasks := []*TaskInfo{}
				for _, t := range s.Tasks {
					if !isIDInMergedID(selectedImgID, t.ImgIDs) {
						newTasks = append(newTasks, t)
					}
				}
				s.Tasks = newTasks

				selectedImgID = ""
				s.Mu.Unlock()
				imageTable.Refresh()
				taskTable.Refresh()
				if s.OnTasksUpdated != nil {
					s.OnTasksUpdated()
				}
				s.Log("Image deleted.")

				s.Mu.RLock()
				// Auto-select next best image
				if len(s.Images) > 0 {
					newIdx := idx
					if newIdx >= len(s.Images) {
						newIdx = len(s.Images) - 1
					}
					// Explicitly update preview for the new selection
					img := s.Images[newIdx]
					s.Mu.RUnlock()
					if img.FullPath != "<GENERATE>" {
						preview.File = img.FullPath
					} else {
						preview.File = ""
					}
					preview.Refresh()
					imageTable.Select(widget.TableCellID{Row: newIdx + 1, Col: 1})
				} else {
					s.Mu.RUnlock()
					// Clear preview if no images left
					preview.File = ""
					preview.Resource = nil
					preview.Refresh()
					// Reset counters if empty
					s.Mu.Lock()
					s.NextImageID = 1
					s.Mu.Unlock()
				}
			} else {
				s.Mu.Unlock()
			}
		} else {
			s.Mu.Unlock()
		}
	}

	leftSplit := container.NewVSplit(topHalf, leftBottom)
	leftSplit.Offset = s.Config.SplitOffsetLeft
	s.LeftSplit = leftSplit

	mainSplit := container.NewHSplit(
		leftSplit,
		right,
	)
	mainSplit.Offset = s.Config.SplitOffsetMain
	s.MainSplit = mainSplit

	return mainSplit
}
