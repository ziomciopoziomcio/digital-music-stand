package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

type TouchButton struct {
	widget.Button
	ignoreNextTap bool
}

func NewTouchButton(text string, onTapped func()) *TouchButton {
	b := &TouchButton{}
	b.Text = text
	b.OnTapped = onTapped
	b.ExtendBaseWidget(b)
	return b
}

func NewTouchButtonWithIcon(text string, icon fyne.Resource, onTapped func()) *TouchButton {
	b := &TouchButton{}
	b.Text = text
	b.Icon = icon
	b.OnTapped = onTapped
	b.ExtendBaseWidget(b)
	return b
}

func (b *TouchButton) MouseIn(_ *fyne.PointEvent) {}

func (b *TouchButton) MouseMoved(_ *fyne.PointEvent) {}

func (b *TouchButton) MouseOut() {}

func (b *TouchButton) FocusGained() {}

func (b *TouchButton) FocusLost() {}

func (b *TouchButton) TouchDown(_ *mobile.TouchEvent) {
	b.ignoreNextTap = true
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

func (b *TouchButton) TouchUp(_ *mobile.TouchEvent) {}

func (b *TouchButton) TouchCancel(_ *mobile.TouchEvent) {}

func (b *TouchButton) Tapped(e *fyne.PointEvent) {
	if b.ignoreNextTap {
		b.ignoreNextTap = false
		return
	}
	if b.OnTapped != nil {
		b.OnTapped()
	}
}
