//go:build linux || darwin

package main

import (
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"

	"github.com/scallister/call-detect/internal/appdir"
)

var lockFile *os.File

func singletonHeld() bool {
	dir, err := appdir.Ensure()
	if err != nil {
		return false
	}
	f, err := os.OpenFile(filepath.Join(dir, "lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return false
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return true
	}
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	_ = f.Sync()
	lockFile = f
	return false
}

func watchRemoteQuit(func()) {}
