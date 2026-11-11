//go:build windows

package syswin

import (
	"testing"
)

func TestProcessHookMessage_TabletQuery(t *testing.T) {
	handled, preMove, result := ProcessHookMessage(wmTabletQuerySystemGestureStatus)
	if !handled {
		t.Errorf("expected wmTabletQuerySystemGestureStatus to be handled")
	}
	if preMove {
		t.Errorf("wmTabletQuerySystemGestureStatus should not synthesize mouse move")
	}
	expected := uintptr(tabletDisablePressAndHold | tabletDisableFlicks)
	if result != expected {
		t.Errorf("expected result 0x%X, got 0x%X", expected, result)
	}
}

func TestProcessHookMessage_LButtonDown(t *testing.T) {
	handled, preMove, _ := ProcessHookMessage(wmLButtonDown)
	if !handled {
		t.Errorf("expected wmLButtonDown to be handled")
	}
	if !preMove {
		t.Errorf("expected wmLButtonDown to trigger pre-dispatch of WM_MOUSEMOVE")
	}
}

func TestProcessHookMessage_LButtonUp(t *testing.T) {
	handled, preMove, _ := ProcessHookMessage(wmLButtonUp)
	if !handled {
		t.Errorf("expected wmLButtonUp to be handled")
	}
	if !preMove {
		t.Errorf("expected wmLButtonUp to trigger pre-dispatch of WM_MOUSEMOVE")
	}
}

func TestProcessHookMessage_OtherMessages(t *testing.T) {
	handled, preMove, _ := ProcessHookMessage(0x000F) // WM_PAINT
	if handled || preMove {
		t.Errorf("unrelated message should not be handled specially by hook")
	}
}
