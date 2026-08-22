//go:build windows

package tray

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/scallister/call-detect/internal/state"
)

const (
	wmTray      = 0x0400 + 1
	wmUpdate    = 0x0400 + 2
	idQuit      = 1001
	idInstall   = 1002
	idUninstall = 1003
	idWebhook   = 1004
	idGitHub    = 1005
	idUpdate    = 1006

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	nimAdd     = 0x00000000
	nimModify  = 0x00000001
	nimDelete  = 0x00000002

	wmDestroy         = 0x0002
	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016
	wmCommand         = 0x0111
	wmRButtonUp       = 0x0205
	wmContextMenu     = 0x007B
	wmLButtonDblClk   = 0x0203

	wsOverlapped   = 0x00000000
	mfString       = 0x00000000
	mfSeparator    = 0x00000800
	mfGrayed       = 0x00000001
	mfDisabled     = 0x00000002
	tpmRightBtn    = 0x0002
	tpmBottomAlign = 0x0020
	tpmLeftAlign   = 0x0000
	tpmReturnCmd   = 0x0100
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterWindowMessageW = user32.NewProc("RegisterWindowMessageW")
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
	procShellExecuteW          = shell32.NewProc("ShellExecuteW")
	procMessageBoxW            = user32.NewProc("MessageBoxW")
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
	mu             sync.Mutex
	hwnd           windows.HWND
	idleIcon       windows.Handle
	busyIcon       windows.Handle
	errorIcon      windows.Handle
	snap           state.Snapshot
	webhookFailed  bool
	done           chan struct{}
	once           sync.Once
	actions        Actions
	taskbarCreated uint32
}

func newHostImpl() hostImpl {
	return hostImpl{snap: SnapshotIdle(), done: make(chan struct{})}
}

func (h *hostImpl) setActions(a Actions) {
	h.mu.Lock()
	h.actions = a
	h.mu.Unlock()
}

func (h *hostImpl) update(s state.Snapshot) {
	h.mu.Lock()
	h.snap = s
	hwnd := h.hwnd
	h.mu.Unlock()
	h.postUpdate(hwnd)
}

func (h *hostImpl) setWebhookFailed(failed bool) {
	h.mu.Lock()
	h.webhookFailed = failed
	hwnd := h.hwnd
	h.mu.Unlock()
	h.postUpdate(hwnd)
}

func (h *hostImpl) postUpdate(hwnd windows.HWND) {
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(uintptr(hwnd), wmUpdate, 0, 0)
	}
}

func (h *hostImpl) quit() {
	RunExitHook()
	h.once.Do(func() { close(h.done) })
	h.mu.Lock()
	hwnd := h.hwnd
	h.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(uintptr(hwnd), wmDestroy, 0, 0)
	}
}

func (h *hostImpl) run(ready func()) {
	runtime.LockOSThread()
	if err := h.setupTray(); err != nil {
		log.Printf("tray setup failed: %v; continuing without icon", err)
		startReady(ready)
		<-h.done
		return
	}
	defer h.teardownTray()
	startReady(ready)
	h.messageLoop()
}

func (h *hostImpl) setupTray() error {
	idle, err := iconFromICO(IdleICO())
	if err != nil {
		return fmt.Errorf("idle icon: %w", err)
	}
	busy, err := iconFromICO(BusyICO())
	if err != nil {
		destroyIcon(idle)
		return fmt.Errorf("busy icon: %w", err)
	}
	fail, err := iconFromICO(ErrorICO())
	if err != nil {
		destroyIcon(idle)
		destroyIcon(busy)
		return fmt.Errorf("error icon: %w", err)
	}
	h.idleIcon = idle
	h.busyIcon = busy
	h.errorIcon = fail
	h.taskbarCreated = registerWindowMessage("TaskbarCreated")

	instance, _, _ := procGetModuleHandleW.Call(0)
	className, _ := windows.UTF16PtrFromString("call-detect-tray")
	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   windows.NewCallback(h.wndProc),
		Instance:  windows.Handle(instance),
		ClassName: className,
	}
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		h.teardownTray()
		return fmt.Errorf("register class: %w", err)
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
		h.teardownTray()
		return fmt.Errorf("create window: %w", err)
	}
	h.mu.Lock()
	h.hwnd = windows.HWND(hwnd)
	h.mu.Unlock()

	if err := h.notify(nimAdd); err != nil {
		h.teardownTray()
		return fmt.Errorf("add icon: %w", err)
	}
	return nil
}

func (h *hostImpl) teardownTray() {
	_ = h.notify(nimDelete)
	h.mu.Lock()
	hwnd := h.hwnd
	idle, busy, fail := h.idleIcon, h.busyIcon, h.errorIcon
	h.hwnd = 0
	h.idleIcon = 0
	h.busyIcon = 0
	h.errorIcon = 0
	h.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
	}
	destroyIcon(idle)
	destroyIcon(busy)
	destroyIcon(fail)
}

func (h *hostImpl) messageLoop() {
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (h *hostImpl) wndProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if h.taskbarCreated != 0 && msg == h.taskbarCreated {
		if err := h.notify(nimAdd); err != nil {
			log.Printf("tray restore: %v", err)
		}
		return 0
	}
	switch msg {
	case wmUpdate:
		if err := h.notify(nimModify); err != nil {
			if err2 := h.notify(nimAdd); err2 != nil {
				log.Printf("tray update: %v", err)
			}
		}
		return 0
	case wmTray:
		switch lParam {
		case wmRButtonUp, wmContextMenu, wmLButtonDblClk:
			h.showMenu(hwnd)
		}
		return 0
	case wmQueryEndSession, wmEndSession:
		RunExitHook()
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		return 1
	case wmCommand:
		h.handleCommand(uint16(wParam), hwnd)
		return 0
	case wmDestroy:
		RunExitHook()
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
	switch {
	case h.webhookFailed:
		nid.Icon = h.errorIcon
	case h.snap.Busy:
		nid.Icon = h.busyIcon
	default:
		nid.Icon = h.idleIcon
	}
	setTip(&nid, TooltipAlert(h.snap, h.webhookFailed))
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
	webhookFailed := h.webhookFailed
	h.mu.Unlock()

	menu, _, err := procCreatePopupMenu.Call()
	if menu == 0 {
		log.Printf("tray menu: %v", err)
		return
	}
	defer procDestroyMenu.Call(menu)

	for _, line := range statusLines(s, webhookFailed) {
		appendGray(menu, line)
	}
	h.mu.Lock()
	actions := h.actions
	h.mu.Unlock()
	_, _, _ = procAppendMenuW.Call(menu, mfSeparator, 0, 0)
	for _, c := range actionChoices(actions) {
		appendItem(menu, winMenuID(c.ID), c.Label)
	}

	var pt point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	_, _, _ = procSetForegroundWindow.Call(uintptr(hwnd))
	r, _, _ := procTrackPopupMenu.Call(menu, tpmRightBtn|tpmBottomAlign|tpmLeftAlign|tpmReturnCmd, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hwnd), 0)
	_, _, _ = procPostMessageW.Call(uintptr(hwnd), 0, 0, 0)
	if r != 0 {
		h.handleCommand(uint16(r), hwnd)
	}
}

func appendItem(menu uintptr, id uint16, text string) {
	p, _ := windows.UTF16PtrFromString(text)
	_, _, _ = procAppendMenuW.Call(menu, mfString, uintptr(id), uintptr(unsafe.Pointer(p)))
}

func (h *hostImpl) handleCommand(id uint16, hwnd windows.HWND) {
	h.mu.Lock()
	actions := h.actions
	h.mu.Unlock()
	switch id {
	case idQuit:
		_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
	case idWebhook:
		if actions.SetWebhookURL == nil {
			return
		}
		cur := ""
		if actions.WebhookURL != nil {
			cur = actions.WebhookURL()
		}
		url, ok := promptText(hwnd, "Webhook URL", "Home Assistant webhook or other POST URL.\nLeave empty to disable.", cur, state.ExampleJSON)
		if !ok {
			return
		}
		if err := actions.SetWebhookURL(url); err != nil {
			alert(hwnd, err.Error(), true)
			return
		}
		if url == "" {
			alert(hwnd, "Webhook disabled.", false)
			return
		}
		alert(hwnd, "Webhook URL saved. Changes apply immediately.", false)
	default:
		handleMenuID(actions, menuIDFromWin(id), func() {
			_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
		})
	}
}

func winMenuID(id string) uint16 {
	switch id {
	case menuInstall:
		return idInstall
	case menuUninstall:
		return idUninstall
	case menuWebhook:
		return idWebhook
	case menuUpdate:
		return idUpdate
	case menuGitHub:
		return idGitHub
	case menuQuit:
		return idQuit
	default:
		return 0
	}
}

func menuIDFromWin(id uint16) string {
	switch id {
	case idInstall:
		return menuInstall
	case idUninstall:
		return menuUninstall
	case idWebhook:
		return menuWebhook
	case idUpdate:
		return menuUpdate
	case idGitHub:
		return menuGitHub
	case idQuit:
		return menuQuit
	default:
		return ""
	}
}

func openURL(url string) error {
	const swShowNormal = 1
	op, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	r, _, callErr := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(op)), uintptr(unsafe.Pointer(file)), 0, 0, swShowNormal)
	// ShellExecute returns a value > 32 on success.
	if r <= 32 {
		if callErr != nil {
			return fmt.Errorf("open %s: %w", url, callErr)
		}
		return fmt.Errorf("open %s: %d", url, r)
	}
	return nil
}

// Confirm asks a yes/no question. Yes is true.
func Confirm(title, text string) bool {
	const mbYesNo = 0x00000004
	const mbIconQuestion = 0x00000020
	const mbSetForeground = 0x00010000
	const idYes = 6
	t, err1 := windows.UTF16PtrFromString(title)
	b, err2 := windows.UTF16PtrFromString(text)
	if err1 != nil || err2 != nil {
		return false
	}
	r, _, _ := procMessageBoxW.Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), mbYesNo|mbIconQuestion|mbSetForeground)
	return r == idYes
}

// Alert shows a modal message.
func Alert(text string, isErr bool) {
	alert(0, text, isErr)
}

func alert(owner windows.HWND, text string, isErr bool) {
	const mbOK = 0x00000000
	const mbIconError = 0x00000010
	const mbIconInfo = 0x00000040
	flags := uintptr(mbOK | mbIconInfo)
	if isErr {
		flags = uintptr(mbOK | mbIconError)
	}
	title, _ := windows.UTF16PtrFromString("call-detect")
	body, _ := windows.UTF16PtrFromString(text)
	_, _, _ = procMessageBoxW.Call(uintptr(owner), uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), flags)
}

func appendGray(menu uintptr, text string) {
	p, _ := windows.UTF16PtrFromString(text)
	_, _, _ = procAppendMenuW.Call(menu, mfString|mfGrayed|mfDisabled, 0, uintptr(unsafe.Pointer(p)))
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

func registerWindowMessage(name string) uint32 {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(p)))
	return uint32(r)
}

func destroyIcon(h windows.Handle) {
	if h != 0 {
		_, _, _ = procDestroyIcon.Call(uintptr(h))
	}
}
