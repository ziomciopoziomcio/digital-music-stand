package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Digital Music Stand - Login")

	title := widget.NewLabel("Login to the music stand")
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email address")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	loginButton := widget.NewButton("Login", func() {
		log.Printf("Login attempt: %s", emailEntry.Text)
		myWindow.SetContent(widget.NewLabel("loging..."))
	})

	formContainer := container.NewVBox(
		title,
		widget.NewLabel(""),
		emailEntry,
		passwordEntry,
		widget.NewLabel(""),
		loginButton,
	)

	paddedContainer := container.NewPadded(formContainer)

	myWindow.SetContent(paddedContainer)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}
