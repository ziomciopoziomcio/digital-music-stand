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

type qsLayout struct {
	panelHeight float32
}

func (l *qsLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, 0)
}

func (l *qsLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	backdrop := objects[0]
	panel := objects[1]

	backdrop.Resize(size)
	backdrop.Move(fyne.NewPos(0, 0))

	currentY := panel.Position().Y
	panel.Resize(fyne.NewSize(size.Width, l.panelHeight))
	panel.Move(fyne.NewPos(0, currentY))
}

func WrapWithQuickSettings(w fyne.Window, a fyne.App, mainWrapper *fyne.Container) *fyne.Container {
	isOpen := false
	globalVisible := true
	var toggleBtn *widget.Button
	var settingsPanel *fyne.Container
	var overlay *fyne.Container
	var backdrop *widget.Button
	var togglePanel func()

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
				if globalVisible {
					toggleBtn.Show()
				}
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

	overlay = container.New(&qsLayout{panelHeight: 220}, backdropContainer, settingsPanel)
	settingsPanel.Move(fyne.NewPos(0, -220))
	overlay.Hide()

	togglePanel = func() {
		panelHeight := float32(220)

		if !isOpen {
			settingsPanel.Move(fyne.NewPos(0, -panelHeight))
			overlay.Show()

			toggleBtn.Hide()

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

	toggleBtn = widget.NewButtonWithIcon("", theme.MenuDropDownIcon(), togglePanel)
	toggleBtn.Importance = widget.LowImportance

	SetQuickSettingsVisible = func(visible bool) {
		if visible {
			toggleBtn.Show()
		} else {
			toggleBtn.Hide()
			closePanel()
		}
	}

	floatingHandle := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			toggleBtn,
			layout.NewSpacer(),
		),
	)

	return container.NewMax(mainWrapper, overlay, floatingHandle)
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
