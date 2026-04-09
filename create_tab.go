package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (s *AppState) makeCreateTab() fyne.CanvasObject {
	var selectedImgRow int = -1

	// Left side: Image List
	imageList := widget.NewTable(
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

	// Task List (Bottom)
	taskList := widget.NewTable(
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

	// Buttons
	newImgBtn := widget.NewButton("New Image", func() {
		s.Images = append(s.Images, &ImageInfo{
			ID:       fmt.Sprintf("%d", len(s.Images)+1),
			FileName: "GENERATE",
			FullPath: "<GENERATE>",
		})
		imageList.Refresh()
	})

	// Right side: Task Editor
	promptEntry := widget.NewMultiLineEntry()
	promptEntry.SetText(s.Config.DefaultPrompt)

	negPromptEntry := widget.NewMultiLineEntry()
	negPromptEntry.SetText(s.Config.DefaultNegPrompt)

	agentOptions := []string{
		"Nano Flash 1K",
		"Nano Pro 1K", "Nano Pro 2K", "Nano Pro 4K",
		"Nano 2 1K", "Nano 2 2K", "Nano 2 4K",
		"Imagen 2K", "Imagen Ultra 2K",
	}
	agentSelect := widget.NewSelect(agentOptions, func(string) {})
	agentSelect.SetSelected("Nano Flash 1K")

	sizeSelect := widget.NewSelect([]string{"1K", "2K", "4K"}, func(string) {})
	sizeSelect.SetSelected("1K")

	ratios := []string{"Default", "1:8", "1:4", "9:16", "2:3", "3:4", "4:5", "1:1", "5:4", "4:3", "3:2", "16:9", "21:9", "4:1", "8:1"}
	ratioSelect := widget.NewSelect(ratios, func(string) {})
	ratioSelect.SetSelected("1:1")

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
		} else if len(s.Images) > 0 {
			// Fallback to first image if nothing selected
			selectedImages = s.Images[0].ID
			sourcePaths = s.Images[0].FullPath
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
			for i, task := range s.Tasks {
				if task.Status != "Pending" && task.Status != "Failed" {
					continue
				}

				if task.Mode == "Batch" {
					task.Status = "Submitted"
					s.BatchJobs = append(s.BatchJobs, &BatchJob{
						JobID:       fmt.Sprintf("Job_%d", task.ID),
						Status:      "Submitted",
						SubmittedAt: time.Now(),
						Progress:    "0%",
					})
					s.Log(fmt.Sprintf("[%d] Task submitted as Batch Job.", i+1))
					taskList.Refresh()
					continue
				}

				task.Status = "Running"
				s.Log(fmt.Sprintf("[%d] Running %s...", i+1, task.Agent))
				taskList.Refresh()
				err := s.RunTask(task)
				if err != nil {
					task.Status = "Failed"
					s.Log(fmt.Sprintf("Task %d failed: %v", task.ID, err))
				} else {
					task.Status = "Success"
				}
				taskList.Refresh()
			}
		}()
	})

	btnBox := container.NewHBox(newImgBtn, addTaskBtn, runBtn)

	taskEditor := widget.NewForm(
		widget.NewFormItem("Positive Prompt", promptEntry),
		widget.NewFormItem("Negative Prompt", negPromptEntry),
		widget.NewFormItem("Agent", agentSelect),
		widget.NewFormItem("Size", sizeSelect),
		widget.NewFormItem("Aspect Ratio", ratioSelect),
		widget.NewFormItem("Mode", modeSelect),
	)

	// Layout
	topHalf := container.New(layout.NewGridLayout(2), imageList, preview)
	middle := container.NewVBox(btnBox, taskList)
	right := container.NewVScroll(taskEditor)

	mainSplit := container.NewHSplit(
		container.NewVSplit(topHalf, middle),
		right,
	)
	mainSplit.Offset = 0.7

	return mainSplit
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
