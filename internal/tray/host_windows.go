//go:build windows

package tray

import (
	"log"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/scallister/call-detect/internal/state"
)

const (
	wmTray   = 0x0400 + 1
	wmUpdate = 0x0400 + 2
	idQuit   = 1001

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nimAdd     = 0x00000000
	nimModify  = 0x00000001
	nimDelete  = 0x00000002

	wmDestroy       = 0x0002
	wmCommand       = 0x0111
	wmRButtonUp     = 0x0205
	wmContextMenu   = 0x007B
	wmLButtonDblClk = 0x0203

	wsOverlapped   = 0x00000000
	mfString       = 0x00000000
	mfSeparator    = 0x00000800
	mfGrayed       = 0x00000001
	mfDisabled     = 0x00000002
	tpmRightBtn    = 0x0002
	tpmBottomAlign = 0x0020
	tpmLeftAlign   = 0x0000
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassExW       = user32.NewProc("RegisterClassExW")
	procCreateWindowExW        = user32.NewProc("CreateWindowExW")
	procDefWindowProcW         = user32.NewProc("DefWindowProcW")
	procGetMessageW            = user32.NewProc("GetMessageW")
	procTranslateMessage       = user32.NewProc("TranslateMessage")
	procDispatchMessageW       = user32.NewProc("DispatchMessageW")
	procPostMessageW           = user32.NewProc("PostMessageW")
	procPostQuitMessage        = user32.NewProc("PostQuitMessage")
	procDestroyWindow          = user32.NewProc("DestroyWindow")
	procSetForegroundWindow    = user32.NewProc("SetForegroundWindow")
	procGetCursorPos           = user32.NewProc("GetCursorPos")
	procCreatePopupMenu        = user32.NewProc("CreatePopupMenu")
	procAppendMenuW            = user32.NewProc("AppendMenuW")
	procTrackPopupMenu         = user32.NewProc("TrackPopupMenu")
	procDestroyMenu            = user32.NewProc("DestroyMenu")
	procDestroyIcon            = user32.NewProc("DestroyIcon")
	procCreateIconFromResource = user32.NewProc("CreateIconFromResourceEx")
	procGetModuleHandleW       = kernel32.NewProc("GetModuleHandleW")
	procShellNotifyIconW       = shell32.NewProc("Shell_NotifyIconW")
)

type notifyIconData struct {
	Size            uint32
	Wnd             windows.HWND
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            windows.Handle
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Timeout         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GUIDItem        windows.GUID
	BalloonIcon     windows.Handle
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type point struct {
	X, Y int32
}

type msg struct {
	Wnd     windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type hostImpl struct {
	mu       sync.Mutex
	hwnd     windows.HWND
	idleIcon windows.Handle
	busyIcon windows.Handle
	snap     state.Snapshot
}

func newHostImpl() hostImpl {
	return hostImpl{snap: SnapshotIdle()}
}

func (h *hostImpl) update(s state.Snapshot) {
	h.mu.Lock()
	h.snap = s
	hwnd := h.hwnd
	h.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(uintptr(hwnd), wmUpdate, 0, 0)
	}
}

func (h *hostImpl) quit() {
	h.mu.Lock()
	hwnd := h.hwnd
	h.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(uintptr(hwnd), wmDestroy, 0, 0)
	}
}

func (h *hostImpl) run(ready func()) {
	idle, err := iconFromICO(IdleICO())
	if err != nil {
		log.Printf("tray idle icon: %v", err)
		return
	}
	busy, err := iconFromICO(BusyICO())
	if err != nil {
		log.Printf("tray busy icon: %v", err)
		return
	}
	h.idleIcon = idle
	h.busyIcon = busy
	defer destroyIcon(idle)
	defer destroyIcon(busy)

	instance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := windows.UTF16PtrFromString("call-detect-tray")
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   windows.NewCallback(h.wndProc),
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		log.Printf("tray register class: %v", err)
		return
	}

	title, _ := windows.UTF16PtrFromString("call-detect")
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsOverlapped,
		0, 0, 0, 0,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		log.Printf("tray create window: %v", err)
		return
	}
	h.mu.Lock()
	h.hwnd = windows.HWND(hwnd)
	h.mu.Unlock()

	if err := h.notify(nimAdd); err != nil {
		log.Printf("tray add icon: %v", err)
		return
	}
	defer func() { _ = h.notify(nimDelete) }()

	if ready != nil {
		go ready()
	}

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (h *hostImpl) wndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmUpdate:
		if err := h.notify(nimModify); err != nil {
			log.Printf("tray update: %v", err)
		}
		return 0
	case wmTray:
		switch lParam {
		case wmRButtonUp, wmContextMenu, wmLButtonDblClk:
			h.showMenu(hwnd)
		}
		return 0
	case wmCommand:
		if uint16(wParam) == idQuit {
			_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		}
		return 0
	case wmDestroy:
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func (h *hostImpl) notify(action uint32) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var nid notifyIconData
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = h.hwnd
	nid.ID = 1
	nid.Flags = nifMessage | nifIcon | nifTip
	nid.CallbackMessage = wmTray
	if h.snap.Busy {
		nid.Icon = h.busyIcon
	} else {
		nid.Icon = h.idleIcon
	}
	setTip(&nid, Tooltip(h.snap))
	r, _, err := procShellNotifyIconW.Call(uintptr(action), uintptr(unsafe.Pointer(&nid)))
	if r == 0 {
		return err
	}
	return nil
}

func setTip(nid *notifyIconData, tip string) {
	u, err := windows.UTF16FromString(tip)
	if err != nil {
		return
	}
	n := len(u)
	if n > len(nid.Tip) {
		n = len(nid.Tip)
		u[n-1] = 0
	}
	copy(nid.Tip[:], u[:n])
}

func (h *hostImpl) showMenu(hwnd windows.HWND) {
	h.mu.Lock()
	s := h.snap
	h.mu.Unlock()

	menu, _, err := procCreatePopupMenu.Call()
	if menu == 0 {
		log.Printf("tray menu: %v", err)
		return
	}
	defer procDestroyMenu.Call(menu)

	appendGray(menu, statusLine(s))
	appendGray(menu, boolLine("Microphone", s.Microphone))
	appendGray(menu, boolLine("Webcam", s.Webcam))
	if len(s.Sources) > 0 {
		appendGray(menu, "Sources: "+joinSources(s.Sources))
	} else {
		appendGray(menu, "Sources: (none)")
	}
	_, _, _ = procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	quit, _ := windows.UTF16PtrFromString("Quit")
	_, _, _ = procAppendMenuW.Call(menu, mfString, idQuit, uintptr(unsafe.Pointer(quit)))

	var pt point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
	_, _, _ = procTrackPopupMenu.Call(menu, tpmRightBtn|tpmBottomAlign|tpmLeftAlign, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hwnd), 0)
	_, _, _ = procPostMessageW.Call(uintptr(hwnd), 0, 0, 0)
}

func appendGray(menu uintptr, text string) {
	p, _ := windows.UTF16PtrFromString(text)
	_, _, _ = procAppendMenuW.Call(menu, mfString|mfGrayed|mfDisabled, 0, uintptr(unsafe.Pointer(p)))
}

func statusLine(s state.Snapshot) string {
	if s.Busy {
		return "On a call"
	}
	return "Idle"
}

func boolLine(name string, on bool) string {
	if on {
		return name + ": yes"
	}
	return name + ": no"
}

func joinSources(src []string) string {
	out := ""
	for i, s := range src {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

func iconFromICO(ico []byte) (windows.Handle, error) {
	if len(ico) < 22 {
		return 0, syscall.EINVAL
	}
	offset := binaryLE32(ico[18:22])
	size := binaryLE32(ico[14:18])
	if int(offset+size) > len(ico) {
		return 0, syscall.EINVAL
	}
	img := ico[offset : offset+size]
	const version = 0x00030000
	r, _, err := procCreateIconFromResource.Call(
		uintptr(unsafe.Pointer(&img[0])),
		uintptr(len(img)),
		1,
		version,
		0,
		0,
		0,
	)
	if r == 0 {
		return 0, err
	}
	return windows.Handle(r), nil
}

func binaryLE32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func destroyIcon(h windows.Handle) {
	if h != 0 {
		_, _, _ = procDestroyIcon.Call(uintptr(h))
	}
}
