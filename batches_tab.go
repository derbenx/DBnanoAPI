package main

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func makeBatchesTab(state *AppState) fyne.CanvasObject {
	batchTable := widget.NewTable(
		func() (int, int) { return len(state.BatchJobs) + 1, 4 },
		func() fyne.CanvasObject { return widget.NewLabel("Header") },
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			if id.Row == 0 {
				headers := []string{"JobID", "Status", "Submitted", "Progress"}
				label.SetText(headers[id.Col])
				return
			}
			job := state.BatchJobs[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(job.JobID)
			case 1:
				label.SetText(job.Status)
			case 2:
				label.SetText(job.SubmittedAt.Format("15:04:05"))
			case 3:
				label.SetText(job.Progress)
			}
		},
	)

	clearBtn := widget.NewButton("Clear Completed", func() {
		newJobs := []*BatchJob{}
		for _, job := range state.BatchJobs {
			if job.Status != "Success" && job.Status != "Failed" && job.Status != "SUCCEEDED" && job.Status != "FAILED" && job.Status != "CANCELLED" {
				newJobs = append(newJobs, job)
			}
		}
		state.BatchJobs = newJobs
		batchTable.Refresh()
		state.Log("Cleared completed batch jobs.")
	})

	tableScroll := container.NewVScroll(batchTable)
	tableScroll.SetMinSize(fyne.NewSize(0, 400))

	return container.NewBorder(nil, clearBtn, nil, nil, tableScroll)
}
