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
	tracker := eyetrack.GetTracker()

	leftZone := canvas.NewRectangle(color.Transparent)
	rightZone := canvas.NewRectangle(color.Transparent)

	gazeProgressCircle := canvas.NewCircle(theme.PrimaryColor())
	gazeProgressCircle.Hide()

	overlay := container.NewWithoutLayout(leftZone, rightZone, gazeProgressCircle)
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				tracker.Stop()
				return
			case <-ticker.C:
				if app.Preferences().BoolWithFallback("eyetrack_enabled", false) {
					camID := app.Preferences().IntWithFallback("eyetrack_camera", 0)
					_ = tracker.Start(camID, false)
				} else {
					tracker.Stop()
				}
			}
		}
	}()

	go func() {
		var gazeTimer time.Time
		var currentTarget string
		requiredDwellTime := 3000 * time.Millisecond

		for {
			select {
			case <-stopChan:
				return
			case point := <-tracker.GazeChan:
				enabled := app.Preferences().BoolWithFallback("eyetrack_enabled", false)
				winSize := w.Canvas().Size()

				if !enabled || winSize.Width == 0 || winSize.Height == 0 || (isLocked != nil && isLocked()) {
					leftZone.FillColor = color.Transparent
					rightZone.FillColor = color.Transparent
					gazeProgressCircle.Hide()
					overlay.Refresh()
					currentTarget = ""
					continue
				}

				leftZone.Resize(fyne.NewSize(winSize.Width*0.2, winSize.Height))
				leftZone.Move(fyne.NewPos(0, 0))

				rightZone.Resize(fyne.NewSize(winSize.Width*0.2, winSize.Height))
				rightZone.Move(fyne.NewPos(winSize.Width*0.8, 0))

				minX := app.Preferences().FloatWithFallback("eyetrack_min_x", 0.3)
				maxX := app.Preferences().FloatWithFallback("eyetrack_max_x", 0.7)

				if maxX <= minX {
					maxX = minX + 0.1
				}

				normX := (point.X - minX) / (maxX - minX)
				if normX < 0 {
					normX = 0
				}
				if normX > 1 {
					normX = 1
				}

				newTarget := ""
				if normX < 0.20 {
					newTarget = "PREV"
				} else if normX > 0.80 {
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
