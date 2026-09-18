package ui

import (
	"fmt"
	"strings"

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

type MemberInfo struct {
	UserID  uint32
	Email   string
	Name    string
	Surname string
	Role    string
}

func BuildProfile(
	w fyne.Window,
	a fyne.App,
	onBack func(),
	onCloudLogout func(),
	fetchBands func() ([]BandInfo, error),
	createBand func(name string) error,
	inviteMember func(bandID uint32, email string) error,
	changePassword func(oldPassword, newPassword string) error,
	listMembers func(bandID uint32) ([]MemberInfo, error),
	removeMember func(bandID uint32, userID uint32, email string) error,
	hasCredentials bool,
	serverAddr string,
) *fyne.Container {
	topBar := container.NewHBox(
		widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), onBack),
		widget.NewLabelWithStyle("User Profile", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	if !hasCredentials || serverAddr == "" {
		notLoggedInLabel := widget.NewLabel("You are currently working in Offline Mode.")
		content := container.NewVBox(
			topBar,
			widget.NewSeparator(),
			notLoggedInLabel,
		)
		return container.NewPadded(content)
	}

	statusLabel := widget.NewLabel(fmt.Sprintf("Connected to: %s", serverAddr))

	formatErr := func(err error) error {
		errMsg := FormatAppError(err).Error()
		if len(errMsg) > 50 || strings.Contains(errMsg, "rpc error") || strings.Contains(errMsg, "connection error") {
			return fmt.Errorf("Server is unavailable. You are currently offline.")
		}
		return fmt.Errorf(errMsg)
	}

	changePassBtn := widget.NewButtonWithIcon("Change Password", theme.SettingsIcon(), func() {
		oldPassEntry := NewAutoKeyboardPasswordEntry()
		oldPassEntry.SetPlaceHolder("Current Password")

		newPassEntry := NewAutoKeyboardPasswordEntry()
		newPassEntry.SetPlaceHolder("New Password")

		form := container.NewVBox(
			widget.NewLabel("Current Password:"),
			oldPassEntry,
			widget.NewLabel("New Password:"),
			newPassEntry,
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
				dialog.ShowError(formatErr(err), w)
				return
			}
			dialog.ShowInformation("Success", "Password updated successfully.", w)
		}, w)
	})

	logoutBtn := widget.NewButtonWithIcon("Logout from Cloud", theme.LogoutIcon(), onCloudLogout)
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
			errorLabel := widget.NewLabelWithStyle(fmt.Sprintf("Could not load bands:\n%s", formatErr(err).Error()), fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
			bandsContainer.Add(errorLabel)
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
				bandLabel := widget.NewLabel(fmt.Sprintf("  %s (%s)", band.Name, roleStr))
				row := container.NewHBox(bandLabel)

				var showMembersDialog func()
				showMembersDialog = func() {
					members, err := listMembers(band.ID)
					if err != nil {
						dialog.ShowError(formatErr(err), w)
						return
					}

					membersBox := container.NewVBox()
					for _, m := range members {
						member := m
						var nameStr string
						if member.Role == "pending" {
							nameStr = fmt.Sprintf("  %s (Pending Invite)", member.Email)
						} else if member.Name == "" && member.Surname == "" {
							nameStr = fmt.Sprintf("%s (%s)", member.Email, member.Role)
						} else {
							nameStr = fmt.Sprintf("%s %s (%s)", member.Name, member.Surname, member.Role)
						}
						memberRow := container.NewHBox(widget.NewLabel(nameStr))

						if band.IsManager && member.Role != "manager" {
							deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
								msg := fmt.Sprintf("Are you sure you want to remove %s from the band?", member.Email)
								if member.Role == "pending" {
									msg = fmt.Sprintf("Are you sure you want to cancel the invitation for %s?", member.Email)
								}
								dialog.ShowConfirm("Confirm Action", msg, func(confirm bool) {
									if confirm {
										err := removeMember(band.ID, member.UserID, member.Email)
										if err != nil {
											dialog.ShowError(formatErr(err), w)
										} else {
											dialog.ShowInformation("Success", "Action completed.", w)
											showMembersDialog()
										}
									}
								}, w)
							})
							deleteBtn.Importance = widget.DangerImportance
							memberRow.Add(deleteBtn)
						}

						membersBox.Add(memberRow)
					}

					scroll := container.NewVScroll(membersBox)
					d := dialog.NewCustom(fmt.Sprintf("Members of %s", band.Name), "Close", scroll, w)

					winSize := w.Canvas().Size()
					targetWidth := float32(400)
					targetHeight := float32(350)
					if winSize.Width < targetWidth {
						targetWidth = winSize.Width * 0.95
					}
					if winSize.Height < targetHeight {
						targetHeight = winSize.Height * 0.95
					}

					d.Resize(fyne.NewSize(targetWidth, targetHeight))
					d.Show()
				}

				membersBtn := widget.NewButtonWithIcon("Members", theme.VisibilityIcon(), showMembersDialog)
				row.Add(membersBtn)

				if band.IsManager {
					inviteBtn := widget.NewButtonWithIcon("Invite", theme.ContentAddIcon(), func() {
						emailEntry := NewAutoKeyboardEntry()
						emailEntry.SetPlaceHolder("musician@example.com")
						dialog.ShowCustomConfirm("Invite to Band", "Send Invite", "Cancel", emailEntry, func(confirm bool) {
							if confirm && emailEntry.Text != "" {
								err := inviteMember(band.ID, emailEntry.Text)
								if err != nil {
									dialog.ShowError(formatErr(err), w)
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
			bandsContainer.Refresh()
		}
	}

	createBandBtn := widget.NewButtonWithIcon("Create New Band", theme.FolderNewIcon(), func() {
		nameEntry := NewAutoKeyboardEntry()
		nameEntry.SetPlaceHolder("Band Name")
		dialog.ShowCustomConfirm("Create Band", "Create", "Cancel", nameEntry, func(confirm bool) {
			if confirm && nameEntry.Text != "" {
				err := createBand(nameEntry.Text)
				if err != nil {
					dialog.ShowError(formatErr(err), w)
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
