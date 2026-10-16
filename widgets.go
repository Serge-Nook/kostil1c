package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// tapArea wraps any object and calls onTap when it is clicked.
type tapArea struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newTapArea(content fyne.CanvasObject, onTap func()) *tapArea {
	t := &tapArea{content: content, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func (t *tapArea) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func layoutSpacer() fyne.CanvasObject { return layout.NewSpacer() }
