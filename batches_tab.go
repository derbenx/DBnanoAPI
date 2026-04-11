package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func makeBatchesTab(state *AppState) fyne.CanvasObject {
	batchList := widget.NewList(
		func() int { return len(state.BatchJobs) },
		func() fyne.CanvasObject {
			jobID := widget.NewLabel("")
			status := widget.NewLabel("")
			submitted := widget.NewLabel("")
			progress := widget.NewLabel("")

			content := container.NewHBox(
				container.NewStack(jobID),
				container.NewStack(status),
				container.NewStack(submitted),
				container.NewStack(progress),
			)
			content.Objects[0].(*fyne.Container).SetMinSize(fyne.NewSize(250, 0))
			content.Objects[1].(*fyne.Container).SetMinSize(fyne.NewSize(150, 0))
			content.Objects[2].(*fyne.Container).SetMinSize(fyne.NewSize(100, 0))
			content.Objects[3].(*fyne.Container).SetMinSize(fyne.NewSize(80, 0))

			return content
		},
		func(id widget.ListItemID, cell fyne.CanvasObject) {
			job := state.BatchJobs[id]
			hbox := cell.(*fyne.Container)
			hbox.Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(job.JobID)
			hbox.Objects[1].(*fyne.Container).Objects[0].(*widget.Label).SetText(job.Status)
			hbox.Objects[2].(*fyne.Container).Objects[0].(*widget.Label).SetText(job.SubmittedAt.Format("15:04:05"))
			hbox.Objects[3].(*fyne.Container).Objects[0].(*widget.Label).SetText(job.Progress)
		},
	)

	headers := container.NewHBox(
		container.NewStack(widget.NewLabel("JobID")),
		container.NewStack(widget.NewLabel("Status")),
		container.NewStack(widget.NewLabel("Submitted")),
		container.NewStack(widget.NewLabel("Progress")),
	)
	headers.Objects[0].(*fyne.Container).SetMinSize(fyne.NewSize(250, 0))
	headers.Objects[1].(*fyne.Container).SetMinSize(fyne.NewSize(150, 0))
	headers.Objects[2].(*fyne.Container).SetMinSize(fyne.NewSize(100, 0))
	headers.Objects[3].(*fyne.Container).SetMinSize(fyne.NewSize(80, 0))

	clearBtn := widget.NewButton("Clear Completed", func() {
		newJobs := []*BatchJob{}
		for _, job := range state.BatchJobs {
			if job.Status != "Success" && job.Status != "Failed" && job.Status != "SUCCEEDED" && job.Status != "FAILED" && job.Status != "CANCELLED" {
				newJobs = append(newJobs, job)
			}
		}
		state.BatchJobs = newJobs
		batchList.Refresh()
		state.Log("Cleared completed batch jobs.")
	})

	state.BatchProgressBar = widget.NewProgressBar()
	state.BatchProgressBar.Min = 0
	state.BatchProgressBar.Max = 1
	state.BatchStatusLabel = widget.NewLabel("No active batch jobs.")

	scroll := container.NewHScroll(container.NewBorder(headers, nil, nil, nil, batchList))
	scroll.SetMinSize(fyne.NewSize(0, 400))

	bottom := container.NewVBox(
		state.BatchStatusLabel,
		state.BatchProgressBar,
		clearBtn,
	)

	return container.NewBorder(nil, bottom, nil, nil, scroll)
}
