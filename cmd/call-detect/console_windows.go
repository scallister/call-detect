//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func enableConsole(alloc bool) {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	attach := kernel32.NewProc("AttachConsole")
	allocConsole := kernel32.NewProc("AllocConsole")

	const attachParent = ^uintptr(0)
	r, _, _ := attach.Call(attachParent)
	if r == 0 && alloc {
		_, _, _ = allocConsole.Call()
	}

	if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 {
		os.Stdout = os.NewFile(uintptr(h), "stdout")
	}
	if h, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE); err == nil && h != 0 {
		os.Stderr = os.NewFile(uintptr(h), "stderr")
	}
	if h, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE); err == nil && h != 0 {
		os.Stdin = os.NewFile(uintptr(h), "stdin")
	}
}
