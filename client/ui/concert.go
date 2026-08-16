package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
)

func BuildConcertMode(w fyne.Window, db *localdb.DBManager, goBack func(), openSetup func()) *fyne.Container {
	contentWrapper := container.NewMax()

	var showConcertList func()
	var playConcert func(concert localdb.Concert)

	showConcertList = func() {
		backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		newConcertBtn := widget.NewButtonWithIcon("New Concert", theme.ContentAddIcon(), openSetup)
		newConcertBtn.Importance = widget.HighImportance

		topControls := container.NewHBox(newConcertBtn)
		header := container.NewBorder(nil, nil, backBtn, topControls,
			widget.NewLabelWithStyle("Concert Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		concerts, err := db.GetConcerts()
		if err != nil {
			dialog.ShowError(err, w)
			concerts = []localdb.Concert{}
		}

		grid := container.NewGridWrap(fyne.NewSize(240, 180))

		for _, c := range concerts {
			concert := c

			nameLabel := widget.NewLabelWithStyle(concert.Name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			detailsLabel := widget.NewLabelWithStyle(fmt.Sprintf("%s\n%s\nSongs: %d", concert.Location, concert.StartTime, len(concert.Setlist)), fyne.TextAlignCenter, fyne.TextStyle{})

			cardContent := container.NewVBox(nameLabel, detailsLabel, layout.NewSpacer())
			card := widget.NewCard("", "", cardContent)

			openBtn := widget.NewButtonWithIcon("ENTER", theme.MediaPlayIcon(), func() {
				playConcert(concert)
			})
			openBtn.Importance = widget.HighImportance

			item := container.NewBorder(nil, openBtn, nil, nil, card)
			grid.Add(item)
		}

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(container.NewVScroll(grid)))
		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	playConcert = func(concert localdb.Concert) {
		if len(concert.Setlist) == 0 {
			dialog.ShowInformation("Empty Setlist", "This concert has no scores assigned.", w)
			return
		}

		currentSongIdx := 0
		currentPage := 0
		totalPages := 0
		pagesToShow := 1

		var currentPdfMgr *pdf.Manager
		var viewerSize fyne.Size

		pdfContainer := container.NewMax()
		songTitleLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		var renderPage func()
		var loadCurrentSong func(startAtEnd bool)

		renderPage = func() {
			if currentPdfMgr == nil || totalPages == 0 {
				pdfContainer.Objects = []fyne.CanvasObject{
					widget.NewLabelWithStyle("No score loaded", fyne.TextAlignCenter, fyne.TextStyle{}),
				}
				pdfContainer.Refresh()
				pageLabel.SetText("Page 0/0")
				return
			}

			pagesToShow = 1
			if viewerSize.Width > viewerSize.Height && viewerSize.Width > 0 && currentPage+1 < totalPages {
				pagesToShow = 2
			}

			if pagesToShow == 2 {
				img1, err1 := currentPdfMgr.GetPageImage(currentPage)
				img2, err2 := currentPdfMgr.GetPageImage(currentPage + 1)

				if err1 == nil && err2 == nil {
					canvasImg1 := canvas.NewImageFromImage(img1)
					canvasImg1.FillMode = canvas.ImageFillContain

					canvasImg2 := canvas.NewImageFromImage(img2)
					canvasImg2.FillMode = canvas.ImageFillContain

					grid := container.NewGridWithColumns(2, canvasImg1, canvasImg2)
					pdfContainer.Objects = []fyne.CanvasObject{grid}
					pdfContainer.Refresh()

					pageLabel.SetText(fmt.Sprintf("Pages %d-%d / %d", currentPage+1, currentPage+2, totalPages))
					return
				}
			}

			img, err := currentPdfMgr.GetPageImage(currentPage)
			if err != nil {
				pdfContainer.Objects = []fyne.CanvasObject{
					widget.NewLabelWithStyle(fmt.Sprintf("Error rendering page %d", currentPage+1), fyne.TextAlignCenter, fyne.TextStyle{}),
				}
				pdfContainer.Refresh()
				return
			}

			canvasImg := canvas.NewImageFromImage(img)
			canvasImg.FillMode = canvas.ImageFillContain

			pdfContainer.Objects = []fyne.CanvasObject{canvasImg}
			pdfContainer.Refresh()

			pageLabel.SetText(fmt.Sprintf("Page %d / %d", currentPage+1, totalPages))
		}

		loadCurrentSong = func(startAtEnd bool) {
			if currentPdfMgr != nil {
				currentPdfMgr.Close()
				currentPdfMgr = nil
			}

			if currentSongIdx < 0 || currentSongIdx >= len(concert.Setlist) {
				return
			}

			song := concert.Setlist[currentSongIdx]
			songTitleLabel.SetText(fmt.Sprintf("%d/%d: %s", currentSongIdx+1, len(concert.Setlist), song.Title))

			pdfMgr, err := pdf.NewManager(song.FilePath)
			if err != nil {
				pdfContainer.Objects = []fyne.CanvasObject{
					widget.NewLabelWithStyle(fmt.Sprintf("PDF file not found:\n%s", song.FilePath), fyne.TextAlignCenter, fyne.TextStyle{}),
				}
				pdfContainer.Refresh()
				totalPages = 0
				pageLabel.SetText("Page 0/0")
				return
			}

			currentPdfMgr = pdfMgr
			totalPages = pdfMgr.GetPageCount()

			if startAtEnd && totalPages > 0 {
				currentPage = totalPages - 1
				if viewerSize.Width > viewerSize.Height && totalPages >= 2 {
					currentPage = totalPages - 2
				}
			} else {
				currentPage = 0
			}

			renderPage()
		}

		exitConcertBtn := widget.NewButtonWithIcon("Exit", theme.CancelIcon(), func() {
			if currentPdfMgr != nil {
				currentPdfMgr.Close()
			}
			showConcertList()
		})
		exitConcertBtn.Importance = widget.DangerImportance

		prevSongBtn := widget.NewButtonWithIcon("Prev Song", theme.MediaSkipPreviousIcon(), func() {
			if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong(false)
			}
		})

		nextSongBtn := widget.NewButtonWithIcon("Next Song", theme.MediaSkipNextIcon(), func() {
			if currentSongIdx < len(concert.Setlist)-1 {
				currentSongIdx++
				loadCurrentSong(false)
			}
		})

		toolsBtn := widget.NewButtonWithIcon("Tools", theme.SettingsIcon(), func() {
			ShowToolsMenu(w)
		})

		prevPageBtn := widget.NewButtonWithIcon("PREV\nPAGE", theme.NavigateBackIcon(), func() {
			if currentPage > 0 {
				currentPage -= pagesToShow
				if currentPage < 0 {
					currentPage = 0
				}
				renderPage()
			} else if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong(true)
			}
		})
		prevPageBtn.Importance = widget.HighImportance

		nextPageBtn := widget.NewButtonWithIcon("NEXT\nPAGE", theme.NavigateNextIcon(), func() {
			if currentPage+pagesToShow < totalPages {
				currentPage += pagesToShow
				renderPage()
			} else if currentSongIdx < len(concert.Setlist)-1 {
				currentSongIdx++
				loadCurrentSong(false)
			}
		})
		nextPageBtn.Importance = widget.HighImportance

		rightSidebar := container.NewBorder(
			container.NewVBox(pageLabel, widget.NewSeparator(), toolsBtn, widget.NewSeparator()),
			nil, nil, nil,
			container.NewGridWithRows(2, prevPageBtn, nextPageBtn),
		)

		topBar := container.NewBorder(nil, nil, exitConcertBtn, container.NewHBox(prevSongBtn, nextSongBtn), songTitleLabel)

		viewer := newResponsiveViewer(func(size fyne.Size) {
			viewerSize = size
			renderPage()
		})
		viewer.content.Objects = []fyne.CanvasObject{pdfContainer}

		mainView := container.NewBorder(
			container.NewPadded(topBar),
			nil,
			nil,
			container.NewPadded(rightSidebar),
			viewer,
		)

		contentWrapper.Objects = []fyne.CanvasObject{mainView}
		contentWrapper.Refresh()

		loadCurrentSong(false)
	}

	showConcertList()
	return contentWrapper
}
