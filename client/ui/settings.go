package ui

import (
	"fmt"
	"io"
	"log"
	"os/exec"

	"fyne.io/fyne/v2"
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

		installKbBtn := widget.NewButtonWithIcon("Install On-Screen Keyboard", theme.DownloadIcon(), func() {

			passEntry := NewAutoKeyboardPasswordEntry()
			passEntry.SetPlaceHolder("Admin (sudo) password")

			dialog.ShowCustomConfirm("Sudo Password Required", "Install", "Cancel", passEntry, func(confirm bool) {
				if confirm && passEntry.Text != "" {
					progress := dialog.NewCustomWithoutButtons("Installing...", container.NewPadded(widget.NewProgressBarInfinite()), w)
					progress.Show()

					go func() {
						pwd := passEntry.Text

						cmd1 := exec.Command("sudo", "-S", "apt-get", "update")
						stdin1, _ := cmd1.StdinPipe()
						go func() {
							defer stdin1.Close()
							io.WriteString(stdin1, pwd+"\n")
						}()
						_ = cmd1.Run()

						cmd2 := exec.Command("sudo", "-S", "apt-get", "install", "-y", "matchbox-keyboard")
						stdin2, _ := cmd2.StdinPipe()
						go func() {
							defer stdin2.Close()
							io.WriteString(stdin2, pwd+"\n")
						}()
						err := cmd2.Run()

						progress.Hide()

						if err != nil {
							dialog.ShowError(fmt.Errorf("Installation failed.\nDid you enter the correct password?\nError: %v", err), w)
						} else {
							dialog.ShowInformation("Success", "Keyboard installed successfully!", w)
						}
					}()
				}
			}, w)
		})

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
			widget.NewSeparator(),
			installKbBtn,
			layout.NewSpacer(),
			rebootBtn,
			shutdownBtn,
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
