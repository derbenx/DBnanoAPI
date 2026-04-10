package main

import (
	"fmt"
	"strconv"

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

	// Generation Config
	tempLabel := widget.NewLabel(fmt.Sprintf("Temperature: %.1f", state.Config.Temperature))
	tempSlider := widget.NewSlider(0, 2)
	tempSlider.Step = 0.1
	tempSlider.Value = float64(state.Config.Temperature)
	tempSlider.OnChanged = func(v float64) {
		tempLabel.SetText(fmt.Sprintf("Temperature: %.1f", v))
	}

	topPLabel := widget.NewLabel(fmt.Sprintf("TopP: %.2f", state.Config.TopP))
	topPSlider := widget.NewSlider(0, 1)
	topPSlider.Step = 0.01
	topPSlider.Value = float64(state.Config.TopP)
	topPSlider.OnChanged = func(v float64) {
		topPLabel.SetText(fmt.Sprintf("TopP: %.2f", v))
	}

	topKEntry := widget.NewEntry()
	topKEntry.SetText(strconv.Itoa(state.Config.TopK))

	maxTokensEntry := widget.NewEntry()
	maxTokensEntry.SetText(strconv.Itoa(state.Config.MaxOutputTokens))

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

		state.Config.Temperature = float32(tempSlider.Value)
		state.Config.TopP = float32(topPSlider.Value)
		if tk, err := strconv.Atoi(topKEntry.Text); err == nil {
			state.Config.TopK = tk
		}
		if mt, err := strconv.Atoi(maxTokensEntry.Text); err == nil {
			state.Config.MaxOutputTokens = mt
		}

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

	genBox := container.NewVBox()
	genBox.Add(widget.NewLabelWithStyle("Generation Parameters", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	genBox.Add(container.NewVBox(tempLabel, tempSlider))
	genBox.Add(container.NewVBox(topPLabel, topPSlider))
	genBox.Add(widget.NewForm(
		widget.NewFormItem("TopK", topKEntry),
		widget.NewFormItem("Max Tokens", maxTokensEntry),
	))

	safetyBox := container.NewVBox()
	safetyBox.Add(widget.NewLabelWithStyle("Safety Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for cat, sel := range safetySelects {
		safetyBox.Add(widget.NewForm(widget.NewFormItem(cat, sel)))
	}

	content := container.NewVBox(
		form,
		genBox,
		safetyBox,
		saveBtn,
	)
	scroll := container.NewVScroll(content)
	return scroll
}
