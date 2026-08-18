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

func BuildPracticeMode(w fyne.Window, db *localdb.DBManager, onScoresChanged func(), goBack func()) *fyne.Container {
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

		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		currentPagesToShow := 1

		getPagesToShow := func(size fyne.Size) int {
			if size.Height == 0 {
				return 1
			}
			aspect := size.Width / size.Height
			pages := int(aspect / 0.7)
			if pages < 1 {
				return 1
			}
			if pages > 3 {
				return 3
			}
			return pages
		}

		var scoreDisplay *responsiveViewer
		var updatePage func(size fyne.Size)

		scoreDisplay = newResponsiveViewer(func(s fyne.Size) {
			if updatePage != nil {
				newPts := getPagesToShow(s)
				if newPts != currentPagesToShow {
					updatePage(s)
				}
			}
		})

		updatePage = func(size fyne.Size) {
			if pdfMgr == nil {
				return
			}

			pagesToShow := getPagesToShow(size)
			currentPagesToShow = pagesToShow

			var pageImages []fyne.CanvasObject
			for i := 0; i < pagesToShow; i++ {
				pageIdx := currentPage + i
				if pageIdx < totalPages {
					img, imgErr := pdfMgr.GetPageImage(pageIdx)
					if imgErr == nil {
						imgCanvas := canvas.NewImageFromImage(img)
						imgCanvas.FillMode = canvas.ImageFillContain
						pageImages = append(pageImages, imgCanvas)
					} else {
						pageImages = append(pageImages, layout.NewSpacer())
					}
				} else {
					pageImages = append(pageImages, layout.NewSpacer())
				}
			}

			grid := container.NewGridWithRows(1, pageImages...)
			scoreDisplay.content.Objects = []fyne.CanvasObject{grid}
			scoreDisplay.content.Refresh()

			endPage := currentPage + pagesToShow
			if endPage > totalPages {
				endPage = totalPages
			}

			if pagesToShow == 1 || currentPage == endPage-1 {
				pageLabel.SetText(fmt.Sprintf("%s\n\nPage %d / %d", score.Title, currentPage+1, totalPages))
			} else {
				pageLabel.SetText(fmt.Sprintf("%s\n\nPages %d-%d / %d", score.Title, currentPage+1, endPage, totalPages))
			}
		}

		if err == nil {
			initSize := w.Canvas().Size()
			initSize.Width -= 180
			updatePage(initSize)
		} else {
			scoreDisplay.content.Objects = []fyne.CanvasObject{
				container.NewCenter(widget.NewLabelWithStyle("Could not load PDF file:\n"+score.FilePath, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
			}
			scoreDisplay.content.Refresh()
		}

		nextBtn := widget.NewButtonWithIcon("Next", theme.MenuDropUpIcon(), func() {
			pts := getPagesToShow(scoreDisplay.Size())
			if currentPage+pts < totalPages {
				currentPage += pts
				updatePage(scoreDisplay.Size())
			}
		})
		nextBtn.Importance = widget.HighImportance

		prevBtn := widget.NewButtonWithIcon("Prev", theme.MenuDropDownIcon(), func() {
			pts := getPagesToShow(scoreDisplay.Size())
			currentPage -= pts
			if currentPage < 0 {
				currentPage = 0
			}
			updatePage(scoreDisplay.Size())
		})
		prevBtn.Importance = widget.HighImportance

		utilitiesBtn := widget.NewButtonWithIcon("Tools", theme.SettingsIcon(), func() {
			ShowToolsMenu(w)
		})
		utilitiesBtn.Importance = widget.HighImportance

		backBtn := widget.NewButtonWithIcon("Library", theme.NavigateBackIcon(), func() {
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
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			filePath := reader.URI().Path()
			defaultName := reader.URI().Name()
			reader.Close()

			entry := widget.NewEntry()
			entry.SetText(defaultName)

			var d dialog.Dialog
			submitAction := func() {
				if entry.Text != "" {
					db.AddScore(entry.Text, filePath)
					d.Hide()
					showLibrary()
					onScoresChanged()
				}
			}
			entry.OnSubmitted = func(_ string) { submitAction() }

			addBtn := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), submitAction)
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
			onScoresChanged()
		}
		entry.OnSubmitted = func(_ string) { submitAction() }

		deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			db.DeleteScore(score.ID)
			d.Hide()
			showLibrary()
			onScoresChanged()
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

type responsiveViewer struct {
	widget.BaseWidget
	content  *fyne.Container
	onResize func(size fyne.Size)
	lastSize fyne.Size
}

func newResponsiveViewer(onResize func(size fyne.Size)) *responsiveViewer {
	v := &responsiveViewer{
		onResize: onResize,
		content:  container.NewMax(),
	}
	v.ExtendBaseWidget(v)
	return v
}

func (v *responsiveViewer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(v.content)
}

func (v *responsiveViewer) Resize(s fyne.Size) {
	v.BaseWidget.Resize(s)
	if s.Width != v.lastSize.Width || s.Height != v.lastSize.Height {
		v.lastSize = s
		if v.onResize != nil {
			v.onResize(s)
		}
	}
}
