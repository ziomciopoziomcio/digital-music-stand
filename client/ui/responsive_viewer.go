package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ResponsiveViewer struct {
	widget.BaseWidget
	Content  *fyne.Container
	onResize func(size fyne.Size)
	lastSize fyne.Size
}

func NewResponsiveViewer(onResize func(size fyne.Size)) *ResponsiveViewer {
	v := &ResponsiveViewer{
		onResize: onResize,
		Content:  container.NewMax(),
	}
	v.ExtendBaseWidget(v)
	return v
}

func (v *ResponsiveViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.Content)
}

func (v *ResponsiveViewer) Resize(size fyne.Size) {
	v.BaseWidget.Resize(size)
	v.Content.Resize(size)

	if v.onResize != nil && (size.Width != v.lastSize.Width || size.Height != v.lastSize.Height) {
		v.lastSize = size
		v.onResize(size)
	}
}
