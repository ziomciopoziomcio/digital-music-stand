package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
)

func BuildSettings(w fyne.Window, onClose func(), netMgr system.NetworkManager, pwrMgr system.PowerManager, medMgr system.MediaManager, devMgr system.DeviceManager) *fyne.Container {
	//todo: handle errors
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

		wifiList := container.NewVBox()
		networks, _ := netMgr.GetAvailableNetworks()

		for _, net := range networks {
			ssid := net.SSID
			wifiList.Add(widget.NewButton(fmt.Sprintf("%s (Signal: %d%%)", ssid, net.Strength), func() {
				netMgr.ConnectWiFi(ssid, "mock_password")
			}))
		}

		return container.NewBorder(container.NewPadded(scanBtn), nil, nil, nil, container.NewVScroll(wifiList))
	}

	buildMediaView := func() fyne.CanvasObject {
		currentVol, _ := medMgr.GetVolume()
		volLabel := widget.NewLabel(fmt.Sprintf("Volume: %d%%", currentVol))
		volSlider := widget.NewSlider(0, 100)
		volSlider.SetValue(float64(currentVol))
		volSlider.OnChanged = func(val float64) {
			medMgr.SetVolume(int(val))
			volLabel.SetText(fmt.Sprintf("Volume: %d%%", int(val)))
		}

		currentBright, _ := medMgr.GetBrightness()
		brightLabel := widget.NewLabel(fmt.Sprintf("Screen Brightness: %d%%", currentBright))
		brightSlider := widget.NewSlider(10, 100)
		brightSlider.SetValue(float64(currentBright))
		brightSlider.OnChanged = func(val float64) {
			medMgr.SetBrightness(int(val))
			brightLabel.SetText(fmt.Sprintf("Screen Brightness: %d%%", int(val)))
		}

		return container.NewVBox(
			volLabel,
			volSlider,
			widget.NewLabel(""),
			brightLabel,
			brightSlider,
		)
	}

	buildPowerView := func() fyne.CanvasObject {
		battLevel, _ := pwrMgr.GetBatteryPercentage()
		charging, _ := pwrMgr.IsCharging()

		status := "Discharging"
		if charging {
			status = "Charging"
		}

		battLabel := widget.NewLabel(fmt.Sprintf("Battery Level: %d%% (%s)", battLevel, status))
		battBar := widget.NewProgressBar()
		battBar.SetValue(float64(battLevel) / 100.0)

		return container.NewVBox(
			battLabel,
			battBar,
		)
	}

	buildSystemView := func() fyne.CanvasObject {
		awakeCheck := widget.NewCheck("Keep Screen Awake", func(checked bool) {
			devMgr.SetKeepAwake(checked)
		})
		awakeCheck.SetChecked(true)

		rebootBtn := widget.NewButtonWithIcon("Reboot Device", theme.ViewRefreshIcon(), func() {
			devMgr.Reboot()
		})
		shutdownBtn := widget.NewButtonWithIcon("Shutdown Device", theme.CancelIcon(), func() {
			devMgr.Shutdown()
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
