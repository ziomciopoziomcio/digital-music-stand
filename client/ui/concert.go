package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/client/pdf"
)

var MockConcerts = []Concert{
	{
		ID:        "conc-1",
		Name:      "Dress Rehearsal",
		Location:  "Chamber Hall",
		StartTime: "2026-08-20 19:00",
		Setlist: []localdb.Score{
			{ID: 1, Title: "Imperial March", FilePath: "/mock/path1"},
			{ID: 2, Title: "Symphony No. 5", FilePath: "/mock/path2"},
		},
	},
}

func BuildConcertMode(w fyne.Window, db *localdb.DBManager, goBack func(), openSetup func()) *fyne.Container {
	contentWrapper := container.NewMax()

	var showConcertList func()
	var playConcert func(concert Concert)

	showConcertList = func() {
		backBtn := widget.NewButtonWithIcon("Dashboard", theme.HomeIcon(), goBack)
		backBtn.Importance = widget.WarningImportance

		newConcertBtn := widget.NewButtonWithIcon("New Concert", theme.ContentAddIcon(), openSetup)
		newConcertBtn.Importance = widget.HighImportance

		header := container.NewBorder(nil, nil, backBtn, newConcertBtn,
			widget.NewLabelWithStyle("Concert Mode", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))

		list := widget.NewList(
			func() int { return len(MockConcerts) },
			func() fyne.CanvasObject {
				nameLabel := widget.NewLabel("")
				detailsLabel := widget.NewLabel("")
				playBtn := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), nil)

				infoBox := container.NewVBox(nameLabel, detailsLabel)
				return container.NewBorder(nil, nil, nil, playBtn, infoBox)
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				c := MockConcerts[i]
				cont := o.(*fyne.Container)

				infoBox := cont.Objects[0].(*fyne.Container)
				nameLabel := infoBox.Objects[0].(*widget.Label)
				detailsLabel := infoBox.Objects[1].(*widget.Label)
				playBtn := cont.Objects[1].(*widget.Button)

				nameLabel.SetText(c.Name)
				nameLabel.TextStyle = fyne.TextStyle{Bold: true}
				detailsLabel.SetText(fmt.Sprintf("Location: %s | Time: %s | Songs: %d", c.Location, c.StartTime, len(c.Setlist)))

				playBtn.OnTapped = func() {
					playConcert(c)
				}
			},
		)

		view := container.NewBorder(header, nil, nil, nil, container.NewPadded(list))
		contentWrapper.Objects = []fyne.CanvasObject{view}
		contentWrapper.Refresh()
	}

	playConcert = func(concert Concert) {
		if len(concert.Setlist) == 0 {
			return
		}

		currentSongIdx := 0
		currentPage := 0
		totalPages := 0

		pdfViewer := widget.NewLabelWithStyle("Loading score...", fyne.TextAlignCenter, fyne.TextStyle{})
		songTitleLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		pageLabel := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{})

		var loadCurrentSong func()

		loadCurrentSong = func() {
			if currentSongIdx < 0 || currentSongIdx >= len(concert.Setlist) {
				return
			}
			song := concert.Setlist[currentSongIdx]
			songTitleLabel.SetText(fmt.Sprintf("Song %d/%d: %s", currentSongIdx+1, len(concert.Setlist), song.Title))

			pdfMgr, err := pdf.NewManager(song.FilePath)
			if err != nil {
				pdfViewer.SetText(fmt.Sprintf("Score PDF not found:\n%s", song.FilePath))
				totalPages = 0
				pageLabel.SetText("Page 0/0")
				return
			}

			totalPages = pdfMgr.GetPageCount()
			currentPage = 0

			renderPage := func() {
				pdfViewer.SetText(fmt.Sprintf("Score: %s | Page %d/%d", song.Title, currentPage+1, totalPages))
				pageLabel.SetText(fmt.Sprintf("Page %d of %d", currentPage+1, totalPages))
			}
			renderPage()
		}

		exitConcertBtn := widget.NewButtonWithIcon("Exit Concert", theme.CancelIcon(), func() {
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

		prevPageBtn := widget.NewButtonWithIcon("<- Page", theme.NavigateBackIcon(), func() {
			if currentPage > 0 {
				currentPage--
				pageLabel.SetText(fmt.Sprintf("Page %d of %d", currentPage+1, totalPages))
			}
		})

		nextPageBtn := widget.NewButtonWithIcon("Page ->", theme.NavigateNextIcon(), func() {
			if currentPage < totalPages-1 {
				currentPage++
				pageLabel.SetText(fmt.Sprintf("Page %d of %d", currentPage+1, totalPages))
			}
		})

		topBar := container.NewBorder(nil, nil, exitConcertBtn, container.NewHBox(prevSongBtn, nextSongBtn), songTitleLabel)
		bottomBar := container.NewBorder(nil, nil, prevPageBtn, nextPageBtn, pageLabel)

		mainView := container.NewBorder(topBar, bottomBar, nil, nil, container.NewCenter(pdfViewer))

		contentWrapper.Objects = []fyne.CanvasObject{mainView}
		contentWrapper.Refresh()

		loadCurrentSong()
	}

	showConcertList()
	return contentWrapper
}
