package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var SetQuickSettingsVisible func(visible bool)

func WrapWithQuickSettings(w fyne.Window, a fyne.App, mainWrapper *fyne.Container) *fyne.Container {
	isOpen := false
	var toggleBtn *widget.Button
	var settingsPanel *fyne.Container
	var overlay *fyne.Container
	var backdrop *widget.Button

	closePanel := func() {
		if isOpen {
			panelHeight := float32(220)
			anim := canvas.NewPositionAnimation(
				fyne.NewPos(0, 0),
				fyne.NewPos(0, -panelHeight),
				time.Millisecond*200,
				settingsPanel.Move,
			)
			time.AfterFunc(time.Millisecond*200, func() {
				overlay.Hide()
			})
			anim.Start()
			toggleBtn.SetIcon(theme.SettingsIcon())
			isOpen = false
		}
	}

	lockBtn := widget.NewButtonWithIcon("Lock Screen", theme.LogoutIcon(), func() {
		closePanel()
		pin := a.Preferences().String("app_pin")
		if pin == "" {
			pinEntry := NewAutoKeyboardPasswordEntry()
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

	closeQuickSettingsBtn := widget.NewButtonWithIcon("Close Quick Settings", theme.CancelIcon(), func() {
		closePanel()
	})

	panelContent := container.NewVBox(
		widget.NewLabelWithStyle("Quick Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		lockBtn,
		widget.NewSeparator(),
		closeQuickSettingsBtn,
	)

	bg := canvas.NewRectangle(theme.BackgroundColor())
	bg.StrokeColor = theme.PrimaryColor()
	bg.StrokeWidth = 2

	settingsPanel = container.NewMax(bg, container.NewPadded(panelContent))

	backdrop = widget.NewButton("", func() {
		closePanel()
	})
	backdropBg := canvas.NewRectangle(color.Transparent)
	backdropContainer := container.NewMax(backdropBg, backdrop)

	overlay = container.NewWithoutLayout(backdropContainer, settingsPanel)

	togglePanel := func() {
		windowSize := w.Canvas().Size()
		panelHeight := float32(220)

		backdropContainer.Resize(windowSize)
		backdropContainer.Move(fyne.NewPos(0, 0))

		settingsPanel.Resize(fyne.NewSize(windowSize.Width, panelHeight))

		if !isOpen {
			settingsPanel.Move(fyne.NewPos(0, -panelHeight))
			overlay.Show()

			anim := canvas.NewPositionAnimation(
				fyne.NewPos(0, -panelHeight),
				fyne.NewPos(0, 0),
				time.Millisecond*200,
				settingsPanel.Move,
			)
			anim.Start()
			toggleBtn.SetIcon(theme.MenuDropUpIcon())
			isOpen = true
		} else {
			closePanel()
		}
	}

	toggleBtn = widget.NewButtonWithIcon("", theme.SettingsIcon(), togglePanel)
	toggleBtn.Importance = widget.LowImportance
	overlay.Hide()

	SetQuickSettingsVisible = func(visible bool) {
		if visible {
			toggleBtn.Show()
		} else {
			toggleBtn.Hide()
			closePanel()
		}
	}

	floatingBtn := container.NewVBox(
		container.NewHBox(
			toggleBtn,
			layout.NewSpacer(),
		),
	)

	return container.NewMax(mainWrapper, floatingBtn, overlay)
}

func lockApp(w fyne.Window, a fyne.App, mainWrapper *fyne.Container) {
	previousObjects := append([]fyne.CanvasObject{}, mainWrapper.Objects...)

	if SetQuickSettingsVisible != nil {
		SetQuickSettingsVisible(false)
	}

	var lockView *fyne.Container
	lockView = BuildLockScreen(w, a, func() {
		mainWrapper.Objects = previousObjects
		mainWrapper.Refresh()

		if SetQuickSettingsVisible != nil {
			SetQuickSettingsVisible(true)
		}
	})

	mainWrapper.Objects = []fyne.CanvasObject{lockView}
	mainWrapper.Refresh()
}
