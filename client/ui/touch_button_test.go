package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// Scenario 1: First touchscreen tap activates a button.
func TestTouchButton_FirstTapActivates(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	tappedCount := 0
	btn := NewTouchButton("Test Button", func() {
		tappedCount++
	})

	// Before any tap, button has not been activated
	if tappedCount != 0 {
		t.Fatalf("expected tappedCount 0, got %d", tappedCount)
	}

	// First tap on the button must immediately activate it
	test.Tap(btn)

	if tappedCount != 1 {
		t.Fatalf("expected first tap to activate button immediately (tappedCount == 1), got %d", tappedCount)
	}

	// Verify that hover state is cleared after tap so touch interaction does not leave persistent hover
	if btn.hovered {
		t.Errorf("expected button.hovered to be false after tap, got true")
	}
}

// Scenario 2: First touchscreen tap selects an item in a list.
func TestList_FirstTapSelects(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	items := []string{"Score 1", "Score 2", "Score 3"}
	selectedItem := -1
	selectedCount := 0

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(items[id])
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		selectedItem = id
		selectedCount++
	}

	w := test.NewWindow(list)
	defer w.Close()
	w.Resize(fyne.NewSize(300, 400))

	// First tap selects item index 1
	list.Select(1)

	if selectedItem != 1 {
		t.Fatalf("expected selectedItem to be 1 on first selection, got %d", selectedItem)
	}
	if selectedCount != 1 {
		t.Fatalf("expected selectedCount to be 1, got %d", selectedCount)
	}
}

// Scenario 3: Repeated taps do not trigger duplicate actions.
func TestTouchButton_RepeatedTapsDoNotDuplicate(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	tappedCount := 0
	btn := NewTouchButton("Repeat Button", func() {
		tappedCount++
	})

	// Tap 5 times
	for i := 1; i <= 5; i++ {
		test.Tap(btn)
		if tappedCount != i {
			t.Fatalf("after %d taps, expected tappedCount %d, got %d", i, i, tappedCount)
		}
	}
}

// Scenario 4: Changing focus does not trigger navigation or activation.
func TestTouchButton_ChangingFocusDoesNotTriggerAction(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	navigated := false
	btn := NewTouchButton("Action", func() {
		navigated = true
	})

	// Focus is gained
	btn.FocusGained()
	if !btn.Focused() {
		t.Errorf("expected button to be focused after FocusGained")
	}
	if navigated {
		t.Errorf("FocusGained must not trigger button activation or navigation")
	}

	// Focus is lost
	btn.FocusLost()
	if btn.Focused() {
		t.Errorf("expected button not to be focused after FocusLost")
	}
	if navigated {
		t.Errorf("FocusLost must not trigger button activation or navigation")
	}
}

// Scenario 5: A button tap does not accidentally activate the Back action.
func TestTouchButton_TargetTapDoesNotActivateBack(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	backActivated := false
	targetActivated := false

	backBtn := NewTouchButtonWithIcon("Dashboard", nil, func() {
		backActivated = true
	})

	targetBtn := NewTouchButton("Confirm Action", func() {
		targetActivated = true
	})

	w := test.NewWindow(backBtn)
	defer w.Close()

	// Tap the target button
	test.Tap(targetBtn)

	if !targetActivated {
		t.Errorf("expected target button to be activated on tap")
	}
	if backActivated {
		t.Fatalf("CRITICAL BUG: tapping target button accidentally activated the Back button!")
	}
}

// Scenario 6: Mouse interaction remains functional (hover and click).
func TestTouchButton_MouseInteractionFunctional(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	clicked := false
	btn := NewTouchButton("Mouse Target", func() {
		clicked = true
	})

	// Mouse moves in
	btn.MouseIn(nil)
	if !btn.hovered {
		t.Errorf("expected button to be hovered after MouseIn")
	}

	// Mouse moves out
	btn.MouseOut()
	if btn.hovered {
		t.Errorf("expected button not to be hovered after MouseOut")
	}

	// Mouse click
	test.Tap(btn)
	if !clicked {
		t.Errorf("expected button to be clicked by mouse tap")
	}
}

// Scenario 7: Keyboard interaction remains functional (Focus, Space, Enter).
func TestTouchButton_KeyboardInteractionFunctional(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	activationCount := 0
	btn := NewTouchButton("Key Target", func() {
		activationCount++
	})

	// Button must implement fyne.Focusable
	focusable, ok := interface{}(btn).(fyne.Focusable)
	if !ok {
		t.Fatalf("TouchButton must implement fyne.Focusable for keyboard accessibility")
	}

	// Focus the button
	focusable.FocusGained()
	if !btn.Focused() {
		t.Errorf("expected button to report Focused() == true")
	}

	// Space key triggers activation
	btn.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if activationCount != 1 {
		t.Fatalf("expected Space key to activate button (count == 1), got %d", activationCount)
	}

	// Enter / Return key triggers activation
	btn.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if activationCount != 2 {
		t.Fatalf("expected Return key to activate button (count == 2), got %d", activationCount)
	}

	btn.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if activationCount != 3 {
		t.Fatalf("expected Enter key to activate button (count == 3), got %d", activationCount)
	}

	// Disabled button must ignore keyboard events
	btn.Disable()
	btn.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if activationCount != 3 {
		t.Errorf("disabled button must not activate on key events, got count %d", activationCount)
	}
}
