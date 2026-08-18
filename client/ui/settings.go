package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
)

func BuildSettings(w fyne.Window, app fyne.App, onClose func(), netMgr system.NetworkManager, pwrMgr system.PowerManager, medMgr system.MediaManager, devMgr system.DeviceManager) *fyne.Container {
	contentWrapper := container.NewMax()

	var showCategories func()

	showDetail := func(title string, detailContent fyne.CanvasObject) {
		backBtn := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
			showCategories()
		})
		backBtn.Importance = widget.WarningImportance

		titleLabel := widget.NewLabel(title)
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}

		header := container.NewBorder(nil, nil, backBtn, nil, container.NewCenter(titleLabel))
		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(detailContent))

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	buildSecurityView := func() fyne.CanvasObject {
		pinEntry := widget.NewPasswordEntry()
		pinEntry.SetPlaceHolder("Enter PIN (numbers only)")

		statusLabel := widget.NewLabel("")
		updateStatus := func() {
			savedPin := app.Preferences().String("app_pin")
			if savedPin != "" {
				statusLabel.SetText("Status: PIN Protection Active")
			} else {
				statusLabel.SetText("Status: PIN Protection Disabled")
			}
		}
		updateStatus()

		savePinBtn := widget.NewButtonWithIcon("Save PIN", theme.DocumentSaveIcon(), func() {
			if pinEntry.Text != "" {
				app.Preferences().SetString("app_pin", pinEntry.Text)
				dialog.ShowInformation("Security", "PIN updated successfully.", w)
				pinEntry.SetText("")
				updateStatus()
			}
		})
		savePinBtn.Importance = widget.HighImportance

		clearPinBtn := widget.NewButtonWithIcon("Remove PIN", theme.DeleteIcon(), func() {
			app.Preferences().SetString("app_pin", "")
			dialog.ShowInformation("Security", "PIN removed successfully.", w)
			pinEntry.SetText("")
			updateStatus()
		})
		clearPinBtn.Importance = widget.DangerImportance

		return container.NewVBox(
			statusLabel,
			widget.NewSeparator(),
			widget.NewLabel("New PIN Code:"),
			pinEntry,
			container.NewHBox(savePinBtn, clearPinBtn),
		)
	}

	buildNetworkView := func() fyne.CanvasObject {
		return widget.NewLabel("Network settings content...")
	}

	buildMediaView := func() fyne.CanvasObject {
		return widget.NewLabel("Display & Audio content...")
	}

	buildPowerView := func() fyne.CanvasObject {
		return widget.NewLabel("Power management content...")
	}

	buildSystemView := func() fyne.CanvasObject {
		awakeCheck := widget.NewCheck("Keep Device Awake", func(checked bool) {
			_ = devMgr.SetKeepAwake(checked)
		})
		awakeCheck.Checked = devMgr.IsKeepAwake()

		rebootBtn := widget.NewButtonWithIcon("Reboot System", theme.ViewRefreshIcon(), func() {
			_ = devMgr.Reboot()
		})
		rebootBtn.Importance = widget.WarningImportance

		shutdownBtn := widget.NewButtonWithIcon("Shutdown System", theme.CancelIcon(), func() {
			_ = devMgr.Shutdown()
		})
		shutdownBtn.Importance = widget.DangerImportance

		return container.NewVBox(
			awakeCheck,
			layout.NewSpacer(),
			rebootBtn,
			shutdownBtn,
		)
	}

	showCategories = func() {
		netBtn := widget.NewButtonWithIcon("Network & Wi-Fi", theme.ComputerIcon(), func() { showDetail("Network Settings", buildNetworkView()) })
		mediaBtn := widget.NewButtonWithIcon("Display & Audio", theme.ColorPaletteIcon(), func() { showDetail("Display & Audio", buildMediaView()) })
		powerBtn := widget.NewButtonWithIcon("Power", theme.InfoIcon(), func() { showDetail("Power Management", buildPowerView()) })
		sysBtn := widget.NewButtonWithIcon("System", theme.SettingsIcon(), func() { showDetail("System Controls", buildSystemView()) })
		secBtn := widget.NewButtonWithIcon("Security & PIN", theme.VisibilityOffIcon(), func() { showDetail("Security Settings", buildSecurityView()) })

		netBtn.Importance = widget.HighImportance
		mediaBtn.Importance = widget.HighImportance
		powerBtn.Importance = widget.HighImportance
		sysBtn.Importance = widget.HighImportance
		secBtn.Importance = widget.HighImportance

		grid := container.NewGridWithColumns(3, netBtn, mediaBtn, powerBtn, sysBtn, secBtn)

		closeBtn := widget.NewButtonWithIcon("Close Settings", theme.CancelIcon(), onClose)
		closeBtn.Importance = widget.DangerImportance
		header := container.NewBorder(nil, nil, nil, closeBtn, widget.NewLabelWithStyle("Settings", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(grid))

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	showCategories()
	return contentWrapper
}
