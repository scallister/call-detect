//go:build darwin

package main

import "runtime"

func init() {
	// AppKit status items and the NSApplication run loop must stay on
	// the thread that created them.
	runtime.LockOSThread()
}
