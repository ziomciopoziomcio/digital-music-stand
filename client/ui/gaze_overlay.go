package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/ziomciopoziomcio/digital-music-stand/client/eyetrack"
)

func NewGazeOverlay(w fyne.Window, app fyne.App, onPrev, onNext func(), isLocked func() bool) (*fyne.Container, func()) {
	tracker := eyetrack.NewTracker()

	enabled := app.Preferences().BoolWithFallback("eyetrack_enabled", false)
	if enabled {
		camID := app.Preferences().IntWithFallback("eyetrack_camera", 0)
		go tracker.Start(camID, false)
	}

	leftZone := canvas.NewRectangle(color.Transparent)
	rightZone := canvas.NewRectangle(color.Transparent)

	gazeProgressCircle := canvas.NewCircle(theme.PrimaryColor())
	gazeProgressCircle.Hide()

	overlay := container.NewWithoutLayout(leftZone, rightZone, gazeProgressCircle)
	stopChan := make(chan struct{})

	go func() {
		var gazeTimer time.Time
		var currentTarget string
		requiredDwellTime := 3000 * time.Millisecond

		for {
			select {
			case <-stopChan:
				tracker.Stop()
				return
			case point := <-tracker.GazeChan:
				winSize := w.Canvas().Size()
				if winSize.Width == 0 || winSize.Height == 0 || (isLocked != nil && isLocked()) {
					leftZone.FillColor = color.Transparent
					rightZone.FillColor = color.Transparent
					gazeProgressCircle.Hide()
					overlay.Refresh()
					continue
				}

				leftZone.Resize(fyne.NewSize(winSize.Width*0.2, winSize.Height))
				leftZone.Move(fyne.NewPos(0, 0))

				rightZone.Resize(fyne.NewSize(winSize.Width*0.2, winSize.Height))
				rightZone.Move(fyne.NewPos(winSize.Width*0.8, 0))

				newTarget := ""
				if point.X < 0.20 {
					newTarget = "PREV"
				} else if point.X > 0.80 {
					newTarget = "NEXT"
				}

				activeColor := color.NRGBA{R: 56, G: 189, B: 248, A: 60}

				if newTarget != currentTarget {
					currentTarget = newTarget
					gazeTimer = time.Now()
					gazeProgressCircle.Hide()

					if newTarget == "PREV" {
						leftZone.FillColor = activeColor
						rightZone.FillColor = color.Transparent
					} else if newTarget == "NEXT" {
						leftZone.FillColor = color.Transparent
						rightZone.FillColor = activeColor
					} else {
						leftZone.FillColor = color.Transparent
						rightZone.FillColor = color.Transparent
					}

					leftZone.Refresh()
					rightZone.Refresh()
					overlay.Refresh()
					continue
				}

				if currentTarget != "" {
					elapsed := time.Since(gazeTimer)
					progress := float64(elapsed) / float64(requiredDwellTime)

					size := float32(120 * progress)
					var circleX float32
					if currentTarget == "PREV" {
						circleX = (winSize.Width * 0.1) - (size / 2)
					} else {
						circleX = (winSize.Width * 0.9) - (size / 2)
					}

					gazeProgressCircle.Resize(fyne.NewSize(size, size))
					gazeProgressCircle.Move(fyne.NewPos(circleX, winSize.Height/2-size/2))
					gazeProgressCircle.Show()
					gazeProgressCircle.Refresh()

					if elapsed >= requiredDwellTime {
						gazeTimer = time.Now()
						gazeProgressCircle.Hide()

						successColor := color.NRGBA{R: 34, G: 197, B: 94, A: 100}
						if currentTarget == "PREV" {
							leftZone.FillColor = successColor
							leftZone.Refresh()
							if onPrev != nil {
								onPrev()
							}
						} else {
							rightZone.FillColor = successColor
							rightZone.Refresh()
							if onNext != nil {
								onNext()
							}
						}

						time.Sleep(1 * time.Second)
						currentTarget = ""
					}
				}
			}
		}
	}()

	stopFunc := func() {
		close(stopChan)
	}

	return overlay, stopFunc
}
