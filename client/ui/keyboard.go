package ui

import (
	"os/exec"
	"runtime"

	"fyne.io/fyne/v2/widget"
)

var keyboardCmd *exec.Cmd

func ShowKeyboard() {
	if keyboardCmd != nil && keyboardCmd.Process != nil {
		return
	}
	switch runtime.GOOS {
	case "windows":
		keyboardCmd = exec.Command("osk")
	case "linux":
		if _, err := exec.LookPath("matchbox-keyboard"); err != nil {
			return
		}
		keyboardCmd = exec.Command("matchbox-keyboard")
	default:
		return
	}
	_ = keyboardCmd.Start()
}

func HideKeyboard() {
	if keyboardCmd != nil && keyboardCmd.Process != nil {
		if runtime.GOOS == "windows" {
			_ = exec.Command("taskkill", "/IM", "osk.exe", "/F").Run()
		} else {
			_ = keyboardCmd.Process.Kill()
		}
		_ = keyboardCmd.Process.Kill()
		keyboardCmd = nil
	}
}

type AutoKeyboardEntry struct {
	widget.Entry
}

func NewAutoKeyboardEntry() *AutoKeyboardEntry {
	e := &AutoKeyboardEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *AutoKeyboardEntry) FocusGained() {
	e.Entry.FocusGained()
	ShowKeyboard()
}

func (e *AutoKeyboardEntry) FocusLost() {
	e.Entry.FocusLost()
	HideKeyboard()
}

type AutoKeyboardPasswordEntry struct {
	widget.Entry
}

func NewAutoKeyboardPasswordEntry() *AutoKeyboardPasswordEntry {
	e := &AutoKeyboardPasswordEntry{}
	e.Password = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *AutoKeyboardPasswordEntry) FocusGained() {
	e.Entry.FocusGained()
	ShowKeyboard()
}

func (e *AutoKeyboardPasswordEntry) FocusLost() {
	e.Entry.FocusLost()
	HideKeyboard()
}
