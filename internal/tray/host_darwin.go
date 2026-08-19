//go:build darwin

package tray

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/scallister/call-detect/internal/state"
)

const (
	nsVariableStatusItemLength             = -1.0
	nsApplicationActivationPolicyAccessory = 1

	tagInstall   = 1
	tagUninstall = 2
	tagWebhook   = 3
	tagGitHub    = 4
	tagQuit      = 5
)

type nsSize struct {
	Width, Height float64
}

var (
	selSharedApplication   = objc.RegisterName("sharedApplication")
	selSetActivationPolicy = objc.RegisterName("setActivationPolicy:")
	selFinishLaunching     = objc.RegisterName("finishLaunching")
	selRun                 = objc.RegisterName("run")
	selStop                = objc.RegisterName("stop:")
	selPerformMain         = objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:")
	selSystemStatusBar     = objc.RegisterName("systemStatusBar")
	selStatusItemWithLen   = objc.RegisterName("statusItemWithLength:")
	selRemoveStatusItem    = objc.RegisterName("removeStatusItem:")
	selButton              = objc.RegisterName("button")
	selSetImage            = objc.RegisterName("setImage:")
	selSetToolTip          = objc.RegisterName("setToolTip:")
	selSetMenu             = objc.RegisterName("setMenu:")
	selAlloc               = objc.RegisterName("alloc")
	selInitWithData        = objc.RegisterName("initWithData:")
	selSetSize             = objc.RegisterName("setSize:")
	selSetTemplate         = objc.RegisterName("setTemplate:")
	selDataWithBytes       = objc.RegisterName("dataWithBytes:length:")
	selStringUTF8          = objc.RegisterName("stringWithUTF8String:")
	selInitWithTitle       = objc.RegisterName("initWithTitle:")
	selInitMenuItem        = objc.RegisterName("initWithTitle:action:keyEquivalent:")
	selAddItem             = objc.RegisterName("addItem:")
	selRemoveAllItems      = objc.RegisterName("removeAllItems")
	selSeparatorItem       = objc.RegisterName("separatorItem")
	selSetTarget           = objc.RegisterName("setTarget:")
	selSetTag              = objc.RegisterName("setTag:")
	selSetEnabled          = objc.RegisterName("setEnabled:")
	selTag                 = objc.RegisterName("tag")
	selRetain              = objc.RegisterName("retain")
	selRelease             = objc.RegisterName("release")
	selMenuClick           = objc.RegisterName("menuClick:")
	selApplyUI             = objc.RegisterName("applyUI")

	darwinActive   atomic.Pointer[hostImpl]
	darwinClass    objc.Class
	darwinClassErr error
	darwinOnce     sync.Once
)

type hostImpl struct {
	mu            sync.Mutex
	snap          state.Snapshot
	webhookFailed bool
	actions       Actions
	app           objc.ID
	item          objc.ID
	menu          objc.ID
	target        objc.ID
	bar           objc.ID
	done          chan struct{}
	once          sync.Once
	ready         bool
}

func newHostImpl() hostImpl {
	return hostImpl{snap: SnapshotIdle(), done: make(chan struct{})}
}

func (h *hostImpl) setActions(a Actions) {
	h.mu.Lock()
	h.actions = a
	h.mu.Unlock()
	h.pokeUI()
}

func (h *hostImpl) update(s state.Snapshot) {
	h.mu.Lock()
	h.snap = s
	h.mu.Unlock()
	h.pokeUI()
}

func (h *hostImpl) setWebhookFailed(failed bool) {
	h.mu.Lock()
	h.webhookFailed = failed
	h.mu.Unlock()
	h.pokeUI()
}

func (h *hostImpl) pokeUI() {
	if t := h.targetID(); t != 0 {
		t.Send(selPerformMain, selApplyUI, objc.ID(0), false)
	}
}

func (h *hostImpl) targetID() objc.ID {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.target
}

func (h *hostImpl) run(ready func()) {
	runtime.LockOSThread()
	darwinActive.Store(h)
	defer darwinActive.Store(nil)
	if err := h.setup(); err != nil {
		log.Printf("tray setup failed: %v; continuing without icon", err)
		startReady(ready)
		<-h.done
		return
	}
	defer h.teardown()
	startReady(ready)
	h.app.Send(selRun)
}

func (h *hostImpl) quit() {
	h.once.Do(func() {
		close(h.done)
		h.mu.Lock()
		app := h.app
		h.mu.Unlock()
		if app != 0 {
			app.Send(selPerformMain, selStop, objc.ID(0), false)
			cfRunLoopStopMain()
		}
	})
}

func (h *hostImpl) setup() error {
	if _, err := purego.Dlopen("/System/Library/Frameworks/Cocoa.framework/Cocoa", purego.RTLD_GLOBAL|purego.RTLD_LAZY); err != nil {
		return fmt.Errorf("cocoa: %w", err)
	}
	class, err := trayClass()
	if err != nil {
		return err
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(selSharedApplication)
	if app == 0 {
		return fmt.Errorf("NSApplication unavailable")
	}
	app.Send(selSetActivationPolicy, nsApplicationActivationPolicyAccessory)
	app.Send(selFinishLaunching)

	target := objc.ID(class).Send(objc.RegisterName("new"))
	if target == 0 {
		return fmt.Errorf("tray target")
	}

	bar := objc.ID(objc.GetClass("NSStatusBar")).Send(selSystemStatusBar)
	item := bar.Send(selStatusItemWithLen, nsVariableStatusItemLength)
	if item == 0 {
		return fmt.Errorf("status item")
	}
	item = item.Send(selRetain)
	menu := objc.ID(objc.GetClass("NSMenu")).Send(selAlloc).Send(selInitWithTitle, nsString("call-detect"))
	item.Send(selSetMenu, menu)

	h.mu.Lock()
	h.app = app
	h.bar = bar
	h.item = item
	h.menu = menu
	h.target = target
	h.ready = true
	h.mu.Unlock()
	h.applyUI()
	return nil
}

func (h *hostImpl) teardown() {
	h.mu.Lock()
	item, bar, menu, target := h.item, h.bar, h.menu, h.target
	h.item, h.bar, h.menu, h.target, h.app = 0, 0, 0, 0, 0
	h.ready = false
	h.mu.Unlock()
	if bar != 0 && item != 0 {
		bar.Send(selRemoveStatusItem, item)
	}
	if item != 0 {
		item.Send(selRelease)
	}
	if menu != 0 {
		menu.Send(selRelease)
	}
	if target != 0 {
		target.Send(selRelease)
	}
}

func (h *hostImpl) applyUI() {
	h.mu.Lock()
	if !h.ready || h.item == 0 {
		h.mu.Unlock()
		return
	}
	item := h.item
	menu := h.menu
	target := h.target
	snap := h.snap
	failed := h.webhookFailed
	actions := h.actions
	h.mu.Unlock()

	png := IdlePNG()
	if failed {
		png = ErrorPNG()
	} else if snap.Busy {
		png = BusyPNG()
	}
	img := imageFromPNG(png)
	button := item.Send(selButton)
	if button != 0 {
		button.Send(selSetImage, img)
		button.Send(selSetToolTip, nsString(TooltipAlert(snap, failed)))
	}

	if menu == 0 {
		return
	}
	menu.Send(selRemoveAllItems)
	for _, line := range statusLines(snap, failed) {
		menu.Send(selAddItem, disabledItem(line))
	}
	menu.Send(selAddItem, objc.ID(objc.GetClass("NSMenuItem")).Send(selSeparatorItem))
	for _, c := range actionChoices(actions) {
		menu.Send(selAddItem, actionItem(c.Label, tagFor(c.ID), target))
	}
}

func (h *hostImpl) handleTag(tag int) {
	h.mu.Lock()
	actions := h.actions
	h.mu.Unlock()
	handleMenuID(actions, idForTag(tag), h.quit)
}

func tagFor(id string) int {
	switch id {
	case menuInstall:
		return tagInstall
	case menuUninstall:
		return tagUninstall
	case menuWebhook:
		return tagWebhook
	case menuGitHub:
		return tagGitHub
	case menuQuit:
		return tagQuit
	default:
		return 0
	}
}

func idForTag(tag int) string {
	switch tag {
	case tagInstall:
		return menuInstall
	case tagUninstall:
		return menuUninstall
	case tagWebhook:
		return menuWebhook
	case tagGitHub:
		return menuGitHub
	case tagQuit:
		return menuQuit
	default:
		return ""
	}
}

func trayClass() (objc.Class, error) {
	darwinOnce.Do(func() {
		darwinClass, darwinClassErr = objc.RegisterClass(
			"CallDetectTrayTarget",
			objc.GetClass("NSObject"),
			nil,
			nil,
			[]objc.MethodDef{
				{Cmd: selMenuClick, Fn: darwinMenuClick},
				{Cmd: selApplyUI, Fn: darwinApplyUI},
			},
		)
	})
	return darwinClass, darwinClassErr
}

func darwinMenuClick(_ objc.ID, _ objc.SEL, sender objc.ID) {
	tag := int(objc.Send[int64](sender, selTag))
	if h := darwinActive.Load(); h != nil {
		go h.handleTag(tag)
	}
}

func darwinApplyUI(objc.ID, objc.SEL) {
	if h := darwinActive.Load(); h != nil {
		h.applyUI()
	}
}

func nsString(s string) objc.ID {
	return objc.ID(objc.GetClass("NSString")).Send(selStringUTF8, s)
}

func imageFromPNG(png []byte) objc.ID {
	if len(png) == 0 {
		return 0
	}
	data := objc.ID(objc.GetClass("NSData")).Send(selDataWithBytes, unsafe.Pointer(&png[0]), len(png))
	img := objc.ID(objc.GetClass("NSImage")).Send(selAlloc).Send(selInitWithData, data)
	if img != 0 {
		img.Send(selSetSize, nsSize{Width: 18, Height: 18})
		img.Send(selSetTemplate, false)
	}
	return img
}

func disabledItem(title string) objc.ID {
	item := objc.ID(objc.GetClass("NSMenuItem")).Send(selAlloc).Send(selInitMenuItem, nsString(title), objc.SEL(0), nsString(""))
	item.Send(selSetEnabled, false)
	return item
}

func actionItem(title string, tag int, target objc.ID) objc.ID {
	item := objc.ID(objc.GetClass("NSMenuItem")).Send(selAlloc).Send(selInitMenuItem, nsString(title), selMenuClick, nsString(""))
	item.Send(selSetTarget, target)
	item.Send(selSetTag, tag)
	item.Send(selSetEnabled, true)
	return item
}

func cfRunLoopStopMain() {
	cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_GLOBAL|purego.RTLD_LAZY)
	if err != nil {
		return
	}
	var getMain func() uintptr
	var stop func(uintptr)
	purego.RegisterLibFunc(&getMain, cf, "CFRunLoopGetMain")
	purego.RegisterLibFunc(&stop, cf, "CFRunLoopStop")
	if getMain != nil && stop != nil {
		stop(getMain())
	}
}
