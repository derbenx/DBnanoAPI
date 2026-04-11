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
	var agentSelect *widget.Select
	var sizeSelect *widget.Select
	var ratioSelect *widget.Select
	var modeSelect *widget.RadioGroup
	var sourceIDsEntry *widget.Entry

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
					fyne.NewMenuItem("Delete", func() {
						selectedImgRow = id.Row
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

		// Manage multi-select
		alreadySelected := false
		for _, r := range selectedImgRows {
			if r == id.Row {
				alreadySelected = true
				break
			}
		}
		if !alreadySelected {
			selectedImgRows = append(selectedImgRows, id.Row)
		}

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
		func() (int, int) { return len(s.Tasks) + 1, 6 },
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
				headers := []string{"Img", "Agent", "Res", "Ratio", "Status", "Prompt"}
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
	taskList.SetColumnWidth(5, 300)

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
			oldA := agentSelect.OnChanged
			oldS := sizeSelect.OnChanged
			oldR := ratioSelect.OnChanged
			oldIDs := sourceIDsEntry.OnChanged

			promptEntry.OnChanged = nil
			negPromptEntry.OnChanged = nil
			agentSelect.OnChanged = nil
			sizeSelect.OnChanged = nil
			ratioSelect.OnChanged = nil
			sourceIDsEntry.OnChanged = nil

			sourceIDsEntry.SetText(task.ImgIDs)
			promptEntry.SetText(task.Prompt)
			negPromptEntry.SetText(task.NegativePrompt)
			agentSelect.SetSelected(task.Agent)
			sizeSelect.SetSelected(task.Size)
			ratioSelect.SetSelected(task.Ratio)

			promptEntry.OnChanged = oldP
			negPromptEntry.OnChanged = oldN
			agentSelect.OnChanged = oldA
			sizeSelect.OnChanged = oldS
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

	agentOptions := []string{"Nano Flash", "Nano Pro", "Nano 2", "Imagen", "Imagen Ultra"}
	agentSelect = widget.NewSelect(agentOptions, nil)

	sizeSelect = widget.NewSelect([]string{"1K", "2K", "4K"}, nil)
	sizeSelect.SetSelected("1K")

	ratioOptions := []string{"Default", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9"}
	ratioOptionsExt := []string{"Default", "1:8", "1:4", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9", "4:1", "8:1"}
	ratioSelect = widget.NewSelect(ratioOptions, nil)
	ratioSelect.SetSelected("1:1")

	updateTaskFromUI := func() {
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			task := s.Tasks[selectedTaskRow-1]
			task.ImgIDs = sourceIDsEntry.Text
			task.Prompt = promptEntry.Text
			task.NegativePrompt = negPromptEntry.Text
			task.Agent = agentSelect.Selected
			task.Size = sizeSelect.Selected
			task.Ratio = ratioSelect.Selected
			taskList.Refresh()
		}
	}

	sourceIDsEntry.OnChanged = func(string) { updateTaskFromUI() }
	promptEntry.OnChanged = func(string) { updateTaskFromUI() }
	negPromptEntry.OnChanged = func(string) { updateTaskFromUI() }
	agentSelect.OnChanged = func(str string) {
		updateTaskFromUI()
		if str == "Nano 2" {
			ratioSelect.Options = ratioOptionsExt
		} else {
			ratioSelect.Options = ratioOptions
		}
		ratioSelect.Refresh()

		if str == "Imagen" || str == "Imagen Ultra" {
			sizeSelect.Options = []string{"2K"}
			sizeSelect.SetSelected("2K")
		} else if str == "Nano Flash" {
			sizeSelect.Options = []string{"1K"}
			sizeSelect.SetSelected("1K")
		} else {
			sizeSelect.Options = []string{"1K", "2K", "4K"}
		}
		sizeSelect.Refresh()
	}
	sizeSelect.OnChanged = func(string) { updateTaskFromUI() }
	ratioSelect.OnChanged = func(string) { updateTaskFromUI() }
	agentSelect.SetSelected("Nano Flash")

	modeSelect = widget.NewRadioGroup([]string{"Immediate", "Batch"}, func(m string) {
		s.GlobalMode = m
		s.Log("Mode switched to: " + m)
	})
	modeSelect.SetSelected("Immediate")
	modeSelect.Horizontal = true

	// --- Buttons ---

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
			s.Log("Session loaded successfully.")
		}
	})

	newImgBtn := widget.NewButton("New Image", func() {
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", len(s.Images)+1),
			FileName: "GENERATE",
			FullPath: "<GENERATE>",
		})
		imageList.Refresh()
	})

	addTaskBtn := widget.NewButton("Add Task", func() {
		imgID := ""
		path := ""
		if selectedImgRow > 0 && selectedImgRow <= len(s.Images) {
			img := s.Images[selectedImgRow-1]
			imgID = img.ID
			path = img.FullPath
			img.TaskCount++
			imageList.Refresh()
		} else if len(s.Images) > 0 {
			img := s.Images[0]
			imgID = img.ID
			path = img.FullPath
			img.TaskCount++
			imageList.Refresh()
		}

		if path == "" {
			s.Log("Error: Add an image first!")
			return
		}

		s.Tasks = append(s.Tasks, &TaskInfo{
			ID:             len(s.Tasks) + 1,
			ImgIDs:         imgID,
			Agent:          agentSelect.Selected,
			Size:           sizeSelect.Selected,
			Ratio:          ratioSelect.Selected,
			Status:         "Pending",
			Prompt:         promptEntry.Text,
			NegativePrompt: negPromptEntry.Text,
			SourcePath:     path,
		})
		taskList.Refresh()

		// Update widths
		data := [][]string{{"Img", "Agent", "Res", "Ratio", "Status", "Prompt"}}
		for _, t := range s.Tasks {
			data = append(data, []string{t.ImgIDs, t.Agent, t.Size, t.Ratio, t.Status, t.Prompt})
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

	btnBox := container.NewHBox(saveBtn, loadBtn, newImgBtn, addTaskBtn, layout.NewSpacer(), widget.NewLabel("Mode:"), modeSelect, runBtn)

	promptEntry.SetMinRowsVisible(8)

	taskEditor := container.New(layout.NewFormLayout(),
		widget.NewLabel("Source\nIDs:"), sourceIDsEntry,
		widget.NewLabel("Positive\nPrompt:"), promptEntry,
		widget.NewLabel("Negative\nPrompt:"), negPromptEntry,
		widget.NewLabel("Agent:"), agentSelect,
		widget.NewLabel("Size:"), sizeSelect,
		widget.NewLabel("Aspect\nRatio:"), ratioSelect,
	)

	imgScroll := container.NewVScroll(imageRTC)
	topHalf := container.NewHSplit(imgScroll, preview)
	topHalf.Offset = 0.5

	taskTableScroll := container.NewVScroll(taskRTC)
	taskTableScroll.SetMinSize(fyne.NewSize(0, 200))

	middle := container.NewBorder(btnBox, nil, nil, nil, taskTableScroll)
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

	s.DeleteHandler = func() {
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			deletedTask := s.Tasks[selectedTaskRow-1]
			s.Tasks = append(s.Tasks[:selectedTaskRow-1], s.Tasks[selectedTaskRow:]...)
			for _, img := range s.Images {
				if img.ID == deletedTask.ImgIDs {
					img.TaskCount--
					break
				}
			}
			selectedTaskRow = -1
			taskList.Refresh()
			imageList.Refresh()
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
					if t.ImgIDs != deletedImgID {
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
			s.Log(fmt.Sprintf("%d Images deleted.", count))
			selectedImgRows = nil
			return
		}
	}

	mainSplit := container.NewHSplit(
		container.NewVSplit(topHalf, middle),
		right,
	)
	mainSplit.Offset = 0.7

	return mainSplit
}
