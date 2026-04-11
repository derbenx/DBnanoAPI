package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type RightClickLabel struct {
	widget.Label
	id           widget.TableCellID
	table        *widget.Table
	OnRightClick func(widget.TableCellID, fyne.Position)
}

func NewRightClickLabel(text string, t *widget.Table) *RightClickLabel {
	l := &RightClickLabel{table: t}
	l.ExtendBaseWidget(l)
	l.Text = text
	l.Truncation = fyne.TextTruncateEllipsis
	return l
}

func (l *RightClickLabel) Tapped(pe *fyne.PointEvent) {
	if l.id.Row == 0 {
		return
	}
	// Pass click to table for selection logic
	l.table.Select(l.id)
}

func (l *RightClickLabel) TappedSecondary(pe *fyne.PointEvent) {
	if l.OnRightClick != nil {
		l.OnRightClick(l.id, pe.AbsolutePosition)
	}
}

type RightClickTable struct {
	*widget.Table
}

func NewRightClickTable(length func() (int, int), create func(*widget.Table) fyne.CanvasObject, update func(widget.TableCellID, fyne.CanvasObject)) *RightClickTable {
	t := &widget.Table{}
	t.Length = length
	t.CreateCell = func() fyne.CanvasObject { return create(t) }
	t.UpdateCell = update
	return &RightClickTable{Table: t}
}
