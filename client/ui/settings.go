package ui

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/plugins"
	"github.com/ziomciopoziomcio/digital-music-stand/client/profiles"
	"github.com/ziomciopoziomcio/digital-music-stand/client/system"
	"github.com/ziomciopoziomcio/digital-music-stand/client/updater"
)

func BuildSettings(w fyne.Window, app fyne.App, currentVersion string, onClose func(), netMgr system.NetworkManager, pwrMgr system.PowerManager, medMgr system.MediaManager, devMgr system.DeviceManager, pm *profiles.Manager, profileID string) *fyne.Container {
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
			if pm.CheckIfHasPin(profileID) {
				statusLabel.SetText("Status: PIN Protection Active")
			} else {
				statusLabel.SetText("Status: PIN Protection Disabled")
			}
		}
		updateStatus()

		savePinBtn := widget.NewButtonWithIcon("Save PIN", theme.DocumentSaveIcon(), func() {
			if pinEntry.Text != "" {
				if err := pm.UpdatePin(profileID, pinEntry.Text); err != nil {
					dialog.ShowError(err, w)
					return
				}
				dialog.ShowInformation("Security", "PIN updated successfully.", w)
				pinEntry.SetText("")
				updateStatus()
			}
		})
		savePinBtn.Importance = widget.HighImportance

		clearPinBtn := widget.NewButtonWithIcon("Remove PIN", theme.DeleteIcon(), func() {
			if err := pm.UpdatePin(profileID, ""); err != nil {
				dialog.ShowError(err, w)
				return
			}
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
		ethConnected, _ := netMgr.GetEthernetStatus()
		ethStatusText := "Ethernet: Disconnected"
		if ethConnected {
			ethStatusText = "Ethernet: Connected (Ready)"
		}
		ethLabel := widget.NewLabelWithStyle(ethStatusText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

		statusLabel := widget.NewLabel(fmt.Sprintf("Wi-Fi Status: %s", netMgr.GetNetworkStatus()))
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
							connectAction := func(password string) {
								progress := dialog.NewCustomWithoutButtons("Connecting to "+net.SSID+"...", container.NewPadded(widget.NewProgressBarInfinite()), w)
								progress.Show()

								go func() {
									err := netMgr.ConnectWiFi(net.SSID, password)
									progress.Hide()
									if err != nil {
										dialog.ShowError(fmt.Errorf("Failed to connect: %v", err), w)
									}
									statusLabel.SetText(fmt.Sprintf("Status: %s", netMgr.GetNetworkStatus()))
								}()
							}

							if net.Secure {
								passEntry := NewAutoKeyboardPasswordEntry()
								passEntry.SetPlaceHolder("Wi-Fi Password")
								dialog.ShowCustomConfirm("Connect to "+net.SSID, "Connect", "Cancel", passEntry, func(confirm bool) {
									if confirm {
										connectAction(passEntry.Text)
									}
								}, w)
							} else {
								connectAction("")
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

		addHiddenBtn := widget.NewButtonWithIcon("Add Hidden Wi-Fi", theme.ContentAddIcon(), func() {
			ssidEntry := widget.NewEntry()
			ssidEntry.SetPlaceHolder("Network Name (SSID)")

			passEntry := NewAutoKeyboardPasswordEntry()
			passEntry.SetPlaceHolder("Password")

			form := container.NewVBox(
				widget.NewLabel("Connect to a non-broadcasted network:"),
				ssidEntry,
				passEntry,
			)

			dialog.ShowCustomConfirm("Add Hidden Network", "Connect", "Cancel", form, func(confirm bool) {
				if confirm && ssidEntry.Text != "" {
					err := netMgr.ConnectHiddenWiFi(ssidEntry.Text, passEntry.Text)
					if err != nil {
						dialog.ShowError(fmt.Errorf("Failed to connect: %v", err), w)
					}
					refreshNetworks()
				}
			}, w)
		})
		addHiddenBtn.Importance = widget.WarningImportance

		disconnectBtn := widget.NewButtonWithIcon("Disconnect", theme.CancelIcon(), func() {
			_ = netMgr.Disconnect()
			statusLabel.SetText(fmt.Sprintf("Status: %s", netMgr.GetNetworkStatus()))
		})
		disconnectBtn.Importance = widget.DangerImportance

		dhcpCheckbox := widget.NewCheck("Enable DHCP", nil)
		dhcpCheckbox.SetChecked(true)

		ipEntry := widget.NewEntry()
		ipEntry.SetPlaceHolder("192.168.1.100")
		maskEntry := widget.NewEntry()
		maskEntry.SetPlaceHolder("255.255.255.0")
		gatewayEntry := widget.NewEntry()
		gatewayEntry.SetPlaceHolder("192.168.1.1")
		dnsEntry := widget.NewEntry()
		dnsEntry.SetPlaceHolder("8.8.8.8")

		ipEntry.Disable()
		maskEntry.Disable()
		gatewayEntry.Disable()
		dnsEntry.Disable()

		dhcpCheckbox.OnChanged = func(checked bool) {
			if checked {
				ipEntry.Disable()
				maskEntry.Disable()
				gatewayEntry.Disable()
				dnsEntry.Disable()
			} else {
				ipEntry.Enable()
				maskEntry.Enable()
				gatewayEntry.Enable()
				dnsEntry.Enable()
			}
		}

		applyIpBtn := widget.NewButtonWithIcon("Apply IP Config", theme.DocumentSaveIcon(), func() {
			if dhcpCheckbox.Checked {
				err := netMgr.SetDHCP("wlan0", true)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to set DHCP: %v", err), w)
				} else {
					dialog.ShowInformation("Success", "DHCP enabled successfully.", w)
				}
			} else {
				err := netMgr.SetStaticIP("wlan0", ipEntry.Text, maskEntry.Text, gatewayEntry.Text, dnsEntry.Text)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to set Static IP: %v", err), w)
				} else {
					dialog.ShowInformation("Success", "Static IP applied successfully.", w)
				}
			}
		})
		applyIpBtn.Importance = widget.HighImportance

		advancedIpForm := container.NewVBox(
			widget.NewLabelWithStyle("Advanced IP Configuration (wlan0)", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			dhcpCheckbox,
			widget.NewLabel("IP Address:"), ipEntry,
			widget.NewLabel("Subnet Mask:"), maskEntry,
			widget.NewLabel("Gateway:"), gatewayEntry,
			widget.NewLabel("DNS:"), dnsEntry,
			widget.NewLabel(""),
			applyIpBtn,
		)

		topBar := container.NewVBox(
			ethLabel,
			container.NewHBox(statusLabel, layout.NewSpacer(), addHiddenBtn, disconnectBtn, refreshBtn),
			widget.NewSeparator(),
		)

		refreshNetworks()

		return container.NewBorder(topBar, nil, nil, nil, container.NewVSplit(container.NewVScroll(listContainer), container.NewVScroll(container.NewPadded(advancedIpForm))))
	}

	buildMediaView := func() fyne.CanvasObject {
		vol, _ := medMgr.GetVolume()
		bright, _ := medMgr.GetBrightness()

		volLabel := widget.NewLabel(fmt.Sprintf("Volume: %d%%", vol))
		volSlider := widget.NewSlider(0, 100)
		volSlider.SetValue(float64(vol))
		volSlider.OnChanged = ThrottledSliderHandler(100*time.Millisecond, func(val int) {
			_ = medMgr.SetVolume(val)
			volLabel.SetText(fmt.Sprintf("Volume: %d%%", val))
		})

		brightLabel := widget.NewLabel(fmt.Sprintf("Brightness: %d%%", bright))
		brightSlider := widget.NewSlider(0, 100)
		brightSlider.SetValue(float64(bright))
		brightSlider.OnChanged = ThrottledSliderHandler(200*time.Millisecond, func(val int) {
			_ = medMgr.SetBrightness(val)
			brightLabel.SetText(fmt.Sprintf("Brightness: %d%%", val))
		})

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

		camEntry := widget.NewEntry()
		camEntry.SetText(fmt.Sprintf("%d", app.Preferences().IntWithFallback("eyetrack_camera", 0)))
		camBtn := NewTouchButton("Save", func() {
			var cID int
			var err error
			if cID, err = strconv.Atoi(camEntry.Text); err == nil {
				app.Preferences().SetInt("eyetrack_camera", cID)
				dialog.ShowInformation("Saved", "Eye tracking camera has been saved", w)
			}
		})

		systemElements := []fyne.CanvasObject{
			awakeCheck,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Eye tracking", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			container.NewHBox(widget.NewLabel("Cam ID:"), camEntry, camBtn),
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

	buildAppearanceView := func() fyne.CanvasObject {
		prefTheme := profileID + "_theme_color"
		currentAccent := app.Preferences().StringWithFallback(prefTheme, "blue")

		colorSelect := widget.NewSelect([]string{"blue", "red", "green", "purple", "orange", "yellow"}, func(selected string) {
			app.Preferences().SetString(prefTheme, selected)
			ApplyAppTheme(app, selected)
		})
		colorSelect.SetSelected(currentAccent)

		return container.NewVBox(
			widget.NewLabelWithStyle("App Accent Color", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Choose the primary color for buttons, selections, and highlights."),
			colorSelect,
		)
	}

	buildMixerView := func() fyne.CanvasObject {
		mixerOptions := append([]string{"None"}, plugins.GetAvailableMixers()...)

		prefMixerPlugin := profileID + "_mixer_plugin"
		prefMixerIP := profileID + "_mixer_ip"

		savedMixer := app.Preferences().StringWithFallback(prefMixerPlugin, "None")
		savedIP := app.Preferences().StringWithFallback(prefMixerIP, "192.168.1.100")

		ipEntry := widget.NewEntry()
		ipEntry.SetText(savedIP)

		statusLabel := widget.NewLabelWithStyle("Status: Disconnected", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

		var mixerSelect *widget.Select

		updateConnection := func(pluginName, ip string) {
			if plugins.GetActiveMixer() != nil {
				plugins.GetActiveMixer().Disconnect()
				plugins.SetActiveMixer(nil)
			}
			if pluginName == "None" || pluginName == "" {
				statusLabel.SetText("Status: Disabled")
				return
			}
			if m, err := plugins.GetMixer(pluginName); err == nil {
				err := m.Connect(ip)
				if err == nil {
					plugins.SetActiveMixer(m)
					statusLabel.SetText(fmt.Sprintf("Status: Connected to %s", pluginName))
				} else {
					statusLabel.SetText(fmt.Sprintf("Error: %v", err))
				}
			}
		}

		mixerSelect = widget.NewSelect(mixerOptions, func(selected string) {
			app.Preferences().SetString(prefMixerPlugin, selected)
		})
		mixerSelect.SetSelected(savedMixer)

		ipEntry.OnChanged = func(s string) {
			app.Preferences().SetString(prefMixerIP, s)
		}

		connectBtn := widget.NewButtonWithIcon("Apply & Connect", theme.MediaPlayIcon(), func() {
			updateConnection(mixerSelect.Selected, ipEntry.Text)
		})
		connectBtn.Importance = widget.HighImportance

		if savedMixer != "None" && savedMixer != "" && plugins.GetActiveMixer() == nil {
			updateConnection(savedMixer, savedIP)
		} else if plugins.GetActiveMixer() != nil {
			statusLabel.SetText(fmt.Sprintf("Status: Connected to %s", plugins.GetActiveMixer().Name()))
		}

		return container.NewVBox(
			widget.NewLabelWithStyle("Stage Mixer Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Select a plugin and enter the mixer's IP address on the local network."),
			widget.NewSeparator(),
			widget.NewLabel("Mixer Plugin:"),
			mixerSelect,
			widget.NewLabel("IP Address:"),
			ipEntry,
			widget.NewLabel(""),
			connectBtn,
			statusLabel,
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
						pwd := passEntry.Text
						var packagesToInstall []string

						for _, dep := range missingDeps {
							switch dep {
							case "nmcli":
								packagesToInstall = append(packagesToInstall, "network-manager")
							case "amixer":
								packagesToInstall = append(packagesToInstall, "alsa-utils")
							case "xset":
								packagesToInstall = append(packagesToInstall, "x11-xserver-utils")
							default:
								packagesToInstall = append(packagesToInstall, dep)
							}
						}

						cmd1 := exec.Command("sudo", "-S", "apt-get", "update")
						if stdin1, err := cmd1.StdinPipe(); err == nil {
							go func() {
								defer stdin1.Close()
								io.WriteString(stdin1, pwd+"\n")
							}()
						}
						_ = cmd1.Run()

						args := append([]string{"-S", "apt-get", "install", "-y"}, packagesToInstall...)
						cmd2 := exec.Command("sudo", args...)
						if stdin2, err := cmd2.StdinPipe(); err == nil {
							go func() {
								defer stdin2.Close()
								io.WriteString(stdin2, pwd+"\n")
							}()
						}
						err := cmd2.Run()

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
		apprBtn := widget.NewButtonWithIcon("Appearance", theme.ColorPaletteIcon(), func() { showDetail("Appearance Settings", buildAppearanceView()) })
		updBtn := widget.NewButtonWithIcon("Update App", theme.DownloadIcon(), func() { showDetail("Application Update", buildUpdateView()) })
		mixerBtn := widget.NewButtonWithIcon("Stage Mixer", theme.VolumeUpIcon(), func() { showDetail("Mixer Configuration", buildMixerView()) })

		netBtn.Importance = widget.HighImportance
		mediaBtn.Importance = widget.HighImportance
		powerBtn.Importance = widget.HighImportance
		sysBtn.Importance = widget.HighImportance
		secBtn.Importance = widget.HighImportance
		updBtn.Importance = widget.HighImportance
		apprBtn.Importance = widget.HighImportance
		mixerBtn.Importance = widget.HighImportance

		grid := container.NewGridWithColumns(3, netBtn, mediaBtn, powerBtn, sysBtn, secBtn, apprBtn, updBtn, mixerBtn)

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
