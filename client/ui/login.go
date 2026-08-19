package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildLoginScreen(
	w fyne.Window,
	a fyne.App,
	loginCallback func(server, email, password string) error,
	registerCallback func(server, email, password, name, surname string) (string, error),
	onCancel func(),
) *fyne.Container {
	wrapper := container.NewMax()

	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("localhost:50051")
	savedServer := a.Preferences().String("server_addr")
	if savedServer != "" {
		serverEntry.SetText(savedServer)
	} else {
		serverEntry.SetText("localhost:50051")
	}

	loginEmailEntry := widget.NewEntry()
	loginEmailEntry.SetPlaceHolder("user@example.com")
	loginPasswordEntry := widget.NewPasswordEntry()
	loginPasswordEntry.SetPlaceHolder("Password")

	regNameEntry := widget.NewEntry()
	regNameEntry.SetPlaceHolder("First Name")
	regSurnameEntry := widget.NewEntry()
	regSurnameEntry.SetPlaceHolder("Last Name")
	regEmailEntry := widget.NewEntry()
	regEmailEntry.SetPlaceHolder("user@example.com")
	regPasswordEntry := widget.NewPasswordEntry()
	regPasswordEntry.SetPlaceHolder("Password")

	var showLogin func()
	var showRegister func()

	loginBtn := widget.NewButtonWithIcon("Login", theme.LoginIcon(), func() {
		server := serverEntry.Text
		email := loginEmailEntry.Text
		password := loginPasswordEntry.Text

		if server == "" || email == "" || password == "" {
			dialog.ShowInformation("Error", "Please fill in all fields.", w)
			return
		}

		err := loginCallback(server, email, password)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		loginPasswordEntry.SetText("")
	})
	loginBtn.Importance = widget.HighImportance

	registerBtn := widget.NewButtonWithIcon("Register", theme.DocumentCreateIcon(), func() {
		server := serverEntry.Text
		name := regNameEntry.Text
		surname := regSurnameEntry.Text
		email := regEmailEntry.Text
		password := regPasswordEntry.Text

		if server == "" || name == "" || surname == "" || email == "" || password == "" {
			dialog.ShowInformation("Error", "Please fill in all fields.", w)
			return
		}

		msg, err := registerCallback(server, email, password, name, surname)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}

		dialog.ShowInformation("Success", msg, w)
		regNameEntry.SetText("")
		regSurnameEntry.SetText("")
		regPasswordEntry.SetText("")
		loginEmailEntry.SetText(email)
		showLogin()
	})
	registerBtn.Importance = widget.HighImportance

	showLogin = func() {
		form := container.NewVBox(
			widget.NewLabelWithStyle("Digital Music Stand - Cloud", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel("Server Address:"), serverEntry,
			widget.NewLabel("Email:"), loginEmailEntry,
			widget.NewLabel("Password:"), loginPasswordEntry,
			widget.NewLabel(""),
			loginBtn,
			widget.NewButton("Need an account? Register here", showRegister),
			widget.NewSeparator(),
			widget.NewButtonWithIcon("Cancel (Work Offline)", theme.CancelIcon(), onCancel),
		)

		scrollForm := container.NewVScroll(container.NewPadded(form))

		centeredCard := container.NewBorder(
			nil, nil,
			widget.NewLabel("          "),
			widget.NewLabel("          "),
			scrollForm,
		)

		wrapper.Objects = []fyne.CanvasObject{container.NewPadded(centeredCard)}
		wrapper.Refresh()
	}

	showRegister = func() {
		form := container.NewVBox(
			widget.NewLabelWithStyle("Create New Account", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			widget.NewLabel("Server Address:"), serverEntry,
			widget.NewLabel("First Name:"), regNameEntry,
			widget.NewLabel("Last Name:"), regSurnameEntry,
			widget.NewLabel("Email:"), regEmailEntry,
			widget.NewLabel("Password:"), regPasswordEntry,
			widget.NewLabel(""),
			registerBtn,
			widget.NewButton("Already have an account? Login", showLogin),
			widget.NewSeparator(),
			widget.NewButtonWithIcon("Cancel (Work Offline)", theme.CancelIcon(), onCancel),
		)

		scrollForm := container.NewVScroll(container.NewPadded(form))

		centeredCard := container.NewBorder(
			nil, nil,
			widget.NewLabel("          "),
			widget.NewLabel("          "),
			scrollForm,
		)

		wrapper.Objects = []fyne.CanvasObject{container.NewPadded(centeredCard)}
		wrapper.Refresh()
	}

	showLogin()
	return wrapper
}
