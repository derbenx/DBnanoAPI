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
	var selectedImgRows []int
	var selectedImgRow int = -1
	var selectedTaskRow int = -1

	updateColumnWidths := func(t *widget.Table, data [][]string) {
		for col := 0; col < len(data[0]); col++ {
			max := 0.0
			for row := 0; row < len(data); row++ {
				l := float64(len(data[row][col])) * 8.0 // Rough estimate
				if l > max {
					max = l
				}
			}
			if max < 30 {
				max = 30
			}
			if max > 400 {
				max = 400
			}
			t.SetColumnWidth(col, float32(max))
		}
	}

	// Pre-declare components to resolve cross-references
	var imageList *widget.Table
	var taskList *widget.Table
	var imageRTC *RightClickTable
	var taskRTC *RightClickTable
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

	imageRTC = NewRightClickTable(
		func() (int, int) { return len(s.Images) + 1, 5 },
		func(t *widget.Table) fyne.CanvasObject {
			l := NewRightClickLabel("", t)
			l.OnRightClick = func(id widget.TableCellID, pos fyne.Position) {
				if id.Row == 0 {
					return
				}
				menu := fyne.NewMenu("",
					fyne.NewMenuItem("Change Image", func() {
						selectedImgRow = id.Row
						fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
							if err == nil && reader != nil {
								p := reader.URI().Path()
								img := s.Images[id.Row-1]
								img.FullPath = p
								img.FileName = filepath.Base(p)
								info, _ := os.Stat(p)
								img.SizeMB = float64(info.Size()) / 1024 / 1024

								// Update tasks that use this image
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
					fyne.NewMenuItem("Select Multiple", func() {
						alreadySelected := false
						for _, r := range selectedImgRows {
							if r == id.Row {
								alreadySelected = true
								break
							}
						}
						if !alreadySelected {
							selectedImgRows = append(selectedImgRows, id.Row)
							imageList.Refresh()
						}
					}),
					fyne.NewMenuItem("Delete", func() {
						selectedImgRow = id.Row
						if len(selectedImgRows) <= 1 {
							selectedImgRows = []int{id.Row}
						}
						s.DeleteHandler()
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(l), pos)
			}
			return l
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*RightClickLabel)
			label.id = id
			if id.Row == 0 {
				headers := []string{"#", "MBs", "tasks", "Image", "Path"}
				label.SetText(headers[id.Col])
				label.Importance = widget.HighImportance
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			isSelected := false
			for _, r := range selectedImgRows {
				if r == id.Row {
					isSelected = true
					break
				}
			}
			if isSelected {
				label.Importance = widget.HighImportance
				label.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
			} else {
				label.Importance = widget.MediumImportance
				label.TextStyle = fyne.TextStyle{}
			}
			img := s.Images[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(img.ID)
			case 1:
				label.SetText(fmt.Sprintf("%.2f", img.SizeMB))
			case 2:
				label.SetText(fmt.Sprintf("%d", img.TaskCount))
			case 3:
				label.SetText(img.FileName)
			case 4:
				label.SetText(img.FullPath)
			}
		},
	)
	imageList = imageRTC.Table
	imageList.SetColumnWidth(0, 30)
	imageList.SetColumnWidth(1, 50)
	imageList.SetColumnWidth(2, 50)
	imageList.SetColumnWidth(3, 100)
	imageList.SetColumnWidth(4, 200)

	imageList.OnSelected = func(id widget.TableCellID) {
		// Standard selection
		selectedImgRow = id.Row

		// AHK-like behavior: single click selects one, clearing others.
		// Use right-click or a future modifier-key check for multi-select if needed.
		// For now, let's allow standard Fyne multi-select if we can, but the user complained about selection issues.
		// By default, just replace selection unless we implement a specific multi-select mode.
		selectedImgRows = []int{id.Row}

		selectedTaskRow = -1 // Clear other selection
		imageList.Refresh()
		taskList.Refresh()
		if id.Row > 0 {
			img := s.Images[id.Row-1]
			if img.FullPath != "<GENERATE>" {
				preview.File = img.FullPath
				preview.Refresh()
			}
		}
	}

	taskRTC = NewRightClickTable(
		func() (int, int) { return len(s.Tasks) + 1, 7 },
		func(t *widget.Table) fyne.CanvasObject {
			l := NewRightClickLabel("", t)
			l.OnRightClick = func(id widget.TableCellID, pos fyne.Position) {
				if id.Row == 0 {
					return
				}
				task := s.Tasks[id.Row-1]
				toggleText := "Disable"
				if task.Disabled {
					toggleText = "Enable"
				}
				menu := fyne.NewMenu("",
					fyne.NewMenuItem(toggleText, func() {
						task.Disabled = !task.Disabled
						taskList.Refresh()
					}),
					fyne.NewMenuItem("Delete", func() {
						selectedTaskRow = id.Row
						s.DeleteHandler()
					}),
				)
				widget.ShowPopUpMenuAtPosition(menu, fyne.CurrentApp().Driver().CanvasForObject(l), pos)
			}
			return l
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*RightClickLabel)
			label.id = id
			if id.Row == 0 {
				headers := []string{"Img", "Agent", "Res", "Ratio", "Status", "Cost", "Prompt"}
				label.SetText(headers[id.Col])
				label.Importance = widget.HighImportance
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			if id.Row == selectedTaskRow {
				label.Importance = widget.HighImportance
				label.TextStyle = fyne.TextStyle{Bold: true, Italic: true}
			} else {
				label.Importance = widget.MediumImportance
				label.TextStyle = fyne.TextStyle{}
			}
			task := s.Tasks[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(task.ImgIDs)
			case 1:
				label.SetText(task.Agent)
			case 2:
				label.SetText(task.Size)
			case 3:
				label.SetText(task.Ratio)
			case 4:
				status := task.Status
				if task.Disabled {
					status += " (Off)"
				}
				label.SetText(status)
			case 5:
				label.SetText(fmt.Sprintf("$%.4f", task.Cost))
			case 6:
				label.SetText(task.Prompt)
			}
		},
	)
	taskList = taskRTC.Table
	taskList.SetColumnWidth(0, 50)
	taskList.SetColumnWidth(1, 100)
	taskList.SetColumnWidth(2, 50)
	taskList.SetColumnWidth(3, 50)
	taskList.SetColumnWidth(4, 100)
	taskList.SetColumnWidth(5, 70)
	taskList.SetColumnWidth(6, 300)

	taskList.OnSelected = func(id widget.TableCellID) {
		selectedTaskRow = id.Row
		selectedImgRow = -1 // Clear other selection
		selectedImgRows = nil
		taskList.Refresh()
		imageList.Refresh()
		if id.Row > 0 && id.Row <= len(s.Tasks) {
			task := s.Tasks[id.Row-1]

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
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			task := s.Tasks[selectedTaskRow-1]
			task.ImgIDs = sourceIDsEntry.Text
			task.Prompt = promptEntry.Text
			task.NegativePrompt = negPromptEntry.Text

			// Parse Tier
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
		// Update all task costs
		for _, t := range s.Tasks {
			t.Cost = s.CalculateCost(t.Agent, t.Size)
		}
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			costLabel.SetText(fmt.Sprintf("$%.4f", s.Tasks[selectedTaskRow-1].Cost))
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

		rows := selectedImgRows
		if len(rows) == 0 && selectedImgRow > 0 {
			rows = []int{selectedImgRow}
		}

		if len(rows) > 0 {
			for _, r := range rows {
				if r <= 0 || r > len(s.Images) {
					continue
				}
				img := s.Images[r-1]
				imgIDs = append(imgIDs, img.ID)
				paths = append(paths, img.FullPath)
				img.TaskCount++
			}
		} else if len(s.Images) > 0 {
			img := s.Images[0]
			imgIDs = append(imgIDs, img.ID)
			paths = append(paths, img.FullPath)
			img.TaskCount++
		}

		if len(paths) == 0 {
			s.Log("Error: Add and select images first!")
			return
		}

		// Parse Tier
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

		// Update widths
		data := [][]string{{"Img", "Agent", "Res", "Ratio", "Status", "Cost", "Prompt"}}
		for _, t := range s.Tasks {
			data = append(data, []string{t.ImgIDs, t.Agent, t.Size, t.Ratio, t.Status, fmt.Sprintf("%.4f", t.Cost), t.Prompt})
		}
		updateColumnWidths(taskList, data)
	})

	runBtn := widget.NewButton("RUN TASKS", func() {
		go func() {
			processedCount := 0
			batchTasks := make(map[string][]*TaskInfo)

			for i, task := range s.Tasks {
				if task.Disabled {
					continue
				}
				if task.Status != "Pending" && task.Status != "Failed" && task.Status != "Success" {
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

	imgScroll := container.NewVScroll(imageRTC)
	topHalf := container.NewHSplit(imgScroll, preview)
	topHalf.Offset = 0.5

	taskTableScroll := container.NewVScroll(taskRTC)
	taskTableScroll.SetMinSize(fyne.NewSize(0, 140))

	leftBottom := container.NewBorder(nil, btnBox, nil, nil, taskTableScroll)
	right := container.NewVScroll(taskEditor)

	s.OnImagesUpdated = func() {
		// Calculate widths
		if len(s.Images) > 0 {
			data := [][]string{{"#", "MBs", "tasks", "Image", "Path"}}
			for _, img := range s.Images {
				data = append(data, []string{
					img.ID,
					fmt.Sprintf("%.2f", img.SizeMB),
					fmt.Sprintf("%d", img.TaskCount),
					img.FileName,
					img.FullPath,
				})
			}
			updateColumnWidths(imageList, data)
		}

		imageList.Refresh()
		// Auto-select first image if nothing selected
		if selectedImgRow == -1 && len(s.Images) > 0 {
			imageList.Select(widget.TableCellID{Row: 1, Col: 0})
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
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			deletedTask := s.Tasks[selectedTaskRow-1]
			s.Tasks = append(s.Tasks[:selectedTaskRow-1], s.Tasks[selectedTaskRow:]...)

			// Update task counts for all images involved in this task
			ids := strings.Split(deletedTask.ImgIDs, "+")
			for _, id := range ids {
				for _, img := range s.Images {
					if img.ID == id {
						img.TaskCount--
						break
					}
				}
			}

			selectedTaskRow = -1
			taskList.Refresh()
			imageList.Refresh()
			if s.OnTasksUpdated != nil {
				s.OnTasksUpdated()
			}
			s.Log("Task deleted.")
			return
		}
		if len(selectedImgRows) > 0 {
			// Sort in descending order to avoid index issues during deletion
			for i := 0; i < len(selectedImgRows); i++ {
				for j := i + 1; j < len(selectedImgRows); j++ {
					if selectedImgRows[i] < selectedImgRows[j] {
						selectedImgRows[i], selectedImgRows[j] = selectedImgRows[j], selectedImgRows[i]
					}
				}
			}

			for _, row := range selectedImgRows {
				if row <= 0 || row > len(s.Images) {
					continue
				}
				deletedImgID := s.Images[row-1].ID
				s.Images = append(s.Images[:row-1], s.Images[row:]...)

				newTasks := []*TaskInfo{}
				for _, t := range s.Tasks {
					if !isIDInMergedID(deletedImgID, t.ImgIDs) {
						newTasks = append(newTasks, t)
					}
				}
				s.Tasks = newTasks
			}
			count := len(selectedImgRows)
			selectedImgRow = -1
			selectedImgRows = nil
			imageList.Refresh()
			taskList.Refresh()
			if s.OnTasksUpdated != nil {
				s.OnTasksUpdated()
			}
			s.Log(fmt.Sprintf("%d Images deleted.", count))
			selectedImgRows = nil
			return
		}
	}

	leftSplit := container.NewVSplit(topHalf, leftBottom)
	leftSplit.Offset = 0.5

	mainSplit := container.NewHSplit(
		leftSplit,
		right,
	)
	mainSplit.Offset = 0.6

	return mainSplit
}
