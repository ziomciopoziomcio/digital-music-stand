package ui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/plugins"
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

func WrapWithQuickSettings(w fyne.Window, a fyne.App, content fyne.CanvasObject, profileID string, onLock func(), onSwitchProfile func(), isCloudConnected func() bool) fyne.CanvasObject {
	isOpen := false
	globalVisible := true

	var toggleBtn *widget.Button
	var settingsPanel *fyne.Container
	var overlay *fyne.Container
	var backdrop *widget.Button
	var togglePanel func()

	panelHeightVal := float32(280)

	closePanel := func() {
		if isOpen {
			anim := canvas.NewPositionAnimation(
				fyne.NewPos(0, 0),
				fyne.NewPos(0, -panelHeightVal),
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

	switchBtn := widget.NewButtonWithIcon("Switch Profile", theme.AccountIcon(), func() {
		closePanel()
		if onSwitchProfile != nil {
			onSwitchProfile()
		}
	})
	switchBtn.Importance = widget.WarningImportance

	lockBtn := widget.NewButtonWithIcon("Lock Screen", theme.LogoutIcon(), func() {
		closePanel()
		if onLock != nil {
			onLock()
		}
	})
	lockBtn.Importance = widget.HighImportance

	closeQuickSettingsBtn := widget.NewButtonWithIcon("Close Quick Settings", theme.CancelIcon(), func() {
		closePanel()
	})

	statusLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	mixerTitle := widget.NewLabelWithStyle("Stage Mixer (Master Volume)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	masterVolSlider := widget.NewSlider(0, 1)
	masterVolSlider.Step = 0.01

	formatDB := func(val float64) string {
		if val <= 0.01 {
			return "-∞ dB"
		}
		if val >= 0.75 {
			db := (val - 0.75) * 40.0
			if db == 0 {
				return "0.0 dB"
			}
			return fmt.Sprintf("+%.1f dB", db)
		}
		db := (val/0.75)*60.0 - 60.0
		return fmt.Sprintf("%.1f dB", db)
	}

	masterDbLabel := widget.NewLabelWithStyle(formatDB(0), fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})

	var ignoreSliderChange bool
	masterVolSlider.OnChanged = func(val float64) {
		masterDbLabel.SetText(formatDB(val))
		if ignoreSliderChange {
			return
		}
		if m := plugins.GetActiveMixer(); m != nil {
			_ = m.SetMainVolume(val)
		}
	}

	personalMixerBtn := widget.NewButtonWithIcon("Open Personal Monitor Mix", theme.SettingsIcon(), func() {
		closePanel()
		ShowPersonalMixerDialog(w, a, profileID)
	})
	personalMixerBtn.Importance = widget.HighImportance

	mixerContainer := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(mixerTitle, layout.NewSpacer(), masterDbLabel),
		masterVolSlider,
		widget.NewLabel(""),
		personalMixerBtn,
		widget.NewSeparator(),
	)
	mixerContainer.Hide()

	panelContent := container.NewVBox(
		widget.NewLabelWithStyle("Quick Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusLabel,
		mixerContainer,
		switchBtn,
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

	overlay = container.New(&qsLayout{panelHeight: panelHeightVal}, backdropContainer, settingsPanel)
	settingsPanel.Move(fyne.NewPos(0, -panelHeightVal))
	overlay.Hide()

	togglePanel = func() {
		if !isOpen {
			if isCloudConnected != nil && isCloudConnected() {
				statusLabel.SetText("Cloud: Connected")
			} else {
				statusLabel.SetText("Cloud: Disconnected")
			}

			m := plugins.GetActiveMixer()
			if m != nil && m.GetConnectionStatus() {
				mixerContainer.Show()
				mixerTitle.SetText(fmt.Sprintf("%s (Master)", m.Name()))
				if vol, err := m.GetMainVolume(); err == nil {
					ignoreSliderChange = true
					masterVolSlider.SetValue(vol)
					masterDbLabel.SetText(formatDB(vol))
					ignoreSliderChange = false
				}
			} else {
				mixerContainer.Hide()
			}

			settingsPanel.Move(fyne.NewPos(0, -panelHeightVal))
			overlay.Show()
			toggleBtn.Hide()

			anim := canvas.NewPositionAnimation(
				fyne.NewPos(0, -panelHeightVal),
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
		globalVisible = visible
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

	return container.NewMax(content, overlay, floatingHandle)
}
