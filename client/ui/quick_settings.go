package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func WrapWithQuickSettings(w fyne.Window, a fyne.App, mainWrapper *fyne.Container) *fyne.Container {
	isOpen := false
	var toggleBtn *widget.Button
	var settingsPanel *fyne.Container
	var overlay *fyne.Container

	closePanel := func() {
		if isOpen {
			panelHeight := float32(180)
			anim := canvas.NewPositionAnimation(
				fyne.NewPos(0, 0),
				fyne.NewPos(0, -panelHeight),
				time.Millisecond*250,
				settingsPanel.Move,
			)
			time.AfterFunc(time.Millisecond*250, func() {
				overlay.Hide()
			})
			anim.Start()
			toggleBtn.SetIcon(theme.MenuDropDownIcon())
			isOpen = false
		}
	}

	lockBtn := widget.NewButtonWithIcon("Lock Screen", theme.LogoutIcon(), func() {
		closePanel()
		pin := a.Preferences().String("app_pin")
		if pin == "" {
			pinEntry := widget.NewPasswordEntry()
			pinEntry.SetPlaceHolder("Enter 4-digit PIN")
			form := container.NewVBox(
				widget.NewLabel("Set a PIN to lock the application:"),
				pinEntry,
			)
			dialog.ShowCustomConfirm("Set PIN", "Save & Lock", "Cancel", form, func(confirm bool) {
				if confirm && len(pinEntry.Text) >= 4 {
					a.Preferences().SetString("app_pin", pinEntry.Text)
					lockApp(w, a, mainWrapper)
				}
			}, w)
			return
		}
		lockApp(w, a, mainWrapper)
	})
	lockBtn.Importance = widget.HighImportance

	panelContent := container.NewVBox(
		widget.NewLabelWithStyle("Quick Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		lockBtn,
		widget.NewSeparator(),
	)

	bg := canvas.NewRectangle(theme.BackgroundColor())
	bg.StrokeColor = theme.PrimaryColor()
	bg.StrokeWidth = 2

	settingsPanel = container.NewMax(bg, container.NewPadded(panelContent))
	overlay = container.NewWithoutLayout(settingsPanel)

	togglePanel := func() {
		windowSize := w.Canvas().Size()
		panelHeight := float32(180)

		settingsPanel.Resize(fyne.NewSize(windowSize.Width, panelHeight))

		if !isOpen {
			settingsPanel.Move(fyne.NewPos(0, -panelHeight))
			overlay.Show()

			anim := canvas.NewPositionAnimation(
				fyne.NewPos(0, -panelHeight),
				fyne.NewPos(0, 0),
				time.Millisecond*250,
				settingsPanel.Move,
			)
			anim.Start()
			toggleBtn.SetIcon(theme.MenuDropUpIcon())
			isOpen = true
		} else {
			closePanel()
		}
	}

	toggleBtn = widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), togglePanel)
	overlay.Hide()

	topBar := container.NewCenter(toggleBtn)
	baseLayout := container.NewBorder(topBar, nil, nil, nil, mainWrapper)

	return container.NewMax(baseLayout, overlay)
}

func lockApp(w fyne.Window, a fyne.App, mainWrapper *fyne.Container) {
	previousObjects := append([]fyne.CanvasObject{}, mainWrapper.Objects...)

	var lockView *fyne.Container
	lockView = BuildLockScreen(w, a, func() {
		mainWrapper.Objects = previousObjects
		mainWrapper.Refresh()
	})

	mainWrapper.Objects = []fyne.CanvasObject{lockView}
	mainWrapper.Refresh()
}
