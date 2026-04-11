package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (s *AppState) makeCreateTab() fyne.CanvasObject {
	var selectedImgID string
	var selectedTaskID int = -1

	fixedWidth := func(obj fyne.CanvasObject, width float32) fyne.CanvasObject {
		rect := canvas.NewRectangle(nil)
		rect.SetMinSize(fyne.NewSize(width, 0))
		return container.NewStack(rect, obj)
	}

	// Pre-declare components
	var imageList *widget.List
	var taskList *widget.List
	var preview *canvas.Image
	var promptEntry *TabbableEntry
	var negPromptEntry *TabbableEntry
	var tierSelect *widget.Select
	var ratioSelect *widget.Select
	var modeSelect *widget.RadioGroup
	var sourceIDsEntry *widget.Entry
	var costLabel *widget.Label

	// --- Component Initialization ---

	preview = canvas.NewImageFromResource(nil)
	preview.FillMode = canvas.ImageFillContain
	preview.SetMinSize(fyne.NewSize(300, 200))

	// Image List
	imageList = widget.NewList(
		func() int { return len(s.Images) },
		func() fyne.CanvasObject {
			check := widget.NewCheck("", nil)
			id := widget.NewLabel("")
			size := widget.NewLabel("")
			tasks := widget.NewLabel("")
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis

			content := container.NewHBox(
				fixedWidth(check, 30),
				fixedWidth(id, 30),
				fixedWidth(size, 60),
				fixedWidth(tasks, 50),
				fixedWidth(name, 400),
			)
			return newTappableListItem(content, 0, nil, nil)
		},
		func(id widget.ListItemID, cell fyne.CanvasObject) {
			item := cell.(*tappableListItem)
			item.id = id
			img := s.Images[id]

			hbox := item.content.(*fyne.Container)
			check := hbox.Objects[0].(*fyne.Container).Objects[1].(*widget.Check)
			check.Checked = img.Selected
			check.OnChanged = func(b bool) {
				img.Selected = b
			}

			hbox.Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(img.ID)
			hbox.Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("%.2f", img.SizeMB))
			hbox.Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("%d", img.TaskCount))
			hbox.Objects[4].(*fyne.Container).Objects[1].(*widget.Label).SetText(img.FileName)

			item.onTapped = func(id widget.ListItemID) {
				imageList.Select(id)
			}
			item.onRightClick = func(id widget.ListItemID, pos fyne.Position) {
				menu := fyne.NewMenu("",
					fyne.NewMenuItem("Change Image", func() {
						fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
							if err == nil && reader != nil {
								p := reader.URI().Path()
								img := s.Images[id]
								img.FullPath = p
								img.FileName = filepath.Base(p)
								info, _ := os.Stat(p)
								img.SizeMB = float64(info.Size()) / 1024 / 1024

								for _, t := range s.Tasks {
									if t.ImgIDs == img.ID {
										t.SourcePath = p
									}
								}
								imageList.Refresh()
								s.Log("Image changed to: " + p)
							}
						}, s.Window)
						fd.Show()
					}),
					fyne.NewMenuItem("Delete", func() {
						selectedImgID = s.Images[id].ID
						s.DeleteHandler()
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(item), pos)
			}
		},
	)

	imageList.OnSelected = func(id widget.ListItemID) {
		img := s.Images[id]
		selectedImgID = img.ID
		selectedTaskID = -1
		taskList.UnselectAll()

		if img.FullPath != "<GENERATE>" {
			preview.File = img.FullPath
			preview.Refresh()
		}
	}

	// Task List
	taskList = widget.NewList(
		func() int { return len(s.Tasks) },
		func() fyne.CanvasObject {
			id := widget.NewLabel("")
			agent := widget.NewLabel("")
			status := widget.NewLabel("")
			cost := widget.NewLabel("")
			prompt := widget.NewLabel("")
			prompt.Truncation = fyne.TextTruncateEllipsis

			content := container.NewHBox(
				fixedWidth(id, 60),
				fixedWidth(agent, 120),
				fixedWidth(status, 100),
				fixedWidth(cost, 80),
				fixedWidth(prompt, 500),
			)
			return newTappableListItem(content, 0, nil, nil)
		},
		func(id widget.ListItemID, cell fyne.CanvasObject) {
			item := cell.(*tappableListItem)
			item.id = id
			task := s.Tasks[id]

			hbox := item.content.(*fyne.Container)
			hbox.Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(task.ImgIDs)
			hbox.Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(task.Agent + " " + task.Size)
			stat := task.Status
			if task.Disabled {
				stat += " (Off)"
			}
			hbox.Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(stat)
			hbox.Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(fmt.Sprintf("$%.4f", task.Cost))
			hbox.Objects[4].(*fyne.Container).Objects[1].(*widget.Label).SetText(task.Prompt)

			item.onTapped = func(id widget.ListItemID) {
				taskList.Select(id)
			}
			item.onRightClick = func(id widget.ListItemID, pos fyne.Position) {
				toggleText := "Disable"
				if task.Disabled {
					toggleText = "Enable"
				}
				menu := fyne.NewMenu("",
					fyne.NewMenuItem(toggleText, func() {
						task.Disabled = !task.Disabled
						taskList.Refresh()
					}),
					fyne.NewMenuItem("Duplicate Task", func() {
						newTask := *task // Shallow copy is fine as fields are primitive or string
						newTask.ID = len(s.Tasks) + 1
						newTask.Status = "Pending"
						s.Tasks = append(s.Tasks, &newTask)
						taskList.Refresh()
						if s.OnTasksUpdated != nil {
							s.OnTasksUpdated()
						}
					}),
					fyne.NewMenuItem("Delete", func() {
						selectedTaskID = task.ID
						s.DeleteHandler()
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(item), pos)
			}
		},
	)

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

	taskList.OnSelected = func(id widget.ListItemID) {
		task := s.Tasks[id]
		selectedTaskID = task.ID
		selectedImgID = ""
		imageList.UnselectAll()

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

		promptEntry.OnChanged = oldP
		negPromptEntry.OnChanged = oldN
		tierSelect.OnChanged = oldT
		ratioSelect.OnChanged = oldR
		sourceIDsEntry.OnChanged = oldIDs
	}

	taskList.OnUnselected = func(id widget.ListItemID) {
		selectedTaskID = -1
		clearEditor()
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
				task.ImgIDs = sourceIDsEntry.Text
				// Update SourcePath based on new IDs
				var newPaths []string
				ids := strings.Split(task.ImgIDs, "+")
				for _, id := range ids {
					id = strings.TrimSpace(id)
					for _, img := range s.Images {
						if img.ID == id {
							newPaths = append(newPaths, img.FullPath)
							break
						}
					}
				}
				task.SourcePath = strings.Join(newPaths, "|")
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

			taskList.Refresh()
			if s.OnTasksUpdated != nil {
				s.OnTasksUpdated()
			}
		}
	}

	sourceIDsEntry.OnChanged = func(string) { updateTaskFromUI() }
	promptEntry.OnChanged = func(string) { updateTaskFromUI() }
	negPromptEntry.OnChanged = func(string) { updateTaskFromUI() }
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
		s.GlobalMode = m
		s.Log("Mode switched to: " + m)

		if runBtn != nil {
			if m == "Batch" {
				runBtn.SetText("RUN BATCH")
			} else {
				runBtn.SetText("RUN IMMEDIATE")
			}
		}

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
		taskList.Refresh()
		if s.OnTasksUpdated != nil {
			s.OnTasksUpdated()
		}
	})
	modeSelect.SetSelected("Immediate")
	modeSelect.Horizontal = true

	// --- Buttons ---

	newImgBtn := widget.NewButton("New Image", func() {
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", len(s.Images)+1),
			FileName: "GENERATE",
			FullPath: "<GENERATE>",
		})
		imageList.Refresh()
	})

	addTaskBtn := widget.NewButton("Add Task", func() {
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
			s.Log("Error: Select images (checkbox) first!")
			return
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

		s.Tasks = append(s.Tasks, &TaskInfo{
			ID:             len(s.Tasks) + 1,
			ImgIDs:         strings.Join(imgIDs, "+"),
			Agent:          agent,
			Size:           size,
			Ratio:          ratioSelect.Selected,
			Status:         "Pending",
			Cost:           s.CalculateCost(agent, size),
			Prompt:         promptEntry.Text,
			NegativePrompt: negPromptEntry.Text,
			SourcePath:     strings.Join(paths, "|"),
		})

		imageList.Refresh()
		taskList.Refresh()
		if s.OnTasksUpdated != nil {
			s.OnTasksUpdated()
		}
	})

	var runBtn *widget.Button
	runBtn = widget.NewButton("RUN IMMEDIATE", func() {
		runBtn.Disable()
		go func() {
			defer fyne.Do(runBtn.Enable)
			processedCount := 0
			batchTasks := make(map[string][]*TaskInfo)

			for i, task := range s.Tasks {
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
				fyne.Do(func() {
					task.Status = "Running"
					taskList.Refresh()
				})
				s.Log(fmt.Sprintf("[%d] Running %s...", i+1, task.Agent))
				err := s.RunTask(task)
				fyne.Do(func() {
					if err != nil {
						task.Status = "Failed"
						s.Log(fmt.Sprintf("Task %d failed: %v", task.ID, err))
					} else {
						task.Status = "Success"
					}
					taskList.Refresh()
					if s.OnTasksUpdated != nil {
						s.OnTasksUpdated()
					}
				})
			}

			for agent, tasks := range batchTasks {
				processedCount++
				s.Log(fmt.Sprintf("Submitting batch for %s (%d tasks)...", agent, len(tasks)))
				err := s.SubmitBatchJob(tasks)
				fyne.Do(func() {
					status := "Submitted"
					if err != nil {
						status = "Failed"
						s.Log("Batch Submission Error: " + err.Error())
					}
					for _, t := range tasks {
						t.Status = status
					}
					taskList.Refresh()
				})
			}

			if processedCount == 0 {
				s.Log("No tasks to run.")
			} else {
				s.Log(fmt.Sprintf("Finished processing %d jobs.", processedCount))
			}
		}()
	})

	// --- Layout Assembly ---

	totalCostLabel := widget.NewLabel("Total: $0.0000")
	s.OnTasksUpdated = func() {
		total := 0.0
		for _, t := range s.Tasks {
			total += t.Cost
		}
		totalCostLabel.SetText(fmt.Sprintf("Total: $%.4f", total))
	}

	saveBtn := widget.NewButton("Save Session", func() {
		err := s.SaveSession()
		if err != nil {
			s.Log("Save failed: " + err.Error())
		} else {
			s.Log("Session saved successfully.")
		}
	})

	loadBtn := widget.NewButton("Load Session", func() {
		err := s.LoadSession()
		if err != nil {
			s.Log("Load failed: " + err.Error())
		} else {
			imageList.Refresh()
			taskList.Refresh()
			if s.OnTasksUpdated != nil {
				s.OnTasksUpdated()
			}
			s.Log("Session loaded successfully.")
		}
	})

	testBtn := widget.NewButton("Test API Key", func() {
		go s.TestAPI()
	})

	btnLine1 := container.NewHBox(newImgBtn, addTaskBtn, totalCostLabel, modeSelect)
	btnLine2 := container.NewHBox(loadBtn, saveBtn, testBtn, runBtn)
	btnBox := container.NewVBox(btnLine1, btnLine2)

	promptEntry.SetMinRowsVisible(8)

	taskEditor := container.New(layout.NewFormLayout(),
		widget.NewLabel("Source IDs:"), sourceIDsEntry,
		widget.NewLabel("Positive Prompt:"), promptEntry,
		widget.NewLabel("Negative Prompt:"), negPromptEntry,
		widget.NewLabel("Tier:"), tierSelect,
		widget.NewLabel("Aspect Ratio:"), ratioSelect,
		widget.NewLabel("Cost:"), costLabel,
	)

	imageHeaders := container.NewHBox(
		fixedWidth(widget.NewLabel("Sel"), 30),
		fixedWidth(widget.NewLabel("#"), 30),
		fixedWidth(widget.NewLabel("MBs"), 60),
		fixedWidth(widget.NewLabel("Tasks"), 50),
		fixedWidth(widget.NewLabel("Image"), 400),
	)
	imageScroll := container.NewHScroll(container.NewBorder(imageHeaders, nil, nil, nil, imageList))
	imageScroll.SetMinSize(fyne.NewSize(400, 0))

	topHalf := container.NewHSplit(imageScroll, preview)
	topHalf.Offset = 0.5

	taskHeaders := container.NewHBox(
		fixedWidth(widget.NewLabel("Img"), 60),
		fixedWidth(widget.NewLabel("Tier"), 120),
		fixedWidth(widget.NewLabel("Status"), 100),
		fixedWidth(widget.NewLabel("Cost"), 80),
		fixedWidth(widget.NewLabel("Prompt"), 500),
	)
	taskScroll := container.NewHScroll(container.NewBorder(taskHeaders, nil, nil, nil, taskList))
	taskScroll.SetMinSize(fyne.NewSize(400, 140))

	leftBottom := container.NewBorder(nil, btnBox, nil, nil, taskScroll)
	right := container.NewVScroll(taskEditor)

	s.OnImagesUpdated = func() {
		imageList.Refresh()
		if selectedImgID == "" && len(s.Images) > 0 {
			imageList.Select(0)
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
					for _, img := range s.Images {
						if img.ID == id {
							img.TaskCount--
							break
						}
					}
				}

				selectedTaskID = -1
				taskList.Refresh()
				imageList.Refresh()
				if s.OnTasksUpdated != nil {
					s.OnTasksUpdated()
				}
				s.Log("Task deleted.")
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
				imageList.Refresh()
				taskList.Refresh()
				if s.OnTasksUpdated != nil {
					s.OnTasksUpdated()
				}
				s.Log("Image deleted.")
			}
		}
	}

	leftSplit := container.NewVSplit(topHalf, leftBottom)
	leftSplit.Offset = 0.5

	mainSplit := container.NewHSplit(
		leftSplit,
		right,
	)
	mainSplit.Offset = 0.75

	return mainSplit
}
