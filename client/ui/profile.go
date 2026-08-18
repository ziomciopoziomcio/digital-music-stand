package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
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
	app fyne.App,
	goBack func(),
	onLogout func(),
	fetchBands func() ([]BandInfo, error),
	onCreateBand func(name string) error,
	onInviteMember func(bandID uint32, email string) error,
) *fyne.Container {
	contentWrapper := container.NewMax()

	var renderProfile func()

	renderProfile = func() {
		backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		header := container.NewBorder(nil, nil, backBtn, nil,
			widget.NewLabelWithStyle("Profile & Bands", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		token := app.Preferences().String("jwt_token")
		server := app.Preferences().String("server_addr")

		serverLabel := widget.NewLabel(fmt.Sprintf("Server: %s", server))
		statusLabel := widget.NewLabel(fmt.Sprintf("Status: %s", func() string {
			if token != "" {
				return "Logged In"
			}
			return "Offline / Guest"
		}()))

		userInfoCard := widget.NewCard("Account Info", "", container.NewVBox(serverLabel, statusLabel))

		logoutBtn := widget.NewButtonWithIcon("Log Out", theme.LogoutIcon(), func() {
			dialog.ShowConfirm("Log Out", "Are you sure you want to log out?", func(confirmed bool) {
				if confirmed {
					onLogout()
				}
			}, w)
		})
		logoutBtn.Importance = widget.DangerImportance

		userSection := container.NewVBox(
			userInfoCard,
			logoutBtn,
			widget.NewSeparator(),
		)

		bandsListContainer := container.NewVBox()

		createBandBtn := widget.NewButtonWithIcon("Create Band", theme.ContentAddIcon(), func() {
			entry := widget.NewEntry()
			entry.SetPlaceHolder("Band Name")

			var d dialog.Dialog
			formContent := container.NewVBox(
				widget.NewLabel("Enter band name:"),
				entry,
				container.NewHBox(
					layout.NewSpacer(),
					widget.NewButton("Create", func() {
						if entry.Text != "" {
							if err := onCreateBand(entry.Text); err != nil {
								dialog.ShowError(err, w)
							} else {
								dialog.ShowInformation("Success", "Band created successfully!", w)
								renderProfile()
							}
							d.Hide()
						}
					}),
					widget.NewButton("Cancel", func() { d.Hide() }),
				),
			)

			d = dialog.NewCustomWithoutButtons("Create Band", formContent, w)
			d.Show()
			w.Canvas().Focus(entry)
		})
		createBandBtn.Importance = widget.HighImportance

		bandsHeader := container.NewBorder(nil, nil,
			widget.NewLabelWithStyle("My Bands", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			createBandBtn,
		)

		if token == "" {
			bandsListContainer.Objects = []fyne.CanvasObject{
				widget.NewLabelWithStyle("Log in to view and manage your bands.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
			}
		} else {
			bands, err := fetchBands()
			if err != nil {
				bandsListContainer.Objects = []fyne.CanvasObject{
					widget.NewLabel(fmt.Sprintf("Error fetching bands: %v", err)),
				}
			} else if len(bands) == 0 {
				bandsListContainer.Objects = []fyne.CanvasObject{
					widget.NewLabelWithStyle("You do not belong to any bands yet.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
				}
			} else {
				grid := container.NewVBox()
				for _, b := range bands {
					band := b
					roleText := "Member"
					if band.IsManager {
						roleText = "Manager"
					}

					bandTitle := widget.NewLabelWithStyle(fmt.Sprintf("%s (%s)", band.Name, roleText), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

					var inviteBtn *widget.Button
					if band.IsManager {
						inviteBtn = widget.NewButtonWithIcon("Invite Member", theme.MailSendIcon(), func() {
							emailEntry := widget.NewEntry()
							emailEntry.SetPlaceHolder("user@example.com")

							var d dialog.Dialog
							inviteForm := container.NewVBox(
								widget.NewLabel(fmt.Sprintf("Invite member to '%s':", band.Name)),
								emailEntry,
								container.NewHBox(
									layout.NewSpacer(),
									widget.NewButton("Send Invite", func() {
										if emailEntry.Text != "" {
											if err := onInviteMember(band.ID, emailEntry.Text); err != nil {
												dialog.ShowError(err, w)
											} else {
												dialog.ShowInformation("Success", "Invitation sent!", w)
											}
											d.Hide()
										}
									}),
									widget.NewButton("Cancel", func() { d.Hide() }),
								),
							)

							d = dialog.NewCustomWithoutButtons("Invite Member", inviteForm, w)
							d.Show()
							w.Canvas().Focus(emailEntry)
						})
						inviteBtn.Importance = widget.HighImportance
					}

					var row *fyne.Container
					if inviteBtn != nil {
						row = container.NewBorder(nil, nil, bandTitle, inviteBtn)
					} else {
						row = container.NewBorder(nil, nil, bandTitle, nil)
					}

					grid.Add(widget.NewCard("", "", row))
				}
				bandsListContainer.Objects = []fyne.CanvasObject{grid}
			}
		}

		bandsSection := container.NewBorder(
			bandsHeader,
			nil, nil, nil,
			container.NewVScroll(bandsListContainer),
		)

		mainLayout := container.NewBorder(
			userSection,
			nil, nil, nil,
			bandsSection,
		)

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(mainLayout))
		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	renderProfile()
	return contentWrapper
}
