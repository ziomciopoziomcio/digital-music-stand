package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
)

func BuildPracticeMode(w fyne.Window, db *localdb.DBManager, goBack func()) *fyne.Container {
	contentWrapper := container.NewMax()

	editMode := false
	var showLibrary func()

	showScore := func(score localdb.Score) {
		pdfMgr, err := pdf.NewManager(score.FilePath)

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
				pageLabel.SetText(score.Title + " - Page " + fmt.Sprint(currentPage+1) + " / " + fmt.Sprint(totalPages))
			}
		}

		var scoreDisplay *fyne.Container
		if err == nil {
			updatePage()
			scoreDisplay = container.NewMax(imgCanvas)
		} else {
			scoreDisplay = container.NewCenter(
				widget.NewLabelWithStyle("Could not load PDF file:\n"+score.FilePath, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
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
		// Najpierw otwieramy systemowy File Picker
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			filePath := reader.URI().Path()
			defaultName := reader.URI().Name()
			reader.Close()

			// Następnie prosimy o nazwę utworu
			entry := widget.NewEntry()
			entry.SetText(defaultName)

			var d dialog.Dialog
			submitAction := func() {
				if entry.Text != "" {
					db.AddScore(entry.Text, filePath)
					d.Hide()
					showLibrary()
				}
			}
			entry.OnSubmitted = func(_ string) { submitAction() }

			addBtn := widget.NewButtonWithIcon("Save to Library", theme.DocumentSaveIcon(), submitAction)
			addBtn.Importance = widget.HighImportance

			cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })

			controls := container.NewHBox(layout.NewSpacer(), addBtn, cancelBtn)
			content := container.NewVBox(widget.NewLabel("Enter score title:"), entry, widget.NewLabel(""), controls)

			d = dialog.NewCustomWithoutButtons("Save Score", content, w)
			d.Show()
			w.Canvas().Focus(entry)

		}, w)

		fd.SetFilter(storage.NewExtensionFileFilter([]string{".pdf"}))
		fd.Show()
	}

	openEditDialog := func(score localdb.Score) {
		entry := widget.NewEntry()
		entry.SetText(score.Title)

		var d dialog.Dialog
		submitAction := func() {
			if entry.Text != "" {
				db.UpdateScore(score.ID, entry.Text)
			}
			d.Hide()
			showLibrary()
		}
		entry.OnSubmitted = func(_ string) { submitAction() }

		deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			db.DeleteScore(score.ID)
			d.Hide()
			showLibrary()
		})
		deleteBtn.Importance = widget.DangerImportance

		saveBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), submitAction)
		saveBtn.Importance = widget.HighImportance

		cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })

		controls := container.NewHBox(layout.NewSpacer(), deleteBtn, saveBtn, cancelBtn)
		content := container.NewVBox(widget.NewLabel("Rename score:"), entry, widget.NewLabel(""), controls)

		d = dialog.NewCustomWithoutButtons("Manage Score", content, w)
		d.Show()
		w.Canvas().Focus(entry)
	}

	showLibrary = func() {
		scores, _ := db.GetScores()

		grid := container.NewGridWithColumns(2)
		for _, s := range scores {
			score := s
			icon := theme.DocumentIcon()
			if editMode {
				icon = theme.SettingsIcon()
			}

			btn := widget.NewButtonWithIcon(score.Title, icon, func() {
				if editMode {
					openEditDialog(score)
				} else {
					showScore(score)
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
