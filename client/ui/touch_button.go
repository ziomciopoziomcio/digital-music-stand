package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type TouchButton struct {
	widget.BaseWidget
	Text       string
	Icon       fyne.Resource
	Importance widget.ButtonImportance
	OnTapped   func()
	disabled   bool
	hovered    bool
}

func NewTouchButton(text string, tapped func()) *TouchButton {
	b := &TouchButton{
		Text:       text,
		OnTapped:   tapped,
		Importance: widget.MediumImportance,
	}
	b.ExtendBaseWidget(b)
	return b
}

func NewTouchButtonWithIcon(text string, icon fyne.Resource, tapped func()) *TouchButton {
	b := &TouchButton{
		Text:       text,
		Icon:       icon,
		OnTapped:   tapped,
		Importance: widget.MediumImportance,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *TouchButton) Disable() {
	b.disabled = true
	b.Refresh()
}

func (b *TouchButton) Enable() {
	b.disabled = false
	b.Refresh()
}

func (b *TouchButton) Disabled() bool {
	return b.disabled
}

func (b *TouchButton) SetText(text string) {
	b.Text = text
	b.Refresh()
}

func (b *TouchButton) SetIcon(icon fyne.Resource) {
	b.Icon = icon
	b.Refresh()
}

func (b *TouchButton) Tapped(_ *fyne.PointEvent) {
	if b.disabled {
		return
	}

	if app := fyne.CurrentApp(); app != nil {
		if driver := app.Driver(); driver != nil {
			for _, window := range driver.AllWindows() {
				if canvas := window.Canvas(); canvas != nil {
					canvas.Unfocus()
				}
			}
		}
	}

	if b.OnTapped != nil {
		b.OnTapped()
	}
}

func (b *TouchButton) MouseIn(_ *desktop.MouseEvent) {
	if b.disabled {
		return
	}
	b.hovered = true
	b.Refresh()
}

func (b *TouchButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

func (b *TouchButton) MouseMoved(_ *desktop.MouseEvent) {}

func (b *TouchButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = theme.InputRadiusSize()

	hoverBg := canvas.NewRectangle(color.Transparent)
	hoverBg.CornerRadius = theme.InputRadiusSize()
	hoverBg.Hide()

	icon := &canvas.Image{FillMode: canvas.ImageFillContain}
	icon.SetMinSize(fyne.NewSquareSize(theme.IconInlineSize()))

	text := canvas.NewText("", color.Transparent)
	text.Alignment = fyne.TextAlignCenter

	content := container.NewHBox(icon, text)
	paddedContent := container.NewPadded(content)

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(64, 40))

	c := container.NewMax(bg, hoverBg, spacer, container.NewCenter(paddedContent))

	r := &touchButtonRenderer{
		WidgetRenderer: widget.NewSimpleRenderer(c),
		button:         b,
		bg:             bg,
		hoverBg:        hoverBg,
		text:           text,
		icon:           icon,
	}
	r.Refresh()
	return r
}

type touchButtonRenderer struct {
	fyne.WidgetRenderer
	button  *TouchButton
	bg      *canvas.Rectangle
	hoverBg *canvas.Rectangle
	text    *canvas.Text
	icon    *canvas.Image
}

func (r *touchButtonRenderer) Refresh() {
	if r.button.Disabled() {
		r.bg.FillColor = color.Transparent
		r.text.Color = theme.DisabledColor()
	} else {
		switch r.button.Importance {
		case widget.HighImportance:
			r.bg.FillColor = theme.PrimaryColor()
			r.text.Color = color.White
		case widget.DangerImportance:
			r.bg.FillColor = theme.ErrorColor()
			r.text.Color = color.White
		case widget.LowImportance:
			r.bg.FillColor = color.Transparent
			r.text.Color = theme.ForegroundColor()
		default:
			r.bg.FillColor = theme.ButtonColor()
			r.text.Color = theme.ForegroundColor()
		}
	}

	if r.button.hovered && !r.button.Disabled() {
		r.hoverBg.FillColor = theme.HoverColor()
		r.hoverBg.Show()
	} else {
		r.hoverBg.Hide()
	}

	r.text.Text = r.button.Text
	if r.button.Text == "" {
		r.text.Hide()
	} else {
		r.text.Show()
	}
	r.text.Refresh()

	r.icon.Resource = r.button.Icon
	if r.button.Icon == nil {
		r.icon.Hide()
	} else {
		r.icon.Show()
	}
	r.icon.Refresh()

	r.bg.Refresh()
	r.hoverBg.Refresh()
}
