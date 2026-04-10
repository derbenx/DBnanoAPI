package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type TabbableEntry struct {
	widget.Entry
}

func NewTabbableEntry() *TabbableEntry {
	e := &TabbableEntry{}
	e.ExtendBaseWidget(e)
	e.MultiLine = true
	return e
}

func (e *TabbableEntry) TypedKey(k *fyne.KeyEvent) {
	if k.Name == fyne.KeyTab {
		fyne.CurrentApp().Driver().CanvasForObject(e).FocusNext()
		return
	}
	e.Entry.TypedKey(k)
}
