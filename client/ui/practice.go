package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
)

func BuildPracticeMode(w fyne.Window, goBack func()) *fyne.Container {
	contentWrapper := container.NewMax()

	scores := []string{
		"Beethoven - Symphony No. 5",
		"Mozart - Fur Elise",
		"Chopin - Nocturne op. 9",
		"Bach - Sonata in G Minor",
		"Vivaldi - Four Seasons",
		"Tchaikovsky - Swan Lake",
	}

	editMode := false

	var showLibrary func()

	showScore := func(scoreName string) {
		pdfMgr, err := pdf.NewManager("sample.pdf")

		currentPage := 0
		totalPages := 0
		if err == nil {
			totalPages = pdfMgr.GetPageCount()
		}

		imgCanvas := canvas.NewImageFromImage(nil)
		imgCanvas.FillMode = canvas.ImageFillContain

		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		updatePage := func() {
			if pdfMgr == nil {
				return
			}
			img, imgErr := pdfMgr.GetPageImage(currentPage)
			if imgErr == nil {
				imgCanvas.Image = img
				imgCanvas.Refresh()
				pageLabel.SetText(scoreName + " - Page " + fmt.Sprint(currentPage+1) + " / " + fmt.Sprint(totalPages))
			}
		}

		var scoreDisplay *fyne.Container
		if err == nil {
			updatePage()
			scoreDisplay = container.NewMax(imgCanvas)
		} else {
			scoreDisplay = container.NewCenter(
				widget.NewLabelWithStyle("Could not load sample.pdf\nPlease put sample.pdf in the client folder.", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			)
		}

		nextBtn := widget.NewButtonWithIcon("Next Page", theme.MenuDropUpIcon(), func() {
			if currentPage < totalPages-1 {
				currentPage++
				updatePage()
			}
		})
		nextBtn.Importance = widget.HighImportance

		prevBtn := widget.NewButtonWithIcon("Prev Page", theme.MenuDropDownIcon(), func() {
			if currentPage > 0 {
				currentPage--
				updatePage()
			}
		})
		prevBtn.Importance = widget.HighImportance

		utilitiesBtn := widget.NewButtonWithIcon("Utilities / Tools", theme.SettingsIcon(), func() {})

		backBtn := widget.NewButtonWithIcon("Back to Library", theme.NavigateBackIcon(), func() {
			if pdfMgr != nil {
				pdfMgr.Close()
			}
			showLibrary()
		})
		backBtn.Importance = widget.WarningImportance

		controlsPanel := container.NewVBox(
			backBtn,
			widget.NewSeparator(),
			pageLabel,
			widget.NewLabel(""),
			prevBtn,
			widget.NewLabel(""),
			nextBtn,
			layout.NewSpacer(),
			utilitiesBtn,
		)

		view := container.NewBorder(nil, nil, nil, container.NewPadded(controlsPanel), scoreDisplay)

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	openAddDialog := func() {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("Score title...")

		var d dialog.Dialog

		submitAction := func() {
			if entry.Text != "" {
				scores = append(scores, entry.Text)
				d.Hide()
				showLibrary()
			}
		}

		entry.OnSubmitted = func(_ string) {
			submitAction()
		}

		addBtn := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), submitAction)
		addBtn.Importance = widget.HighImportance

		cancelBtn := widget.NewButton("Cancel", func() {
			d.Hide()
		})

		controls := container.NewHBox(layout.NewSpacer(), addBtn, cancelBtn)
		content := container.NewVBox(widget.NewLabel("New Score Name:"), entry, widget.NewLabel(""), controls)

		d = dialog.NewCustomWithoutButtons("Add Score", content, w)
		d.Show()

		w.Canvas().Focus(entry)
	}

	openEditDialog := func(index int, currentName string) {
		entry := widget.NewEntry()
		entry.SetText(currentName)

		var d dialog.Dialog

		submitAction := func() {
			if entry.Text != "" {
				scores[index] = entry.Text
			}
			d.Hide()
			showLibrary()
		}

		entry.OnSubmitted = func(_ string) {
			submitAction()
		}

		deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			scores = append(scores[:index], scores[index+1:]...)
			d.Hide()
			showLibrary()
		})
		deleteBtn.Importance = widget.DangerImportance

		saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), submitAction)
		saveBtn.Importance = widget.HighImportance

		cancelBtn := widget.NewButton("Cancel", func() {
			d.Hide()
		})

		controls := container.NewHBox(layout.NewSpacer(), deleteBtn, saveBtn, cancelBtn)
		content := container.NewVBox(widget.NewLabel("Rename score:"), entry, widget.NewLabel(""), controls)

		d = dialog.NewCustomWithoutButtons("Manage Score", content, w)
		d.Show()

		w.Canvas().Focus(entry)
	}

	showLibrary = func() {
		grid := container.NewGridWithColumns(2)
		for i, s := range scores {
			index := i
			scoreName := s

			icon := theme.DocumentIcon()
			if editMode {
				icon = theme.SettingsIcon()
			}

			btn := widget.NewButtonWithIcon(scoreName, icon, func() {
				if editMode {
					openEditDialog(index, scoreName)
				} else {
					showScore(scoreName)
				}
			})

			if editMode {
				btn.Importance = widget.WarningImportance
			} else {
				btn.Importance = widget.HighImportance
			}

			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(0, 80))

			tile := container.NewMax(spacer, btn)
			grid.Add(tile)
		}

		addBtn := widget.NewButtonWithIcon("Add Score", theme.ContentAddIcon(), openAddDialog)
		addBtn.Importance = widget.HighImportance

		editToggleBtn := widget.NewButtonWithIcon("Manage", theme.SettingsIcon(), func() {
			editMode = !editMode
			showLibrary()
		})

		if editMode {
			editToggleBtn.SetText("Done")
			editToggleBtn.SetIcon(theme.ConfirmIcon())
			editToggleBtn.Importance = widget.HighImportance
		}

		topControls := container.NewHBox(addBtn, editToggleBtn)

		backToDashBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backToDashBtn.Importance = widget.WarningImportance

		header := container.NewBorder(nil, nil, backToDashBtn, topControls, widget.NewLabelWithStyle("Practice Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(container.NewVScroll(grid)))

		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	showLibrary()

	return contentWrapper
}
