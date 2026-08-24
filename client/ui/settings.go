package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/updater"
)

func BuildSettings(w fyne.Window, app fyne.App, currentVersion string, onClose func(), netMgr system.NetworkManager, pwrMgr system.PowerManager, medMgr system.MediaManager, devMgr system.DeviceManager) *fyne.Container {
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
		pinEntry := NewAutoKeyboardPasswordEntry()
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
		statusLabel := widget.NewLabel(fmt.Sprintf("Status: %s", netMgr.GetNetworkStatus()))
		listContainer := container.NewVBox()

		refreshNetworks := func() {
			listContainer.Objects = []fyne.CanvasObject{widget.NewLabel("Scanning for networks...")}
			listContainer.Refresh()

			go func() {
				networks, err := netMgr.GetAvailableNetworks()
				var objs []fyne.CanvasObject

				if err != nil {
					objs = append(objs, widget.NewLabel(fmt.Sprintf("Failed to scan: %v", err)))
				} else if len(networks) == 0 {
					objs = append(objs, widget.NewLabel("No networks found."))
				} else {
					for _, n := range networks {
						net := n
						icon := theme.ComputerIcon()

						btn := widget.NewButtonWithIcon(fmt.Sprintf("%s (%d%%)", net.SSID, net.Strength), icon, func() {
							if net.Secure {
								passEntry := NewAutoKeyboardPasswordEntry()
								passEntry.SetPlaceHolder("Wi-Fi Password")

								dialog.ShowCustomConfirm("Connect to "+net.SSID, "Connect", "Cancel", passEntry, func(confirm bool) {
									if confirm {
										err := netMgr.ConnectWiFi(net.SSID, passEntry.Text)
										if err != nil {
											dialog.ShowError(fmt.Errorf("Failed to connect: %v", err), w)
										}
										statusLabel.SetText(fmt.Sprintf("Status: %s", netMgr.GetNetworkStatus()))
									}
								}, w)
							} else {
								err := netMgr.ConnectWiFi(net.SSID, "")
								if err != nil {
									dialog.ShowError(fmt.Errorf("Failed to connect: %v", err), w)
								}
								statusLabel.SetText(fmt.Sprintf("Status: %s", netMgr.GetNetworkStatus()))
							}
						})
						objs = append(objs, btn)
					}
				}

				listContainer.Objects = objs
				listContainer.Refresh()
			}()
		}

		refreshBtn := widget.NewButtonWithIcon("Scan Networks", theme.SearchIcon(), refreshNetworks)
		refreshBtn.Importance = widget.HighImportance

		disconnectBtn := widget.NewButtonWithIcon("Disconnect", theme.CancelIcon(), func() {
			_ = netMgr.Disconnect()
			statusLabel.SetText(fmt.Sprintf("Status: %s", netMgr.GetNetworkStatus()))
		})
		disconnectBtn.Importance = widget.DangerImportance

		topBar := container.NewHBox(statusLabel, layout.NewSpacer(), disconnectBtn, refreshBtn)

		refreshNetworks()

		return container.NewBorder(topBar, nil, nil, nil, container.NewVScroll(listContainer))
	}

	buildMediaView := func() fyne.CanvasObject {
		vol, _ := medMgr.GetVolume()
		bright, _ := medMgr.GetBrightness()

		volLabel := widget.NewLabel(fmt.Sprintf("Volume: %d%%", vol))
		volSlider := widget.NewSlider(0, 100)
		volSlider.SetValue(float64(vol))
		volSlider.OnChanged = func(val float64) {
			newVol := int(val)
			_ = medMgr.SetVolume(newVol)
			volLabel.SetText(fmt.Sprintf("Volume: %d%%", newVol))
		}

		brightLabel := widget.NewLabel(fmt.Sprintf("Brightness: %d%%", bright))
		brightSlider := widget.NewSlider(0, 100)
		brightSlider.SetValue(float64(bright))
		brightSlider.OnChanged = func(val float64) {
			newBright := int(val)
			_ = medMgr.SetBrightness(newBright)
			brightLabel.SetText(fmt.Sprintf("Brightness: %d%%", newBright))
		}

		return container.NewVBox(
			volLabel,
			volSlider,
			widget.NewSeparator(),
			brightLabel,
			brightSlider,
		)
	}

	buildPowerView := func() fyne.CanvasObject {
		batLevel, err := pwrMgr.GetBatteryPercentage()
		batText := fmt.Sprintf("Battery Level: %d%%", batLevel)
		if err != nil {
			batText = "Battery Level: Unknown / Desktop"
		}

		charging, err := pwrMgr.IsCharging()
		chargeText := "Status: Discharging"
		if charging {
			chargeText = "Status: Charging / AC Power"
		}
		if err != nil {
			chargeText = "Status: Unknown"
		}

		batLabel := canvas.NewText(batText, theme.ForegroundColor())
		batLabel.TextSize = 24
		batLabel.TextStyle = fyne.TextStyle{Bold: true}

		return container.NewVBox(
			batLabel,
			widget.NewLabel(chargeText),
		)
	}

	buildSystemView := func() fyne.CanvasObject {
		awakeCheck := widget.NewCheck("Keep Device Awake", func(checked bool) {
			_ = devMgr.SetKeepAwake(checked)
		})
		awakeCheck.Checked = devMgr.IsKeepAwake()

		systemElements := []fyne.CanvasObject{
			awakeCheck,
			widget.NewSeparator(),
		}

		rebootBtn := widget.NewButtonWithIcon("Reboot System", theme.ViewRefreshIcon(), func() {
			_ = devMgr.Reboot()
		})
		rebootBtn.Importance = widget.WarningImportance

		shutdownBtn := widget.NewButtonWithIcon("Shutdown System", theme.CancelIcon(), func() {
			_ = devMgr.Shutdown()
		})
		shutdownBtn.Importance = widget.DangerImportance

		systemElements = append(systemElements,
			layout.NewSpacer(),
			rebootBtn,
			shutdownBtn,
		)

		return container.NewVBox(
			systemElements...,
		)
	}

	buildUpdateView := func() fyne.CanvasObject {
		versionLabel := widget.NewLabel(fmt.Sprintf("Current Version: %s", currentVersion))

		updateBtn := widget.NewButtonWithIcon("Check for Updates", theme.DownloadIcon(), func() {

			loadingContent := container.NewVBox(
				widget.NewLabel("Looking for updates on GitHub..."),
				widget.NewProgressBarInfinite(),
			)
			loadingDialog := dialog.NewCustomWithoutButtons("Checking", container.NewPadded(loadingContent), w)
			loadingDialog.Show()

			owner := "ziomciopoziomcio"
			repo := "digital-music-stand"

			go func() {
				hasUpdate, newVer, downloadURL, err := updater.CheckForUpdates(owner, repo, currentVersion)

				loadingDialog.Hide()

				if err != nil {
					log.Println("Update check error:", err)
					dialog.ShowError(fmt.Errorf("failed to check for updates: %v", err), w)
					return
				}

				if !hasUpdate {
					dialog.ShowInformation("Up to date", "You are running the latest version.", w)
					return
				}

				dialog.ShowConfirm("Update Available", fmt.Sprintf("Version %s is available. Do you want to download and restart now?", newVer), func(confirm bool) {
					if confirm {
						downloadContent := container.NewVBox(
							widget.NewLabel("Downloading and applying update..."),
							widget.NewProgressBarInfinite(),
						)
						progressDialog := dialog.NewCustomWithoutButtons("Downloading Update", container.NewPadded(downloadContent), w)
						progressDialog.Show()

						go func() {
							err := updater.DoUpdate(downloadURL)
							progressDialog.Hide()

							if err != nil {
								dialog.ShowError(fmt.Errorf("update failed: %v", err), w)
								return
							}

							dialog.ShowInformation("Success", "Update installed. Please close and restart the application.", w)
						}()
					}
				}, w)
			}()
		})
		updateBtn.Importance = widget.HighImportance

		return container.NewVBox(
			versionLabel,
			widget.NewSeparator(),
			updateBtn,
		)
	}

	showCategories = func() {
		missingDeps := system.CheckMissingDependencies()

		if len(missingDeps) > 0 {
			warningMsg := fmt.Sprintf("Missing system tools detected: %v.\nSome features may not work. Install them now?", missingDeps)

			passEntry := NewAutoKeyboardPasswordEntry()
			passEntry.SetPlaceHolder("Admin (sudo) password")

			content := container.NewVBox(
				widget.NewLabel(warningMsg),
				passEntry,
			)

			dialog.ShowCustomConfirm("Missing Dependencies", "Install", "Ignore", content, func(confirm bool) {
				if confirm && passEntry.Text != "" {
					progress := dialog.NewCustomWithoutButtons("Installing...", container.NewPadded(widget.NewProgressBarInfinite()), w)
					progress.Show()

					go func() {
						err := system.InstallDependencies(passEntry.Text, missingDeps)

						progress.Hide()

						if err != nil {
							dialog.ShowError(fmt.Errorf("Installation failed.\nDid you enter the correct password?\nError: %v", err), w)
						} else {
							dialog.ShowInformation("Success", "All missing dependencies installed successfully!", w)
						}
					}()
				}
			}, w)
		}

		netBtn := widget.NewButtonWithIcon("Network & Wi-Fi", theme.ComputerIcon(), func() { showDetail("Network Settings", buildNetworkView()) })
		mediaBtn := widget.NewButtonWithIcon("Display & Audio", theme.ColorPaletteIcon(), func() { showDetail("Display & Audio", buildMediaView()) })
		powerBtn := widget.NewButtonWithIcon("Power", theme.InfoIcon(), func() { showDetail("Power Management", buildPowerView()) })
		sysBtn := widget.NewButtonWithIcon("System", theme.SettingsIcon(), func() { showDetail("System Controls", buildSystemView()) })
		secBtn := widget.NewButtonWithIcon("Security & PIN", theme.VisibilityOffIcon(), func() { showDetail("Security Settings", buildSecurityView()) })

		updBtn := widget.NewButtonWithIcon("Update App", theme.DownloadIcon(), func() { showDetail("Application Update", buildUpdateView()) })

		netBtn.Importance = widget.HighImportance
		mediaBtn.Importance = widget.HighImportance
		powerBtn.Importance = widget.HighImportance
		sysBtn.Importance = widget.HighImportance
		secBtn.Importance = widget.HighImportance
		updBtn.Importance = widget.HighImportance

		grid := container.NewGridWithColumns(3, netBtn, mediaBtn, powerBtn, sysBtn, secBtn, updBtn)

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
