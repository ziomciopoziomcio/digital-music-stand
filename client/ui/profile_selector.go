package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/profiles"
)

func ShowProfileSelector(w fyne.Window, a fyne.App, pm *profiles.Manager, onProfileSelected func(profileID string)) {
	var showProfiles func()
	var showPinScreen func(p profiles.Profile)

	showProfiles = func() {
		profs, err := pm.GetProfiles()
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		title := widget.NewLabelWithStyle("Select Profile", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		grid := container.NewGridWrap(fyne.NewSize(200, 150))

		for _, p := range profs {
			profile := p

			bg := canvas.NewRectangle(ParseColor(profile.Color))
			bg.CornerRadius = 12

			nameLabel := canvas.NewText(profile.Name, color.White)
			nameLabel.Alignment = fyne.TextAlignCenter
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			nameLabel.TextSize = 20

			btn := widget.NewButton("", func() {
				if profile.PinHash != "" {
					showPinScreen(profile)
				} else {
					onProfileSelected(profile.ID)
				}
			})
			btn.Importance = widget.LowImportance

			tile := container.NewMax(bg, container.NewCenter(nameLabel), btn)
			grid.Add(tile)
		}

		addBtn := widget.NewButtonWithIcon("New Profile", theme.ContentAddIcon(), func() {
			showCreateProfileDialog(w, a, pm, showProfiles)
		})
		addBtn.Importance = widget.HighImportance
		grid.Add(addBtn)

		content := container.NewBorder(
			container.NewPadded(title),
			nil, nil, nil,
			container.NewCenter(grid),
		)
		w.SetContent(content)
	}

	showPinScreen = func(p profiles.Profile) {
		enteredPin := ""
		pinDisplay := widget.NewLabelWithStyle("Enter PIN", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		updateDisplay := func() {
			if enteredPin == "" {
				pinDisplay.SetText("Enter PIN")
			} else {
				pinDisplay.SetText(strings.Repeat("*", len(enteredPin)))
			}
		}

		backBtn := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() {
			showProfiles()
		})

		keys := container.NewGridWithColumns(3)
		for i := 1; i <= 9; i++ {
			digit := i
			keys.Add(widget.NewButton(string(rune('0'+digit)), func() {
				enteredPin += string(rune('0' + digit))
				updateDisplay()
			}))
		}
		keys.Add(widget.NewButton("C", func() {
			enteredPin = ""
			updateDisplay()
		}))
		keys.Add(widget.NewButton("0", func() {
			enteredPin += "0"
			updateDisplay()
		}))
		keys.Add(widget.NewButton("OK", func() {
			if pm.VerifyPin(p.ID, enteredPin) {
				onProfileSelected(p.ID)
			} else {
				enteredPin = ""
				pinDisplay.SetText("Invalid PIN!")
			}
		}))

		content := container.NewBorder(
			container.NewHBox(backBtn),
			nil, nil, nil,
			container.NewCenter(container.NewVBox(
				widget.NewLabelWithStyle("Login: "+p.Name, fyne.TextAlignCenter, fyne.TextStyle{}),
				pinDisplay,
				keys,
			)),
		)
		w.SetContent(content)
	}

	showProfiles()
}

func showCreateProfileDialog(w fyne.Window, a fyne.App, pm *profiles.Manager, refresh func()) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Name")

	pinEntry := widget.NewPasswordEntry()
	pinEntry.SetPlaceHolder("Optional PIN")

	colorSelect := widget.NewSelect([]string{"blue", "red", "green", "purple", "orange", "yellow"}, nil)
	colorSelect.SetSelected("blue")

	items := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("PIN", pinEntry),
		widget.NewFormItem("Tile Color", colorSelect),
	}

	dialog.ShowForm("Create Profile", "Add", "Cancel", items, func(confirm bool) {
		if confirm && nameEntry.Text != "" {
			newProfile, err := pm.CreateProfile(nameEntry.Text, pinEntry.Text, colorSelect.Selected)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			a.Preferences().SetString(newProfile.ID+"_theme_color", colorSelect.Selected)
			refresh()
		}
	}, w)
}
