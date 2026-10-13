//go:build windows

package system

import (
	"os"
	"syscall"
)

func redirectStderr(f *os.File) {
	_ = syscall.SetStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	_ = syscall.SetStdHandle(syscall.STD_OUTPUT_HANDLE, syscall.Handle(f.Fd()))
}
