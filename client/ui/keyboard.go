package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var (
	currentKbDialog dialog.Dialog
	isShift         bool
	kbContainer     *fyne.Container
)

func ShowKeyboard(target *widget.Entry) {
	if currentKbDialog != nil {
		return
	}

	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) == 0 {
		return
	}
	w := windows[0]

	kbContainer = container.NewMax()

	var refreshKeyboard func()
	refreshKeyboard = func() {
		rows := []string{
			"1234567890-",
			"qwertyuiop+",
			"asdfghjkl_",
			"zxcvbnm!?",
		}

		vbox := container.NewVBox()

		for _, row := range rows {
			hbox := container.NewHBox()
			for _, char := range row {
				c := char
				if isShift && c >= 'a' && c <= 'z' {
					c = rune(strings.ToUpper(string(c))[0])
				}
				btn := widget.NewButton(string(c), func() {
					target.TypedRune(c)
				})
				hbox.Add(btn)
			}
			vbox.Add(container.NewCenter(hbox))
		}

		shiftBtn := widget.NewButton("⇧ Shift", func() {
			isShift = !isShift
			refreshKeyboard()
		})
		if isShift {
			shiftBtn.Importance = widget.HighImportance
		}

		backspaceBtn := widget.NewButton("⌫ Del", func() {
			target.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
		})

		spaceBtn := widget.NewButton("          Space          ", func() {
			target.TypedRune(' ')
		})

		atBtn := widget.NewButton(" @ ", func() {
			target.TypedRune('@')
		})

		dotBtn := widget.NewButton(" . ", func() {
			target.TypedRune('.')
		})

		closeBtn := widget.NewButton("✓ Enter", func() {
			HideKeyboard()
			if target.OnSubmitted != nil {
				target.OnSubmitted(target.Text)
			}
		})
		closeBtn.Importance = widget.HighImportance

		bottomRow := container.NewCenter(container.NewHBox(shiftBtn, atBtn, dotBtn, spaceBtn, backspaceBtn, closeBtn))
		vbox.Add(bottomRow)

		kbContainer.Objects = []fyne.CanvasObject{container.NewPadded(vbox)}
		kbContainer.Refresh()
	}

	refreshKeyboard()

	currentKbDialog = dialog.NewCustomWithoutButtons("", kbContainer, w)
	currentKbDialog.Show()
}

func HideKeyboard() {
	if currentKbDialog != nil {
		currentKbDialog.Hide()
		currentKbDialog = nil
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
