package ui

import (
	"image/color"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	currentKbPopUp *widget.PopUp
	isShift        bool
)

func ShowKeyboard(target *widget.Entry) {
	if currentKbPopUp != nil {
		currentKbPopUp.Hide()
	}

	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	w := windows[0]

	mirrorLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	updateMirror := func() {
		txt := target.Text
		if txt == "" {
			mirrorLabel.SetText("...")
		} else if target.Password {
			mirrorLabel.SetText(strings.Repeat("*", utf8.RuneCountInString(txt)))
		} else {
			mirrorLabel.SetText(txt)
		}
	}
	updateMirror()

	mirrorBg := canvas.NewRectangle(theme.InputBackgroundColor())
	mirrorBg.CornerRadius = 8
	mirrorContainer := container.NewPadded(container.NewMax(mirrorBg, mirrorLabel))

	typeRune := func(r rune) {
		target.TypedRune(r)
		updateMirror()
	}

	typeKey := func(k fyne.KeyName) {
		target.TypedKey(&fyne.KeyEvent{Name: k})
		updateMirror()
	}

	kbContent := container.NewMax()

	var buildKeyboard func() *fyne.Container
	buildKeyboard = func() *fyne.Container {
		vbox := container.NewVBox()

		rows := []string{
			"1234567890-",
			"qwertyuiop+",
			"asdfghjkl_@",
			"zxcvbnm.!?",
		}

		for _, row := range rows {
			hbox := container.NewGridWithColumns(len(row))
			for _, char := range row {
				c := char
				if isShift && c >= 'a' && c <= 'z' {
					c = rune(strings.ToUpper(string(c))[0])
				}
				btn := widget.NewButton(string(c), func() {
					typeRune(c)
				})
				hbox.Add(btn)
			}
			vbox.Add(hbox)
		}

		shiftBtn := widget.NewButton("⇧ Shift", func() {
			isShift = !isShift
			kbContent.Objects = []fyne.CanvasObject{buildKeyboard()}
			kbContent.Refresh()
		})
		if isShift {
			shiftBtn.Importance = widget.HighImportance
		}

		spaceBtn := widget.NewButton("Space", func() {
			typeRune(' ')
		})

		backspaceBtn := widget.NewButton("⌫ Del", func() {
			typeKey(fyne.KeyBackspace)
		})

		closeBtn := widget.NewButton("✓ Enter", func() {
			HideKeyboard()
			if target.OnSubmitted != nil {
				target.OnSubmitted(target.Text)
			}
		})
		closeBtn.Importance = widget.HighImportance

		bottomRow := container.NewGridWithColumns(4, shiftBtn, spaceBtn, backspaceBtn, closeBtn)
		vbox.Add(bottomRow)

		return container.NewVBox(
			mirrorContainer,
			widget.NewSeparator(),
			vbox,
		)
	}

	kbContent.Objects = []fyne.CanvasObject{buildKeyboard()}

	bg := canvas.NewRectangle(theme.BackgroundColor())
	bg.CornerRadius = 12
	bg.StrokeColor = theme.PrimaryColor()
	bg.StrokeWidth = 2

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(750, 300))

	finalContainer := container.NewPadded(container.NewMax(bg, spacer, container.NewPadded(kbContent)))

	currentKbPopUp = widget.NewPopUp(finalContainer, w.Canvas())
	currentKbPopUp.Show()
}

func HideKeyboard() {
	if currentKbPopUp != nil {
		currentKbPopUp.Hide()
		currentKbPopUp = nil
	}
}

type AutoKeyboardEntry struct {
	widget.Entry
}

func NewAutoKeyboardEntry() *AutoKeyboardEntry {
	e := &AutoKeyboardEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *AutoKeyboardEntry) Tapped(pe *fyne.PointEvent) {
	e.Entry.Tapped(pe)
	ShowKeyboard(&e.Entry)
}

type AutoKeyboardPasswordEntry struct {
	widget.Entry
}

func NewAutoKeyboardPasswordEntry() *AutoKeyboardPasswordEntry {
	e := &AutoKeyboardPasswordEntry{}
	e.Password = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *AutoKeyboardPasswordEntry) Tapped(pe *fyne.PointEvent) {
	e.Entry.Tapped(pe)
	ShowKeyboard(&e.Entry)
}
