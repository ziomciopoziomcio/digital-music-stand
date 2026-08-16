package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
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

		pdfViewer := widget.NewLabelWithStyle("Loading...", fyne.TextAlignCenter, fyne.TextStyle{})
		songTitleLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

		var renderPage func()
		var loadCurrentSong func()

		renderPage = func() {
			song := concert.Setlist[currentSongIdx]
			pdfViewer.SetText(fmt.Sprintf("Score: %s | Page %d/%d", song.Title, currentPage+1, totalPages))
		}

		loadCurrentSong = func() {
			if currentSongIdx < 0 || currentSongIdx >= len(concert.Setlist) {
				return
			}
			song := concert.Setlist[currentSongIdx]
			songTitleLabel.SetText(fmt.Sprintf("%d/%d: %s", currentSongIdx+1, len(concert.Setlist), song.Title))

			pdfMgr, err := pdf.NewManager(song.FilePath)
			if err != nil {
				pdfViewer.SetText(fmt.Sprintf("PDF not found:\n%s", song.FilePath))
				totalPages = 0
				return
			}

			totalPages = pdfMgr.GetPageCount()
			currentPage = 0
			renderPage()
		}

		exitConcertBtn := widget.NewButtonWithIcon("Exit", theme.CancelIcon(), func() {
			showConcertList()
		})
		exitConcertBtn.Importance = widget.DangerImportance

		prevSongBtn := widget.NewButtonWithIcon("Prev Song", theme.MediaSkipPreviousIcon(), func() {
			if currentSongIdx > 0 {
				currentSongIdx--
				loadCurrentSong()
			}
		})

		nextSongBtn := widget.NewButtonWithIcon("Next Song", theme.MediaSkipNextIcon(), func() {
			if currentSongIdx < len(concert.Setlist)-1 {
				currentSongIdx++
				loadCurrentSong()
			}
		})

		prevPageBtn := widget.NewButtonWithIcon("PREV PAGE", theme.NavigateBackIcon(), func() {
			if currentPage > 0 {
				currentPage--
				renderPage()
			}
		})
		prevPageBtn.Importance = widget.HighImportance

		nextPageBtn := widget.NewButtonWithIcon("NEXT PAGE", theme.NavigateNextIcon(), func() {
			if currentPage < totalPages-1 {
				currentPage++
				renderPage()
			}
		})
		nextPageBtn.Importance = widget.HighImportance

		topBar := container.NewBorder(nil, nil, exitConcertBtn, container.NewHBox(prevSongBtn, nextSongBtn), songTitleLabel)
		bottomBar := container.NewGridWithColumns(2, prevPageBtn, nextPageBtn)

		mainView := container.NewBorder(container.NewPadded(topBar), container.NewPadded(bottomBar), nil, nil, container.NewCenter(pdfViewer))

		contentWrapper.Objects = []fyne.CanvasObject{mainView}
		contentWrapper.Refresh()

		loadCurrentSong()
	}

	showConcertList()
	return contentWrapper
}
