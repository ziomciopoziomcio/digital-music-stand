//go:build windows

package syswin

import (
	"log"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procCallWindowProcW = user32.NewProc("CallWindowProcW")
	procDefWindowProcW  = user32.NewProc("DefWindowProcW")
	procSetPropW        = user32.NewProc("SetPropW")

	procSetWindowLongPtrW = func() *syscall.LazyProc {
		p := user32.NewProc("SetWindowLongPtrW")
		if err := p.Find(); err != nil {
			return user32.NewProc("SetWindowLongW")
		}
		return p
	}()
)

const (
	gwlpWndProc = ^uintptr(3) // -4 in 2's complement (GWLP_WNDPROC)

	wmDestroy                        = 0x0002
	wmNcDestroy                      = 0x0082
	wmMouseMove                      = 0x0200
	wmLButtonDown                    = 0x0201
	wmLButtonUp                      = 0x0202
	wmTabletQuerySystemGestureStatus = 0x02CC

	tabletDisablePressAndHold = 0x00000001
	tabletDisableFlicks       = 0x00010000
)

var (
	hookLock     sync.RWMutex
	origWndProcs = make(map[uintptr]uintptr)
	hookCallback uintptr
)

// HookWindowsTouch registers the touchscreen hook on the native Win32 window backing win.
func HookWindowsTouch(win fyne.Window) {
	if win == nil {
		return
	}

	hookWindow(win)

	// Start a short background poll to ensure the window is hooked once the native HWND is ready
	go func() {
		for i := 0; i < 30; i++ {
			time.Sleep(100 * time.Millisecond)
			if hookWindow(win) {
				return
			}
		}
	}()
}

func hookWindow(win fyne.Window) bool {
	nw, ok := win.(driver.NativeWindow)
	if !ok {
		return false
	}

	var hwnd uintptr
	nw.RunNative(func(ctx any) {
		if winCtx, ok := ctx.(driver.WindowsWindowContext); ok {
			hwnd = winCtx.HWND
		}
	})

	if hwnd == 0 {
		return false
	}

	return HookWindowsTouchHWND(hwnd)
}

// HookWindowsTouchHWND subclasses a specific Win32 HWND to synchronize touch events with GLFW cursor tracking.
func HookWindowsTouchHWND(hwnd uintptr) bool {
	hookLock.Lock()
	defer hookLock.Unlock()

	if _, exists := origWndProcs[hwnd]; exists {
		return true // already hooked
	}

	if hookCallback == 0 {
		hookCallback = syscall.NewCallback(touchWndProc)
	}

	// Disable Windows tablet pen/touch press-and-hold delay and flicks for this window
	propName, _ := syscall.UTF16PtrFromString("MicrosoftTabletPenServiceProperty")
	procSetPropW.Call(hwnd, uintptr(unsafe.Pointer(propName)), tabletDisablePressAndHold|tabletDisableFlicks)

	orig, _, _ := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, hookCallback)
	if orig != 0 {
		origWndProcs[hwnd] = orig
		log.Printf("[Touch] Installed Windows touchscreen interaction hook on HWND 0x%X", hwnd)
		return true
	}

	return false
}

// touchWndProc is the subclass WindowProc that synchronizes touch input coordinates with GLFW.
func touchWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	hookLock.RLock()
	orig := origWndProcs[hwnd]
	hookLock.RUnlock()

	if orig == 0 {
		ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case wmTabletQuerySystemGestureStatus:
		// Return TABLET_DISABLE_PRESSANDHOLD | TABLET_DISABLE_FLICKS.
		// This eliminates the OS-level hold-for-right-click delay, making touches respond immediately.
		return tabletDisablePressAndHold | tabletDisableFlicks

	case wmLButtonDown:
		// Windows touch emulation sends WM_LBUTTONDOWN directly with the touch coordinates in lParam,
		// without emitting an initial WM_MOUSEMOVE.
		// GLFW on Windows only calls _glfwInputCursorPos (which updates Fyne's pointer position) upon WM_MOUSEMOVE.
		// Without synthesizing WM_MOUSEMOVE here, Fyne processes the press event at its stale cursor position
		// (which defaults to (0,0) or the prior tap location). This sets w.mousePressed to whatever sits at (0,0)
		// (e.g. the Back button), while the release event occurs at the new position, causing Fyne to drop the tap
		// (co != w.mousePressed).
		// By forwarding WM_MOUSEMOVE first with the exact coordinates from lParam, GLFW and Fyne update their
		// cursor position BEFORE Fyne handles the press event.
		procCallWindowProcW.Call(orig, hwnd, wmMouseMove, wParam, lParam)
		ret, _, _ := procCallWindowProcW.Call(orig, hwnd, uintptr(msg), wParam, lParam)
		return ret

	case wmLButtonUp:
		// Forward WM_MOUSEMOVE with the release coordinates to keep pointer tracking consistent on release.
		procCallWindowProcW.Call(orig, hwnd, wmMouseMove, wParam, lParam)
		ret, _, _ := procCallWindowProcW.Call(orig, hwnd, uintptr(msg), wParam, lParam)
		return ret

	case wmDestroy, wmNcDestroy:
		hookLock.Lock()
		delete(origWndProcs, hwnd)
		hookLock.Unlock()
	}

	ret, _, _ := procCallWindowProcW.Call(orig, hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// ProcessHookMessage allows testing of the message translation logic without a real window.
func ProcessHookMessage(msg uint32) (handled bool, preMouseMoveSent bool, result uintptr) {
	switch msg {
	case wmTabletQuerySystemGestureStatus:
		return true, false, tabletDisablePressAndHold | tabletDisableFlicks
	case wmLButtonDown, wmLButtonUp:
		return true, true, 0
	default:
		return false, false, 0
	}
}
