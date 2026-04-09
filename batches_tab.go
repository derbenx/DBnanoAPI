package main

import (
	"fyne.io/fyne/v2"
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

	return batchTable
}
