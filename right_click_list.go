package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type RightClickList struct {
	*widget.List
	OnRightClick func(widget.ListItemID, fyne.Position)
}

func NewRightClickList(length func() int, create func() fyne.CanvasObject, update func(widget.ListItemID, fyne.CanvasObject)) *RightClickList {
	l := &RightClickList{}
	l.List = widget.NewList(length, create, update)
	return l
}

type tappableListItem struct {
	widget.BaseWidget
	content      fyne.CanvasObject
	id           widget.ListItemID
	onTapped     func(widget.ListItemID)
	onRightClick func(widget.ListItemID, fyne.Position)
}

func newTappableListItem(content fyne.CanvasObject, id widget.ListItemID, tapped func(widget.ListItemID), rightClick func(widget.ListItemID, fyne.Position)) *tappableListItem {
	item := &tappableListItem{content: content, id: id, onTapped: tapped, onRightClick: rightClick}
	item.ExtendBaseWidget(item)
	return item
}

func (i *tappableListItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(i.content)
}

func (i *tappableListItem) Tapped(pe *fyne.PointEvent) {
	if i.onTapped != nil {
		i.onTapped(i.id)
	}
}

func (i *tappableListItem) TappedSecondary(pe *fyne.PointEvent) {
	if i.onRightClick != nil {
		i.onRightClick(i.id, pe.AbsolutePosition)
	}
}
