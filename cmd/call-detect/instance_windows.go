//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const quitEventName = `Local\call-detect-quit`

var (
	instanceMutex windows.Handle
	quitEvent     windows.Handle
)

func singletonHeld() bool {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	createMutex := kernel32.NewProc("CreateMutexW")
	getLastError := kernel32.NewProc("GetLastError")
	name, err := windows.UTF16PtrFromString("Local\\call-detect-singleton")
	if err != nil {
		return false
	}
	r, _, _ := createMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return false
	}
	instanceMutex = windows.Handle(r)
	const errorAlreadyExists = 183
	code, _, _ := getLastError.Call()
	return code == errorAlreadyExists
}

func watchRemoteQuit(onQuit func()) {
	if onQuit == nil {
		return
	}
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	createEvent := kernel32.NewProc("CreateEventW")
	waitFor := kernel32.NewProc("WaitForSingleObject")
	name, err := windows.UTF16PtrFromString(quitEventName)
	if err != nil {
		return
	}
	h, _, _ := createEvent.Call(0, 0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return
	}
	quitEvent = windows.Handle(h)
	go func() {
		_, _, _ = waitFor.Call(h, 0xFFFFFFFF)
		onQuit()
	}()
}
