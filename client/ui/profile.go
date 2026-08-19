package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type BandInfo struct {
	ID        uint32
	Name      string
	IsManager bool
}

func BuildProfile(
	w fyne.Window,
	a fyne.App,
	onBack func(),
	onLogout func(),
	fetchBands func() ([]BandInfo, error),
	createBand func(name string) error,
	inviteMember func(bandID uint32, email string) error,
	changePassword func(oldPassword, newPassword string) error,
) *fyne.Container {
	token := a.Preferences().String("jwt_token")
	server := a.Preferences().String("server_addr")

	topBar := container.NewHBox(
		widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), onBack),
		widget.NewLabelWithStyle("User Profile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	if token == "" || server == "" {
		notLoggedInLabel := widget.NewLabel("You are currently working in Offline Mode.")
		content := container.NewVBox(
			topBar,
			widget.NewSeparator(),
			notLoggedInLabel,
		)
		return container.NewPadded(content)
	}

	statusLabel := widget.NewLabel(fmt.Sprintf("Connected to: %s", server))

	changePassBtn := widget.NewButtonWithIcon("Change Password", theme.SettingsIcon(), func() {
		oldPassEntry := widget.NewPasswordEntry()
		oldPassEntry.SetPlaceHolder("Current Password")
		newPassEntry := widget.NewPasswordEntry()
		newPassEntry.SetPlaceHolder("New Password")

		form := container.NewVBox(
			widget.NewLabel("Current Password:"), oldPassEntry,
			widget.NewLabel("New Password:"), newPassEntry,
		)

		dialog.ShowCustomConfirm("Change Password", "Save", "Cancel", form, func(confirm bool) {
			if !confirm {
				return
			}
			if oldPassEntry.Text == "" || newPassEntry.Text == "" {
				dialog.ShowInformation("Error", "Please fill in both password fields.", w)
				return
			}

			err := changePassword(oldPassEntry.Text, newPassEntry.Text)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Success", "Password updated successfully.", w)
		}, w)
	})

	logoutBtn := widget.NewButtonWithIcon("Logout", theme.LogoutIcon(), onLogout)
	logoutBtn.Importance = widget.DangerImportance

	accountSection := container.NewVBox(
		widget.NewLabelWithStyle("Account Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		statusLabel,
		container.NewHBox(changePassBtn, logoutBtn),
		widget.NewSeparator(),
	)

	bandsContainer := container.NewVBox()
	refreshBandsList := func() {
		bandsContainer.Objects = nil
		bands, err := fetchBands()
		if err != nil {
			bandsContainer.Add(widget.NewLabel(fmt.Sprintf("Failed to load bands: %v", err)))
			bandsContainer.Refresh()
			return
		}

		if len(bands) == 0 {
			bandsContainer.Add(widget.NewLabel("You are not a member of any band yet."))
		} else {
			for _, b := range bands {
				band := b
				roleStr := "Member"
				if band.IsManager {
					roleStr = "Manager"
				}

				bandLabel := widget.NewLabel(fmt.Sprintf("• %s (%s)", band.Name, roleStr))
				row := container.NewHBox(bandLabel)

				if band.IsManager {
					inviteBtn := widget.NewButtonWithIcon("Invite Member", theme.ContentAddIcon(), func() {
						emailEntry := widget.NewEntry()
						emailEntry.SetPlaceHolder("musician@example.com")

						dialog.ShowCustomConfirm("Invite to Band", "Send Invite", "Cancel", emailEntry, func(confirm bool) {
							if confirm && emailEntry.Text != "" {
								err := inviteMember(band.ID, emailEntry.Text)
								if err != nil {
									dialog.ShowError(err, w)
								} else {
									dialog.ShowInformation("Success", "Invitation sent successfully!", w)
								}
							}
						}, w)
					})
					row.Add(inviteBtn)
				}
				bandsContainer.Add(row)
			}
		}
		bandsContainer.Refresh()
	}

	createBandBtn := widget.NewButtonWithIcon("Create New Band", theme.FolderNewIcon(), func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Band Name")

		dialog.ShowCustomConfirm("Create Band", "Create", "Cancel", nameEntry, func(confirm bool) {
			if confirm && nameEntry.Text != "" {
				err := createBand(nameEntry.Text)
				if err != nil {
					dialog.ShowError(err, w)
				} else {
					dialog.ShowInformation("Success", "Band created successfully!", w)
					refreshBandsList()
				}
			}
		}, w)
	})

	bandsSection := container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle("My Bands", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			createBandBtn,
		),
		bandsContainer,
	)

	refreshBandsList()

	mainLayout := container.NewVBox(
		topBar,
		widget.NewSeparator(),
		accountSection,
		bandsSection,
	)

	return container.NewPadded(container.NewVScroll(mainLayout))
}
