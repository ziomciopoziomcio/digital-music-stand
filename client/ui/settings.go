package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildSettings(w fyne.Window, onClose func()) *fyne.Container {
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

	buildNetworkView := func() fyne.CanvasObject {
		scanBtn := widget.NewButtonWithIcon("Scan for Wi-Fi", theme.SearchIcon(), func() {})

		wifiList := container.NewVBox(
			widget.NewButton("Philharmonic_Guest", func() {}),
			widget.NewButton("Concert_Hall_Stage", func() {}),
			widget.NewButton("Free_WiFi", func() {}),
		)

		return container.NewBorder(container.NewPadded(scanBtn), nil, nil, nil, container.NewVScroll(wifiList))
	}

	buildMediaView := func() fyne.CanvasObject {
		volLabel := widget.NewLabel("Volume")
		volSlider := widget.NewSlider(0, 100)
		volSlider.SetValue(50)

		brightLabel := widget.NewLabel("Screen Brightness")
		brightSlider := widget.NewSlider(10, 100)
		brightSlider.SetValue(80)

		return container.NewVBox(
			volLabel,
			volSlider,
			widget.NewLabel(""),
			brightLabel,
			brightSlider,
		)
	}

	buildPowerView := func() fyne.CanvasObject {
		battLabel := widget.NewLabel("Battery Level: 85% (Charging)")
		battBar := widget.NewProgressBar()
		battBar.SetValue(0.85)

		return container.NewVBox(
			battLabel,
			battBar,
		)
	}

	buildSystemView := func() fyne.CanvasObject {
		awakeCheck := widget.NewCheck("Keep Screen Awake", func(checked bool) {})
		awakeCheck.SetChecked(true)

		rebootBtn := widget.NewButtonWithIcon("Reboot Device", theme.ViewRefreshIcon(), func() {})
		shutdownBtn := widget.NewButtonWithIcon("Shutdown Device", theme.CancelIcon(), func() {})
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

		netBtn.Importance = widget.HighImportance
		mediaBtn.Importance = widget.HighImportance
		powerBtn.Importance = widget.HighImportance
		sysBtn.Importance = widget.HighImportance

		grid := container.NewGridWithColumns(2, netBtn, mediaBtn, powerBtn, sysBtn)

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
