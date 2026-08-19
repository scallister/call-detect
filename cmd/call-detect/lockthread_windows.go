//go:build windows

package main

import "runtime"

func init() {
	// Win32 tray windows and GetMessage must stay on the thread that
	// created the HWND. Without this, Go can move the goroutine after a
	// syscall and the icon stops responding until the process is restarted.
	runtime.LockOSThread()
}
