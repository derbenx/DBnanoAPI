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
	l.table.Select(l.id)
}

func (l *RightClickLabel) TappedSecondary(pe *fyne.PointEvent) {
	if l.OnRightClick != nil {
		l.OnRightClick(l.id, pe.AbsolutePosition)
	}
}

type RightClickTable struct {
	*widget.Table
	selectedID widget.TableCellID
	OnSelected func(widget.TableCellID)
}

func NewRightClickTable(length func() (int, int), create func(*widget.Table) fyne.CanvasObject, update func(widget.TableCellID, fyne.CanvasObject)) *RightClickTable {
	t := &widget.Table{}
	t.Length = length
	t.CreateCell = func() fyne.CanvasObject { return create(t) }
	t.UpdateCell = update

	rt := &RightClickTable{Table: t, selectedID: widget.TableCellID{Row: -1, Col: -1}}

	t.OnSelected = func(id widget.TableCellID) {
		rt.selectedID = id
		if rt.OnSelected != nil {
			rt.OnSelected(id)
		}
	}

	return rt
}

func (t *RightClickTable) UnselectAll() {
	if t.selectedID.Row != -1 {
		t.Table.Unselect(t.selectedID)
		t.selectedID = widget.TableCellID{Row: -1, Col: -1}
	}
}
