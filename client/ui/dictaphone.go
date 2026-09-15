package ui

import (
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
)

func ShowDictaphoneDialog(w fyne.Window, db *localdb.DBManager, recorder *audio.RecorderAudio, scoreID, scoreTitle, profilePath string) {
	var d dialog.Dialog
	listContainer := container.NewVBox()

	var refreshList func()
	refreshList = func() {
		recs, _ := db.GetRecordingsForScore(scoreID)
		var objs []fyne.CanvasObject

		if len(recs) == 0 {
			objs = append(objs, widget.NewLabelWithStyle("No recordings for this score.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}))
		}

		for _, r := range recs {
			rec := r
			titleStr := fmt.Sprintf("%s (%s)", rec.Name, rec.Location)
			dateStr := rec.CreatedAt.Format("2006-01-02 15:04")

			infoLabel := widget.NewLabel(fmt.Sprintf("%s\n%s", titleStr, dateStr))

			playBtn := widget.NewButtonWithIcon("", theme.MediaPlayIcon(), nil)
			playBtn.OnTapped = func() {
				if playBtn.Icon == theme.MediaStopIcon() {
					recorder.StopPlayback()
					playBtn.SetIcon(theme.MediaPlayIcon())
				} else {
					playBtn.SetIcon(theme.MediaStopIcon())
					recorder.PlayRecording(rec.FilePath, func() {
						playBtn.SetIcon(theme.MediaPlayIcon())
					})
				}
			}

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
				nameEntry := widget.NewEntry()
				nameEntry.SetText(rec.Name)
				locEntry := widget.NewEntry()
				locEntry.SetText(rec.Location)

				form := container.NewVBox(
					widget.NewLabel("Name:"), nameEntry,
					widget.NewLabel("Location:"), locEntry,
					widget.NewLabel(fmt.Sprintf("Recorded: %s (Uneditable)", dateStr)),
				)

				dialog.ShowCustomConfirm("Edit Metadata", "Save", "Cancel", form, func(b bool) {
					if b {
						db.UpdateRecording(rec.ID, nameEntry.Text, locEntry.Text)
						refreshList()
					}
				}, w)
			})

			deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
				dialog.ShowConfirm("Delete", "Delete this recording permanently?", func(b bool) {
					if b {
						os.Remove(rec.FilePath)
						db.DeleteRecording(rec.ID)
						refreshList()
					}
				}, w)
			})
			deleteBtn.Importance = widget.DangerImportance

			row := container.NewBorder(nil, nil, nil, container.NewHBox(playBtn, editBtn, deleteBtn), infoLabel)
			objs = append(objs, widget.NewCard("", "", row))
		}
		listContainer.Objects = objs
		listContainer.Refresh()
	}

	toggleBtn := widget.NewButtonWithIcon("Start Recording", theme.MediaRecordIcon(), nil)
	toggleBtn.Importance = widget.HighImportance

	if recorder.IsRecording() {
		toggleBtn.SetText("Stop Recording")
		toggleBtn.SetIcon(theme.MediaStopIcon())
	}

	toggleBtn.OnTapped = func() {
		if recorder.IsRecording() {
			recorder.StopRecording()
			toggleBtn.SetText("Start Recording")
			toggleBtn.SetIcon(theme.MediaRecordIcon())
			refreshList()
		} else {
			fileName := fmt.Sprintf("rec_%d", time.Now().Unix())
			path, err := recorder.StartRecording(profilePath+"/recordings", fileName)
			if err == nil {
				db.AddRecording(scoreID, "New Recording", "Studio", path)
				toggleBtn.SetText("Stop Recording")
				toggleBtn.SetIcon(theme.MediaStopIcon())
			}
		}
	}

	closeBtn := widget.NewButton("Close", func() {
		d.Hide()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Recordings for: %s", scoreTitle), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		container.NewVScroll(listContainer),
		widget.NewSeparator(),
		toggleBtn,
		closeBtn,
	)

	refreshList()

	d = dialog.NewCustomWithoutButtons("Dictaphone", container.NewPadded(content), w)
	d.Resize(fyne.NewSize(500, 600))
	d.Show()
}
