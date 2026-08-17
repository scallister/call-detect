//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var instanceMutex windows.Handle

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
