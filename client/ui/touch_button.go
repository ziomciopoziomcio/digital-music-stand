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

var (
	_ fyne.Focusable    = (*TouchButton)(nil)
	_ fyne.Tappable     = (*TouchButton)(nil)
	_ desktop.Hoverable = (*TouchButton)(nil)
)

type TouchButton struct {
	widget.BaseWidget
	Text       string
	Icon       fyne.Resource
	Importance widget.ButtonImportance
	OnTapped   func()
	disabled   bool
	hovered    bool
	focused    bool
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
	b.focused = false
	b.hovered = false
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

// FocusGained marks the button as focused for keyboard navigation and triggers visual refresh.
func (b *TouchButton) FocusGained() {
	if b.disabled {
		return
	}
	b.focused = true
	b.Refresh()
}

// FocusLost clears the button focus state.
func (b *TouchButton) FocusLost() {
	b.focused = false
	b.Refresh()
}

// Focused returns whether the button currently has keyboard focus.
func (b *TouchButton) Focused() bool {
	return b.focused
}

// TypedRune implements fyne.Focusable.
func (b *TouchButton) TypedRune(_ rune) {}

// TypedKey implements fyne.Focusable so Enter and Space activate the button.
func (b *TouchButton) TypedKey(e *fyne.KeyEvent) {
	if b.disabled {
		return
	}
	if e.Name == fyne.KeySpace || e.Name == fyne.KeyReturn || e.Name == fyne.KeyEnter {
		b.Tapped(nil)
	}
}

// Tapped is triggered when the button is activated by a tap, click, or keyboard activation.
func (b *TouchButton) Tapped(_ *fyne.PointEvent) {
	if b.disabled {
		return
	}

	// Reset hover state so touchscreen taps don't leave lingering hover backgrounds.
	b.hovered = false
	b.Refresh()

	// Clear focus on this canvas if this button was focused, matching standard Fyne button behavior.
	if app := fyne.CurrentApp(); app != nil {
		if driver := app.Driver(); driver != nil {
			if c := driver.CanvasForObject(b); c != nil {
				c.Focus(nil)
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

	// Focus highlight indicator for keyboard navigation
	if r.button.focused && !r.button.Disabled() {
		r.bg.StrokeColor = theme.FocusColor()
		r.bg.StrokeWidth = 2
	} else {
		r.bg.StrokeColor = color.Transparent
		r.bg.StrokeWidth = 0
	}

	// Hover highlight for mouse pointers
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
