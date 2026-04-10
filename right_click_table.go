package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type RightClickLabel struct {
	widget.Label
	id           widget.TableCellID
	OnRightClick func(widget.TableCellID, fyne.Position)
}

func NewRightClickLabel(text string) *RightClickLabel {
	l := &RightClickLabel{}
	l.ExtendBaseWidget(l)
	l.Text = text
	return l
}

func (l *RightClickLabel) TappedSecondary(pe *fyne.PointEvent) {
	if l.OnRightClick != nil {
		l.OnRightClick(l.id, pe.AbsolutePosition)
	}
}

type RightClickTable struct {
	*widget.Table
}

func NewRightClickTable(length func() (int, int), create func() fyne.CanvasObject, update func(widget.TableCellID, fyne.CanvasObject)) *RightClickTable {
	return &RightClickTable{
		Table: widget.NewTable(length, create, update),
	}
}
