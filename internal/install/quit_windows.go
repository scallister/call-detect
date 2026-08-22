//go:build windows

package install

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	quitEventName    = `Local\call-detect-quit`
	eventModifyState = 0x0002
)

// SignalQuit asks a running installed tray instance to exit.
// Only the installed copy listens for this event, so Install from a
// downloaded exe does not quit itself.
func SignalQuit() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	openEvent := kernel32.NewProc("OpenEventW")
	setEvent := kernel32.NewProc("SetEvent")
	name, err := windows.UTF16PtrFromString(quitEventName)
	if err != nil {
		return
	}
	h, _, _ := openEvent.Call(eventModifyState, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return
	}
	_, _, _ = setEvent.Call(h)
	_ = windows.CloseHandle(windows.Handle(h))
}
