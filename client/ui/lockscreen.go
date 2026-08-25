package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildLockScreen(w fyne.Window, a fyne.App, verifyPin func(string) bool, onSuccess func()) fyne.CanvasObject {
	contentWrapper := container.NewMax()

	timeText := canvas.NewText(time.Now().Format("15:04"), theme.ForegroundColor())
	timeText.TextSize = 96
	timeText.Alignment = fyne.TextAlignCenter
	timeText.TextStyle = fyne.TextStyle{Bold: true}

	dateText := canvas.NewText(time.Now().Format("Monday, 02 January 2006"), theme.ForegroundColor())
	dateText.TextSize = 24
	dateText.Alignment = fyne.TextAlignCenter

	stopClock := make(chan struct{})
	go func(stop <-chan struct{}) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				timeText.Text = time.Now().Format("15:04")
				timeText.Refresh()
				dateText.Text = time.Now().Format("Monday, 02 January 2006")
				dateText.Refresh()
			}
		}
	}(stopClock)

	pinEntry := NewAutoKeyboardPasswordEntry()
	pinEntry.Disable()
	pinEntry.SetPlaceHolder("Enter PIN")

	keypad := container.NewGridWithColumns(3)

	appendPin := func(digit string) {
		if len(pinEntry.Text) < 8 {
			pinEntry.SetText(pinEntry.Text + digit)
		}
	}

	for i := 1; i <= 9; i++ {
		digit := fmt.Sprintf("%d", i)
		btn := widget.NewButton(digit, func() {
			appendPin(digit)
		})
		keypad.Add(btn)
	}

	keypad.Add(widget.NewButton("C", func() {
		pinEntry.SetText("")
		pinEntry.SetPlaceHolder("Enter PIN")
	}))
	keypad.Add(widget.NewButton("0", func() {
		appendPin("0")
	}))

	unlockBtn := widget.NewButton("OK", func() {
		if verifyPin(pinEntry.Text) {
			close(stopClock)
			pinEntry.SetText("")
			onSuccess()
		} else {
			pinEntry.SetText("")
			pinEntry.SetPlaceHolder("Wrong PIN!")
		}
	})
	unlockBtn.Importance = widget.HighImportance
	keypad.Add(unlockBtn)

	clockContainer := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(timeText),
		container.NewCenter(dateText),
		layout.NewSpacer(),
	)

	padWidthConstraint := container.NewGridWrap(fyne.NewSize(250, 350), container.NewVBox(
		container.NewPadded(pinEntry),
		keypad,
	))

	centerLayout := container.NewGridWithColumns(2,
		clockContainer,
		container.NewCenter(padWidthConstraint),
	)

	contentWrapper.Objects = []fyne.CanvasObject{centerLayout}
	contentWrapper.Refresh()

	return contentWrapper
}
