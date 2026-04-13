package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func makeBatchesTab(state *AppState) fyne.CanvasObject {
	fixedWidth := func(obj fyne.CanvasObject, width float32) fyne.CanvasObject {
		rect := canvas.NewRectangle(color.Transparent)
		rect.SetMinSize(fyne.NewSize(width, 0))
		return container.NewStack(rect, obj)
	}

	state.BatchList = widget.NewList(
		func() int { return len(state.BatchJobs) },
		func() fyne.CanvasObject {
			status := widget.NewLabel("")
			status.Truncation = fyne.TextTruncateEllipsis
			progress := widget.NewLabel("")
			progress.Truncation = fyne.TextTruncateEllipsis
			submitted := widget.NewLabel("")
			jobID := widget.NewLabel("")
			jobID.Truncation = fyne.TextTruncateEllipsis

			content := container.NewHBox(
				fixedWidth(status, 150),
				fixedWidth(progress, 100),
				fixedWidth(submitted, 120),
				fixedWidth(jobID, 500),
			)
			return content
		},
		func(id widget.ListItemID, cell fyne.CanvasObject) {
			job := state.BatchJobs[id]
			hbox := cell.(*fyne.Container)
			hbox.Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(job.Status)
			hbox.Objects[1].(*fyne.Container).Objects[1].(*widget.Label).SetText(job.Progress)
			hbox.Objects[2].(*fyne.Container).Objects[1].(*widget.Label).SetText(job.SubmittedAt.Format("15:04:05"))
			hbox.Objects[3].(*fyne.Container).Objects[1].(*widget.Label).SetText(job.JobID)
		},
	)

	headers := container.NewHBox(
		fixedWidth(widget.NewLabel("Status"), 150),
		fixedWidth(widget.NewLabel("Progress"), 100),
		fixedWidth(widget.NewLabel("Submitted"), 120),
		fixedWidth(widget.NewLabel("JobID"), 500),
	)

	clearBtn := widget.NewButton("Clear Completed", func() {
		newJobs := []*BatchJob{}
		for _, job := range state.BatchJobs {
			// Filter out all terminal states
			s := job.Status
			if s == "Success" || s == "Failed" || s == "SUCCEEDED" || s == "FAILED" || s == "CANCELLED" || s == "EXPIRED" {
				continue
			}
			newJobs = append(newJobs, job)
		}
		state.BatchJobs = newJobs
		state.BatchList.Refresh()
		state.CleanupJobsFile()
		state.Log("Cleared completed batch jobs.")
	})

	state.BatchProgressBar = widget.NewProgressBar()
	state.BatchProgressBar.Min = 0
	state.BatchProgressBar.Max = 1
	state.BatchStatusLabel = widget.NewLabel("No active batch jobs.")

	scroll := container.NewHScroll(container.NewBorder(headers, nil, nil, nil, state.BatchList))
	scroll.SetMinSize(fyne.NewSize(0, 400))

	bottom := container.NewVBox(
		state.BatchStatusLabel,
		state.BatchProgressBar,
		clearBtn,
	)

	return container.NewBorder(nil, bottom, nil, nil, scroll)
}
