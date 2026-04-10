package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func makeSettingsTab(state *AppState) fyne.CanvasObject {
	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetText(state.Config.APIKey)

	testBtn := widget.NewButton("Test API Key", func() {
		state.Config.APIKey = apiKeyEntry.Text
		go func() {
			err := state.TestAPI()
			if err != nil {
				state.Log("Test Failed: " + err.Error())
			}
		}()
	})

	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetText(state.Config.OutputDir)

	defPromptEntry := NewTabbableEntry()
	defPromptEntry.SetText(state.Config.DefaultPrompt)
	defPromptEntry.Wrapping = fyne.TextWrapWord

	defNegEntry := NewTabbableEntry()
	defNegEntry.SetText(state.Config.DefaultNegPrompt)
	defNegEntry.Wrapping = fyne.TextWrapWord

	encourageEdtEntry := NewTabbableEntry()
	encourageEdtEntry.SetText(state.Config.EncourageEdt)
	encourageEdtEntry.Wrapping = fyne.TextWrapWord

	encourageGenEntry := NewTabbableEntry()
	encourageGenEntry.SetText(state.Config.EncourageGen)
	encourageGenEntry.Wrapping = fyne.TextWrapWord

	debugCheck := widget.NewCheck("Debug Mode", func(b bool) {})
	debugCheck.SetChecked(state.Config.Debug)

	// Safety Settings
	thresholds := []string{"BLOCK_NONE", "BLOCK_ONLY_HIGH", "BLOCK_MEDIUM_AND_ABOVE", "BLOCK_LOW_AND_ABOVE"}

	safetySelects := make(map[string]*widget.Select)
	for _, s := range state.Config.SafetySettings {
		sel := widget.NewSelect(thresholds, func(string) {})
		sel.SetSelected(s.Threshold)
		safetySelects[s.Category] = sel
	}

	saveBtn := widget.NewButton("Save Configuration", func() {
		state.Config.APIKey = apiKeyEntry.Text
		state.Config.OutputDir = outputDirEntry.Text
		state.Config.DefaultPrompt = defPromptEntry.Text
		state.Config.DefaultNegPrompt = defNegEntry.Text
		state.Config.EncourageEdt = encourageEdtEntry.Text
		state.Config.EncourageGen = encourageGenEntry.Text
		state.Config.Debug = debugCheck.Checked

		for i, s := range state.Config.SafetySettings {
			if sel, ok := safetySelects[s.Category]; ok {
				state.Config.SafetySettings[i].Threshold = sel.Selected
			}
		}

		err := SaveConfig(state.Config)
		if err != nil {
			state.Log("Error saving config: " + err.Error())
		} else {
			state.Log("Configuration saved successfully.")
		}
	})

	form := widget.NewForm(
		widget.NewFormItem("Gemini API Key", container.NewBorder(nil, nil, nil, testBtn, apiKeyEntry)),
		widget.NewFormItem("Output Directory", outputDirEntry),
		widget.NewFormItem("Default Prompt", defPromptEntry),
		widget.NewFormItem("Default Negative Prompt", defNegEntry),
		widget.NewFormItem("Encourage (Edit)", encourageEdtEntry),
		widget.NewFormItem("Encourage (Generate)", encourageGenEntry),
		widget.NewFormItem("", debugCheck),
	)

	safetyBox := container.NewVBox()
	safetyBox.Add(widget.NewLabelWithStyle("Safety Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for cat, sel := range safetySelects {
		safetyBox.Add(widget.NewForm(widget.NewFormItem(cat, sel)))
	}

	return container.NewVScroll(container.NewVBox(
		form,
		safetyBox,
		saveBtn,
	))
}
