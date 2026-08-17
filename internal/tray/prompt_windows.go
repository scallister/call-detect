//go:build windows

package tray

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wsVisible       = 0x10000000
	wsChild         = 0x40000000
	wsTabstop       = 0x00010000
	wsBorder        = 0x00800000
	wsCaption       = 0x00C00000
	wsSysMenu       = 0x00080000
	wsPopup         = 0x80000000
	esAutoHScroll   = 0x0080
	bsPushButton    = 0x00000000
	bsDefPushButton = 0x00000001
	idOK            = 1
	idCancel        = 2
	wmSetFont       = 0x0030
	wmGetTextLength = 0x000E
	wmClose         = 0x0010
	defaultGUIFont  = 17
)

var (
	procIsDialogMessageW = user32.NewProc("IsDialogMessageW")
	procGetWindowTextW   = user32.NewProc("GetWindowTextW")
	procSetFocus         = user32.NewProc("SetFocus")
	procGetStockObject   = windows.NewLazySystemDLL("gdi32.dll").NewProc("GetStockObject")
	procSendMessageW     = user32.NewProc("SendMessageW")
	promptMu             sync.Mutex
	promptClassOnce      sync.Once
	promptWndProcFn      uintptr
)

type promptResult struct {
	edit  windows.HWND
	ok    bool
	value string
	done  bool
}

func promptText(owner windows.HWND, title, label, value string) (string, bool) {
	promptMu.Lock()
	defer promptMu.Unlock()

	instance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := windows.UTF16PtrFromString("call-detect-prompt")
	promptClassOnce.Do(func() {
		promptWndProcFn = windows.NewCallback(promptWndProc)
		wc := wndClassEx{
			Size:      uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:   promptWndProcFn,
			Instance:  windows.Handle(instance),
			ClassName: className,
		}
		_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	state := &promptResult{}
	titleU, _ := windows.UTF16PtrFromString(title)
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titleU)),
		wsPopup|wsCaption|wsSysMenu|wsVisible,
		200, 200, 520, 200,
		uintptr(owner), 0, instance, 0,
	)
	if hwnd == 0 {
		return "", false
	}

	font, _, _ := procGetStockObject.Call(defaultGUIFont)
	createChild := func(class, text string, style, x, y, w, h, id uintptr) windows.HWND {
		cls, _ := windows.UTF16PtrFromString(class)
		txt, _ := windows.UTF16PtrFromString(text)
		ch, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(txt)),
			wsChild|wsVisible|style,
			x, y, w, h,
			hwnd, id, instance, 0,
		)
		if font != 0 && ch != 0 {
			_, _, _ = procSendMessageW.Call(ch, wmSetFont, font, 1)
		}
		return windows.HWND(ch)
	}

	createChild("STATIC", label, 0, 16, 16, 470, 40, 0)
	state.edit = createChild("EDIT", value, wsTabstop|wsBorder|esAutoHScroll, 16, 64, 470, 24, 10)
	createChild("BUTTON", "OK", wsTabstop|bsDefPushButton, 300, 110, 88, 28, idOK)
	createChild("BUTTON", "Cancel", wsTabstop|bsPushButton, 398, 110, 88, 28, idCancel)
	if state.edit != 0 {
		_, _, _ = procSetFocus.Call(uintptr(state.edit))
	}

	setPrompt(windows.HWND(hwnd), state)

	var m msg
	for !state.done {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		ir, _, _ := procIsDialogMessageW.Call(hwnd, uintptr(unsafe.Pointer(&m)))
		if ir == 0 {
			_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	return state.value, state.ok
}

var prompts sync.Map

func setPrompt(hwnd windows.HWND, p *promptResult) {
	prompts.Store(hwnd, p)
}

func getPrompt(hwnd windows.HWND) *promptResult {
	v, ok := prompts.Load(hwnd)
	if !ok {
		return nil
	}
	return v.(*promptResult)
}

func promptWndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCommand:
		id := uint16(wParam)
		p := getPrompt(hwnd)
		if p == nil {
			break
		}
		if id == idOK {
			p.value = windowText(p.edit)
			p.ok = true
			p.done = true
			_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
			return 0
		}
		if id == idCancel {
			p.done = true
			_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
			return 0
		}
	case wmClose:
		if p := getPrompt(hwnd); p != nil {
			p.done = true
		}
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		return 0
	case wmDestroy:
		prompts.Delete(hwnd)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func windowText(hwnd windows.HWND) string {
	n, _, _ := procSendMessageW.Call(uintptr(hwnd), wmGetTextLength, 0, 0)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	_, _, _ = procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), n+1)
	return windows.UTF16ToString(buf)
}
