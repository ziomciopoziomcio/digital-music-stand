package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
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

func NewTouchButtonIcon(icon fyne.Resource, onTapped func()) *TouchButton {
	b := &TouchButton{}
	b.Icon = icon
	b.OnTapped = onTapped
	b.ExtendBaseWidget(b)
	return b
}

func (b *TouchButton) MouseIn(_ *desktop.MouseEvent) {}

func (b *TouchButton) MouseMoved(_ *desktop.MouseEvent) {}

func (b *TouchButton) MouseOut() {}

func (b *TouchButton) FocusGained() {}

func (b *TouchButton) FocusLost() {}

func (b *TouchButton) MouseDown(_ *desktop.MouseEvent) {
	b.ignoreNextTap = true
	if b.OnTapped != nil {
		b.OnTapped()
	}
}

func (b *TouchButton) MouseUp(_ *desktop.MouseEvent) {}

func (b *TouchButton) Tapped(e *fyne.PointEvent) {
	if b.ignoreNextTap {
		b.ignoreNextTap = false
		return
	}
	if b.OnTapped != nil {
		b.OnTapped()
	}
}
