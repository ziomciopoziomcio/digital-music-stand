package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func BuildPracticeMode(w fyne.Window, goBack func()) *fyne.Container {
	contentWrapper := container.NewMax()

	var showLibrary func()

	showScore := func(scoreName string) {
		nextBtn := widget.NewButtonWithIcon("Next Page", theme.MenuDropUpIcon(), func() {})
		nextBtn.Importance = widget.HighImportance

		prevBtn := widget.NewButtonWithIcon("Prev Page", theme.MenuDropDownIcon(), func() {})
		prevBtn.Importance = widget.HighImportance

		utilitiesBtn := widget.NewButtonWithIcon("Utilities / Tools", theme.SettingsIcon(), func() {})

		backBtn := widget.NewButtonWithIcon("Back to Library", theme.NavigateBackIcon(), func() {
			showLibrary()
		})
		backBtn.Importance = widget.WarningImportance

		controlsPanel := container.NewVBox(
			backBtn,
			widget.NewSeparator(),
			widget.NewLabel(""),
			prevBtn,
			widget.NewLabel(""),
			nextBtn,
			layout.NewSpacer(),
			utilitiesBtn,
		)

		scoreDisplay := container.NewCenter(
			widget.NewLabelWithStyle("Viewing: "+scoreName+"\n\n[ PDF / Image rendering goes here ]", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		)

		view := container.NewBorder(nil, nil, nil, container.NewPadded(controlsPanel), scoreDisplay)

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	showLibrary = func() {
		scores := []string{
			"Beethoven - Symphony No. 5",
			"Mozart - Fur Elise",
			"Chopin - Nocturne op. 9",
			"Bach - Sonata in G Minor",
			"Vivaldi - Four Seasons",
			"Tchaikovsky - Swan Lake",
		}

		grid := container.NewGridWithColumns(2)
		for _, s := range scores {
			scoreName := s
			btn := widget.NewButtonWithIcon(scoreName, theme.DocumentIcon(), func() {
				showScore(scoreName)
			})
			btn.Importance = widget.HighImportance
			grid.Add(btn)
		}

		backToDashBtn := widget.NewButtonWithIcon("Back to Dashboard", theme.HomeIcon(), goBack)
		backToDashBtn.Importance = widget.WarningImportance

		header := container.NewBorder(nil, nil, backToDashBtn, nil, widget.NewLabelWithStyle("Practice Mode - My Library", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(container.NewVScroll(grid)))

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	showLibrary()

	return contentWrapper
}
