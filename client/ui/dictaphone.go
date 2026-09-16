package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ziomciopoziomcio/digital-music-stand/client/audio"
	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
)

func ShowDictaphoneDialog(w fyne.Window, db *localdb.DBManager, recorder *audio.RecorderAudio, scoreID, scoreTitle, profilePath string) {
	var d dialog.Dialog
	listContainer := container.NewVBox()

	currentRecordPath := ""
	if recorder.IsRecording() {
		currentRecordPath = recorder.GetRecordingPath()
	}

	playerContainer := container.NewVBox()
	playerContainer.Hide()

	slider := widget.NewSlider(0, 1)
	timeLabel := widget.NewLabelWithStyle("00:00 / 00:00", fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})
	nowPlayingLabel := widget.NewLabelWithStyle("Playing: ...", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	playPauseBtn := widget.NewButtonWithIcon("", theme.MediaPauseIcon(), nil)
	stopBtn := widget.NewButtonWithIcon("", theme.MediaStopIcon(), nil)
	rewindBtn := widget.NewButtonWithIcon("-5s", theme.MediaFastRewindIcon(), nil)
	forwardBtn := widget.NewButtonWithIcon("+5s", theme.MediaFastForwardIcon(), nil)

	var updateTicker *time.Ticker
	var ignoreSliderChange bool

	formatTime := func(sec float64) string {
		s := int(sec)
		return fmt.Sprintf("%02d:%02d", s/60, s%60)
	}

	startPlayerUIUpdater := func() {
		if updateTicker != nil {
			updateTicker.Stop()
		}
		updateTicker = time.NewTicker(200 * time.Millisecond)
		go func() {
			for range updateTicker.C {
				playing, paused, currentSec, totalSec, _ := recorder.GetPlaybackState()

				if playing && totalSec > 0 && currentSec >= totalSec {
					recorder.StopPlayback()
					playerContainer.Hide()
					playPauseBtn.SetIcon(theme.MediaPlayIcon())
					updateTicker.Stop()
					continue
				}

				if !playing {
					playerContainer.Hide()
					updateTicker.Stop()
					continue
				}
				if !paused {
					ignoreSliderChange = true
					slider.Max = totalSec
					slider.SetValue(currentSec)
					ignoreSliderChange = false
					timeLabel.SetText(fmt.Sprintf("%s / %s", formatTime(currentSec), formatTime(totalSec)))
				}
			}
		}()
	}

	playPauseBtn.OnTapped = func() {
		paused := recorder.TogglePause()
		if paused {
			playPauseBtn.SetIcon(theme.MediaPlayIcon())
		} else {
			playPauseBtn.SetIcon(theme.MediaPauseIcon())
		}
	}

	stopBtn.OnTapped = func() {
		recorder.StopPlayback()
		playerContainer.Hide()
		if updateTicker != nil {
			updateTicker.Stop()
		}
	}

	rewindBtn.OnTapped = func() { recorder.SeekRelative(-5.0) }
	forwardBtn.OnTapped = func() { recorder.SeekRelative(5.0) }

	slider.OnChanged = func(val float64) {
		if ignoreSliderChange {
			return
		}
		recorder.SeekAbsolute(val)
	}

	controlsRow := container.NewHBox(layout.NewSpacer(), rewindBtn, playPauseBtn, stopBtn, forwardBtn, layout.NewSpacer())
	playerContainer.Objects = []fyne.CanvasObject{
		widget.NewSeparator(),
		nowPlayingLabel,
		slider,
		timeLabel,
		controlsRow,
	}

	playing, paused, cur, tot, fpath := recorder.GetPlaybackState()
	if playing {
		playerContainer.Show()
		nowPlayingLabel.SetText(fmt.Sprintf("Playing: %s", filepath.Base(fpath)))
		ignoreSliderChange = true
		slider.Max = tot
		slider.SetValue(cur)
		ignoreSliderChange = false
		if paused {
			playPauseBtn.SetIcon(theme.MediaPlayIcon())
		} else {
			playPauseBtn.SetIcon(theme.MediaPauseIcon())
		}
		startPlayerUIUpdater()
	}

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

			playBtn := widget.NewButtonWithIcon("Play", theme.MediaPlayIcon(), nil)

			if recorder.IsRecording() {
				playBtn.Disable()
			}

			playBtn.OnTapped = func() {
				if recorder.IsRecording() {
					return
				}
				err := recorder.PlayRecording(rec.FilePath, nil)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				nowPlayingLabel.SetText(fmt.Sprintf("Playing: %s", rec.Name))
				playPauseBtn.SetIcon(theme.MediaPauseIcon())
				playerContainer.Show()
				startPlayerUIUpdater()
			}

			editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
				nameEntry := widget.NewEntry()
				nameEntry.SetText(rec.Name)
				locEntry := widget.NewEntry()
				locEntry.SetText(rec.Location)

				form := container.NewVBox(
					widget.NewLabel("Name:"), nameEntry,
					widget.NewLabel("Location:"), locEntry,
					widget.NewLabel(fmt.Sprintf("Recorded: %s", dateStr)),
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
			if currentRecordPath != "" {
				db.AddRecording(scoreID, "New Recording", "Studio", currentRecordPath)
				currentRecordPath = ""
			}
			toggleBtn.SetText("Start Recording")
			toggleBtn.SetIcon(theme.MediaRecordIcon())
			refreshList()
		} else {
			if playing, _, _, _, _ := recorder.GetPlaybackState(); playing {
				recorder.StopPlayback()
			}

			fileName := fmt.Sprintf("rec_%d", time.Now().Unix())
			recordingsPath := filepath.Join(profilePath, "recordings")
			path, err := recorder.StartRecording(recordingsPath, fileName)
			if err == nil {
				currentRecordPath = path
				toggleBtn.SetText("Stop Recording")
				toggleBtn.SetIcon(theme.MediaStopIcon())
				refreshList()
			} else {
				dialog.ShowError(err, w)
			}
		}
	}

	closeBtn := widget.NewButton("Close", func() {
		if updateTicker != nil {
			updateTicker.Stop()
		}
		d.Hide()
	})

	listScroll := container.NewVScroll(listContainer)
	listScroll.SetMinSize(fyne.NewSize(450, 300))

	content := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("Recordings for: %s", scoreTitle), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		listScroll,
		playerContainer,
		widget.NewSeparator(),
		toggleBtn,
		closeBtn,
	)

	refreshList()

	scrollDialogContent := container.NewVScroll(container.NewPadded(content))
	d = dialog.NewCustomWithoutButtons("Dictaphone", scrollDialogContent, w)

	winSize := w.Canvas().Size()
	targetWidth := float32(500)
	targetHeight := float32(700)

	if winSize.Width < targetWidth {
		targetWidth = winSize.Width * 0.95
	}
	if winSize.Height < targetHeight {
		targetHeight = winSize.Height * 0.95
	}

	d.Resize(fyne.NewSize(targetWidth, targetHeight))
	d.Show()
}
