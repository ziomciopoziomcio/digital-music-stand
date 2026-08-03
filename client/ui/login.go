package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildLoginScreen(app fyne.App, attemptLogin func(server, email, password string) error, goBack func()) *fyne.Container {
	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("Server Address (e.g. localhost:50051)")
	serverEntry.SetText(app.Preferences().StringWithFallback("server_ip", "localhost:50051"))

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email Address")
	emailEntry.SetText(app.Preferences().StringWithFallback("last_email", ""))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	errorLabel := widget.NewLabel("")
	errorLabel.Hide()

	loginBtn := widget.NewButtonWithIcon("Connect & Login", theme.ConfirmIcon(), func() {
		errorLabel.Hide()
		app.Preferences().SetString("server_ip", serverEntry.Text)
		app.Preferences().SetString("last_email", emailEntry.Text)

		err := attemptLogin(serverEntry.Text, emailEntry.Text, passwordEntry.Text)
		if err != nil {
			errorLabel.SetText(err.Error())
			errorLabel.Show()
		}
	})
	loginBtn.Importance = widget.HighImportance

	backBtn := widget.NewButtonWithIcon("Back to Dashboard", theme.NavigateBackIcon(), goBack)
	backBtn.Importance = widget.WarningImportance

	title := widget.NewLabelWithStyle("Digital Music Stand - Login", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	formContainer := container.NewVBox(
		title,
		widget.NewLabel(""),
		widget.NewLabel("Server IP:"),
		serverEntry,
		widget.NewLabel("Email:"),
		emailEntry,
		widget.NewLabel("Password:"),
		passwordEntry,
		widget.NewLabel(""),
		loginBtn,
		widget.NewLabel(""),
		backBtn,
		errorLabel,
	)

	paddedForm := container.NewPadded(formContainer)
	centeredForm := container.NewCenter(paddedForm)

	return container.NewBorder(nil, nil, layout.NewSpacer(), layout.NewSpacer(), centeredForm)
}
