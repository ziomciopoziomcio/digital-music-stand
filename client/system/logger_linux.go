//go:build !windows

package system

import (
	"os"
	"syscall"
)

func redirectStderr(f *os.File) {
	_ = syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
	_ = syscall.Dup2(int(f.Fd()), int(os.Stdout.Fd()))
}
