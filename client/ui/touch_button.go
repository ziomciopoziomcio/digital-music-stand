package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type TouchButton struct {
	widget.Button
}

func NewTouchButton(text string, onTapped func()) *TouchButton {
	b := &TouchButton{}
	b.Text = text
	b.OnTapped = onTapped
	b.ExtendBaseWidget(b)
	return b
}

func (b *TouchButton) MouseIn(_ *fyne.PointEvent) {}

func (b *TouchButton) MouseMoved(_ *fyne.PointEvent) {}

func (b *TouchButton) MouseOut() {}
