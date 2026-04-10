package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (s *AppState) makeCreateTab() fyne.CanvasObject {
	var selectedImgRow int = -1
	var selectedTaskRow int = -1

	// Pre-declare tables so they can be referenced in each other's callbacks if needed
	var imageList *widget.Table
	var taskList *widget.Table

	// Left side: Image List
	imageList = widget.NewTable(
		func() (int, int) { return len(s.Images) + 1, 5 },
		func() fyne.CanvasObject { return widget.NewLabel("Header") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			if id.Row == 0 {
				headers := []string{"#", "MBs", "tasks", "Image", "Path"}
				label.SetText(headers[id.Col])
				return
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

	imageList.SetColumnWidth(0, 30)
	imageList.SetColumnWidth(1, 50)
	imageList.SetColumnWidth(2, 50)
	imageList.SetColumnWidth(3, 100)
	imageList.SetColumnWidth(4, 200)

	// Preview
	preview := canvas.NewImageFromResource(nil)
	preview.FillMode = canvas.ImageFillContain
	preview.SetMinSize(fyne.NewSize(300, 200))

	imageList.OnSelected = func(id widget.TableCellID) {
		selectedImgRow = id.Row
		if id.Row > 0 {
			img := s.Images[id.Row-1]
			if img.FullPath != "<GENERATE>" {
				preview.File = img.FullPath
				preview.Refresh()
			}
		}
	}

	// Right side: Task Editor
	promptEntry := NewTabbableEntry()
	promptEntry.SetText(s.Config.DefaultPrompt)
	promptEntry.Wrapping = fyne.TextWrapWord

	negPromptEntry := NewTabbableEntry()
	negPromptEntry.SetText(s.Config.DefaultNegPrompt)
	negPromptEntry.Wrapping = fyne.TextWrapWord

	agentOptions := []string{"Nano Flash", "Nano Pro", "Nano 2", "Imagen", "Imagen Ultra"}
	agentSelect := widget.NewSelect(agentOptions, nil)

	sizeSelect := widget.NewSelect([]string{"1K", "2K", "4K"}, nil)
	sizeSelect.SetSelected("1K")

	ratioOptions := []string{"Default", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9"}
	ratioOptionsExt := []string{"Default", "1:8", "1:4", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9", "4:1", "8:1"}
	ratioSelect := widget.NewSelect(ratioOptions, nil)
	ratioSelect.SetSelected("1:1")

	updateTaskFromUI := func() {
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			task := s.Tasks[selectedTaskRow-1]
			task.Prompt = promptEntry.Text
			task.NegativePrompt = negPromptEntry.Text
			task.Agent = agentSelect.Selected
			task.Size = sizeSelect.Selected
			task.Ratio = ratioSelect.Selected
			taskList.Refresh()
		}
	}

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

	// Task List (Bottom)
	taskList = widget.NewTable(
		func() (int, int) { return len(s.Tasks) + 1, 6 },
		func() fyne.CanvasObject { return widget.NewLabel("Header") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			if id.Row == 0 {
				headers := []string{"Img", "Agent", "Res", "Ratio", "Status", "Prompt"}
				label.SetText(headers[id.Col])
				return
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
				label.SetText(task.Status)
			case 5:
				label.SetText(task.Prompt)
			}
		},
	)

	taskList.OnSelected = func(id widget.TableCellID) {
		selectedTaskRow = id.Row
		if id.Row > 0 && id.Row <= len(s.Tasks) {
			task := s.Tasks[id.Row-1]
			// Temporarily disable OnChanged to avoid self-update loop during population
			oldP := promptEntry.OnChanged
			oldN := negPromptEntry.OnChanged
			oldA := agentSelect.OnChanged
			oldS := sizeSelect.OnChanged
			oldR := ratioSelect.OnChanged

			promptEntry.OnChanged = nil
			negPromptEntry.OnChanged = nil
			agentSelect.OnChanged = nil
			sizeSelect.OnChanged = nil
			ratioSelect.OnChanged = nil

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
		}
	}

	// Buttons
	newImgBtn := widget.NewButton("New Image", func() {
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", len(s.Images)+1),
			FileName: "GENERATE",
			FullPath: "<GENERATE>",
		})
		imageList.Refresh()
	})


	modeSelect := widget.NewRadioGroup([]string{"Immediate", "Batch"}, func(string) {})
	modeSelect.SetSelected("Immediate")
	modeSelect.Horizontal = true

	addTaskBtn := widget.NewButton("Add Task", func() {
		selectedImages := ""
		sourcePaths := ""

		if selectedImgRow > 0 && selectedImgRow <= len(s.Images) {
			img := s.Images[selectedImgRow-1]
			selectedImages = img.ID
			sourcePaths = img.FullPath
			img.TaskCount++
			imageList.Refresh()
		} else if len(s.Images) > 0 {
			// Fallback to first image if nothing selected
			img := s.Images[0]
			selectedImages = img.ID
			sourcePaths = img.FullPath
			img.TaskCount++
			imageList.Refresh()
		}

		if sourcePaths == "" {
			s.Log("Error: No image selected to add task.")
			return
		}

		s.Tasks = append(s.Tasks, &TaskInfo{
			ID:             len(s.Tasks) + 1,
			ImgIDs:         selectedImages,
			Agent:          agentSelect.Selected,
			Size:           sizeSelect.Selected,
			Ratio:          ratioSelect.Selected,
			Status:         "Pending",
			Prompt:         promptEntry.Text,
			NegativePrompt: negPromptEntry.Text,
			Mode:           modeSelect.Selected,
			SourcePath:     sourcePaths,
		})
		taskList.Refresh()
	})

	runBtn := widget.NewButton("RUN TASKS", func() {
		go func() {
			processedCount := 0
			batchTasks := make(map[string][]*TaskInfo)

			for i, task := range s.Tasks {
				if task.Status != "Pending" && task.Status != "Failed" {
					continue
				}

				if task.Mode == "Batch" {
					if strings.Contains(task.Agent, "Imagen") {
						s.Log(fmt.Sprintf("[%d] Warning: Imagen doesn't support batch. Using Immediate mode.", i+1))
					} else {
						batchTasks[task.Agent] = append(batchTasks[task.Agent], task)
						processedCount++
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

			// Submit batch groups
			for agent, tasks := range batchTasks {
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
				s.Log("No pending or failed tasks to run.")
			} else {
				s.Log(fmt.Sprintf("Finished processing %d tasks.", processedCount))
			}
		}()
	})

	btnBox := container.NewHBox(newImgBtn, addTaskBtn, layout.NewSpacer(), widget.NewLabel("Mode:"), modeSelect, runBtn)

	taskEditor := widget.NewForm(
		widget.NewFormItem("Positive Prompt", promptEntry),
		widget.NewFormItem("Negative Prompt", negPromptEntry),
		widget.NewFormItem("Agent", agentSelect),
		widget.NewFormItem("Size", sizeSelect),
		widget.NewFormItem("Aspect Ratio", ratioSelect),
	)

	// Layout
	imgScroll := container.NewVScroll(imageList)
	topHalf := container.New(layout.NewGridLayout(2), imgScroll, preview)

	taskTableScroll := container.NewVScroll(taskList)
	taskTableScroll.SetMinSize(fyne.NewSize(0, 150)) // Set height for task list

	middle := container.NewVBox(btnBox, taskTableScroll)
	right := container.NewVScroll(taskEditor)

	s.OnImagesUpdated = func() {
		imageList.Refresh()
	}

	s.DeleteHandler = func() {
		if selectedTaskRow > 0 && selectedTaskRow <= len(s.Tasks) {
			deletedTask := s.Tasks[selectedTaskRow-1]
			s.Tasks = append(s.Tasks[:selectedTaskRow-1], s.Tasks[selectedTaskRow:]...)

			// Decrement TaskCount for the associated image
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
		if selectedImgRow > 0 && selectedImgRow <= len(s.Images) {
			deletedImgID := s.Images[selectedImgRow-1].ID
			s.Images = append(s.Images[:selectedImgRow-1], s.Images[selectedImgRow:]...)

			// Remove all tasks associated with this image
			newTasks := []*TaskInfo{}
			for _, t := range s.Tasks {
				if t.ImgIDs != deletedImgID {
					newTasks = append(newTasks, t)
				}
			}
			s.Tasks = newTasks

			selectedImgRow = -1
			imageList.Refresh()
			taskList.Refresh()
			s.Log("Image deleted.")
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
