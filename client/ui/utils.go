package ui

import (
	"sync"
	"time"
)

func ThrottledSliderHandler(delay time.Duration, action func(int)) func(float64) {
	var mu sync.Mutex
	var timer *time.Timer
	var lastExec time.Time
	var pendingVal int

	return func(val float64) {
		mu.Lock()
		defer mu.Unlock()

		pendingVal = int(val)
		now := time.Now()

		if now.Sub(lastExec) >= delay {
			lastExec = now
			if timer != nil {
				timer.Stop()
			}
			go action(pendingVal)
		} else {
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(delay, func() {
				mu.Lock()
				lastExec = time.Now()
				valToApply := pendingVal
				mu.Unlock()
				action(valToApply)
			})
		}
	}
}
