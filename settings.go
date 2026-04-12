package main

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ratioLayout struct {
	ratio float32
}

func (r *ratioLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	contentWidth := size.Width * r.ratio
	padding := (size.Width - contentWidth) / 2

	for _, o := range objects {
		o.Resize(fyne.NewSize(contentWidth, size.Height))
		o.Move(fyne.NewPos(padding, 0))
	}
}

func (r *ratioLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	min := objects[0].MinSize()
	return fyne.NewSize(min.Width/r.ratio, min.Height)
}

func makeSettingsTab(state *AppState) fyne.CanvasObject {
	def := DefaultConfig()

	makeReset := func(f func()) *widget.Button {
		b := widget.NewButton("Reset", f)
		b.Importance = widget.LowImportance
		return b
	}

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
	outputDirReset := makeReset(func() { outputDirEntry.SetText(def.OutputDir) })

	defPromptEntry := NewTabbableEntry()
	defPromptEntry.SetText(state.Config.DefaultPrompt)
	defPromptEntry.Wrapping = fyne.TextWrapWord
	defPromptReset := makeReset(func() { defPromptEntry.SetText(def.DefaultPrompt) })

	defNegEntry := NewTabbableEntry()
	defNegEntry.SetText(state.Config.DefaultNegPrompt)
	defNegEntry.Wrapping = fyne.TextWrapWord
	defNegReset := makeReset(func() { defNegEntry.SetText(def.DefaultNegPrompt) })

	encourageEdtEntry := NewTabbableEntry()
	encourageEdtEntry.SetText(state.Config.EncourageEdt)
	encourageEdtEntry.Wrapping = fyne.TextWrapWord
	encEdtReset := makeReset(func() { encourageEdtEntry.SetText(def.EncourageEdt) })

	encourageGenEntry := NewTabbableEntry()
	encourageGenEntry.SetText(state.Config.EncourageGen)
	encourageGenEntry.Wrapping = fyne.TextWrapWord
	encGenReset := makeReset(func() { encourageGenEntry.SetText(def.EncourageGen) })

	debugCheck := widget.NewCheck("Debug Mode", func(b bool) {})
	debugCheck.SetChecked(state.Config.Debug)
	debugReset := makeReset(func() { debugCheck.SetChecked(def.Debug) })

	// Generation Config
	tempLabel := widget.NewLabel(fmt.Sprintf("Temperature: %.1f", state.Config.Temperature))
	tempSlider := widget.NewSlider(0, 2)
	tempSlider.Step = 0.1
	tempSlider.Value = float64(state.Config.Temperature)
	tempSlider.OnChanged = func(v float64) {
		tempLabel.SetText(fmt.Sprintf("Temperature: %.1f", v))
	}
	tempReset := makeReset(func() { tempSlider.SetValue(float64(def.Temperature)) })

	topPLabel := widget.NewLabel(fmt.Sprintf("TopP: %.2f", state.Config.TopP))
	topPSlider := widget.NewSlider(0, 1)
	topPSlider.Step = 0.01
	topPSlider.Value = float64(state.Config.TopP)
	topPSlider.OnChanged = func(v float64) {
		topPLabel.SetText(fmt.Sprintf("TopP: %.2f", v))
	}
	topPReset := makeReset(func() { topPSlider.SetValue(float64(def.TopP)) })

	topKEntry := widget.NewEntry()
	topKEntry.SetText(strconv.Itoa(state.Config.TopK))
	topKReset := makeReset(func() { topKEntry.SetText(strconv.Itoa(def.TopK)) })

	maxTokensEntry := widget.NewEntry()
	maxTokensEntry.SetText(strconv.Itoa(state.Config.MaxOutputTokens))
	maxTokensReset := makeReset(func() { maxTokensEntry.SetText(strconv.Itoa(def.MaxOutputTokens)) })

	// Model Endpoints
	nanoFlashEntry := widget.NewEntry()
	nanoFlashEntry.SetText(state.Config.ModelNanoFlash)
	nanoFlashReset := makeReset(func() { nanoFlashEntry.SetText(def.ModelNanoFlash) })

	nanoProEntry := widget.NewEntry()
	nanoProEntry.SetText(state.Config.ModelNanoPro)
	nanoProReset := makeReset(func() { nanoProEntry.SetText(def.ModelNanoPro) })

	nano2Entry := widget.NewEntry()
	nano2Entry.SetText(state.Config.ModelNano2)
	nano2Reset := makeReset(func() { nano2Entry.SetText(def.ModelNano2) })

	imagenEntry := widget.NewEntry()
	imagenEntry.SetText(state.Config.ModelImagen)
	imagenReset := makeReset(func() { imagenEntry.SetText(def.ModelImagen) })

	imagenUltraEntry := widget.NewEntry()
	imagenUltraEntry.SetText(state.Config.ModelImagenUltra)
	imagenUltraReset := makeReset(func() { imagenUltraEntry.SetText(def.ModelImagenUltra) })

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

		state.Config.ModelNanoFlash = nanoFlashEntry.Text
		state.Config.ModelNanoPro = nanoProEntry.Text
		state.Config.ModelNano2 = nano2Entry.Text
		state.Config.ModelImagen = imagenEntry.Text
		state.Config.ModelImagenUltra = imagenUltraEntry.Text

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
		widget.NewFormItem("Output Directory", container.NewBorder(nil, nil, nil, outputDirReset, outputDirEntry)),
		widget.NewFormItem("Default Prompt", container.NewBorder(nil, nil, nil, defPromptReset, defPromptEntry)),
		widget.NewFormItem("Default Negative Prompt", container.NewBorder(nil, nil, nil, defNegReset, defNegEntry)),
		widget.NewFormItem("Encourage (Edit)", container.NewBorder(nil, nil, nil, encEdtReset, encourageEdtEntry)),
		widget.NewFormItem("Encourage (Generate)", container.NewBorder(nil, nil, nil, encGenReset, encourageGenEntry)),
		widget.NewFormItem("", container.NewHBox(debugCheck, debugReset)),
	)

	genBox := container.NewVBox()
	genBox.Add(widget.NewLabelWithStyle("Generation Parameters", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	genBox.Add(container.NewBorder(nil, nil, nil, tempReset, container.NewVBox(tempLabel, tempSlider)))
	genBox.Add(container.NewBorder(nil, nil, nil, topPReset, container.NewVBox(topPLabel, topPSlider)))
	genBox.Add(widget.NewForm(
		widget.NewFormItem("TopK", container.NewBorder(nil, nil, nil, topKReset, topKEntry)),
		widget.NewFormItem("Max Tokens", container.NewBorder(nil, nil, nil, maxTokensReset, maxTokensEntry)),
	))

	modelBox := container.NewVBox()
	modelBox.Add(widget.NewLabelWithStyle("Model Endpoints", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	modelBox.Add(widget.NewForm(
		widget.NewFormItem("Nano Flash", container.NewBorder(nil, nil, nil, nanoFlashReset, nanoFlashEntry)),
		widget.NewFormItem("Nano Pro", container.NewBorder(nil, nil, nil, nanoProReset, nanoProEntry)),
		widget.NewFormItem("Nano 2", container.NewBorder(nil, nil, nil, nano2Reset, nano2Entry)),
		widget.NewFormItem("Imagen", container.NewBorder(nil, nil, nil, imagenReset, imagenEntry)),
		widget.NewFormItem("Imagen Ultra", container.NewBorder(nil, nil, nil, imagenUltraReset, imagenUltraEntry)),
	))

	safetyBox := container.NewVBox()
	safetyBox.Add(widget.NewLabelWithStyle("Safety Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for cat, sel := range safetySelects {
		c := cat
		s := sel
		reset := makeReset(func() { s.SetSelected("BLOCK_NONE") })
		safetyBox.Add(widget.NewForm(widget.NewFormItem(c, container.NewBorder(nil, nil, nil, reset, s))))
	}

	content := container.NewVBox(
		form,
		genBox,
		modelBox,
		safetyBox,
	)

	centeredContent := container.New(&ratioLayout{ratio: 5.0 / 6.0}, content)
	scroll := container.NewVScroll(centeredContent)

	saveBtn.Importance = widget.HighImportance
	footer := container.NewPadded(container.New(&ratioLayout{ratio: 5.0 / 6.0}, saveBtn))

	return container.NewBorder(nil, footer, nil, nil, scroll)
}
