package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func makeSettingsTab(state *AppState) fyne.CanvasObject {
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetText(state.Config.APIKey)

	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetText(state.Config.OutputDir)

	encourageEdtEntry := widget.NewMultiLineEntry()
	encourageEdtEntry.SetText(state.Config.EncourageEdt)

	encourageGenEntry := widget.NewMultiLineEntry()
	encourageGenEntry.SetText(state.Config.EncourageGen)

	saveBtn := widget.NewButton("Save Configuration", func() {
		state.Config.APIKey = apiKeyEntry.Text
		state.Config.OutputDir = outputDirEntry.Text
		state.Config.EncourageEdt = encourageEdtEntry.Text
		state.Config.EncourageGen = encourageGenEntry.Text

		err := SaveConfig(state.Config)
		if err != nil {
			state.Log("Error saving config: " + err.Error())
		} else {
			state.Log("Configuration saved successfully.")
		}
	})

	form := widget.NewForm(
		widget.NewFormItem("Gemini API Key", apiKeyEntry),
		widget.NewFormItem("Output Directory", outputDirEntry),
		widget.NewFormItem("Encourage (Edit)", encourageEdtEntry),
		widget.NewFormItem("Encourage (Generate)", encourageGenEntry),
	)

	return container.NewVScroll(container.NewVBox(
		form,
		saveBtn,
	))
}
