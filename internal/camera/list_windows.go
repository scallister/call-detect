//go:build windows

package camera

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/scallister/call-detect/internal/detect"
)

const (
	coinitMultithreaded = 0x0
	sFalse              = 1
	processQueryLimited = 0x1000
	mfVersion           = 0x00020070 // MF_SDK_VERSION<<16 | MF_API_VERSION
	mfStartupLite       = 0x1
	eNoInterface        = 0x80004002
	ePointer            = 0x80004003
	rpcEChangedMode     = 0x80010106
)

var (
	ole32         = windows.NewLazySystemDLL("ole32.dll")
	kernel32      = windows.NewLazySystemDLL("kernel32.dll")
	mfplat        = windows.NewLazySystemDLL("mfplat.dll")
	mfsensorgroup = windows.NewLazySystemDLL("mfsensorgroup.dll")

	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
	queryFullProcess   = kernel32.NewProc("QueryFullProcessImageNameW")
	procMFStartup      = mfplat.NewProc("MFStartup")
	procCreateMonitor  = mfsensorgroup.NewProc("MFCreateSensorActivityMonitor")

	iidIUnknown = mustGUID("{00000000-0000-0000-C000-000000000046}")
	// IMFSensorActivitiesReportCallback from mfidl.h
	iidCallback = mustGUID("{DE5072EE-DBE3-46DC-8A87-B6F631194751}")
)

type comObj struct{ vtbl *iUnknownVtbl }

type iUnknownVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
}

type callbackVtbl struct {
	queryInterface     uintptr
	addRef             uintptr
	release            uintptr
	onActivitiesReport uintptr
}

type callback struct {
	vtbl *callbackVtbl
	refs int32
}

var cbVtbl = callbackVtbl{
	queryInterface:     windows.NewCallback(cbQueryInterface),
	addRef:             windows.NewCallback(cbAddRef),
	release:            windows.NewCallback(cbRelease),
	onActivitiesReport: windows.NewCallback(cbOnActivitiesReport),
}

var (
	cbObj  = &callback{vtbl: &cbVtbl, refs: 1}
	pinner runtime.Pinner

	startOnce   sync.Once
	startErr    error
	activityMon *comObj

	cacheMu sync.Mutex
	cache   []string

	firstCh   = make(chan struct{})
	firstOnce sync.Once
)

func init() {
	pinner.Pin(cbObj)
}

func listStreaming() detect.Camera {
	if err := ensureStarted(); err != nil {
		return detect.Camera{Err: err}
	}
	waitFirst(750 * time.Millisecond)
	return detect.Camera{Streaming: currentStreaming()}
}

func ensureStarted() error {
	startOnce.Do(func() {
		startErr = startMonitor()
	})
	return startErr
}

func startMonitor() error {
	errc := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		if err := initCOMAndMonitor(); err != nil {
			errc <- err
			return
		}
		errc <- nil
		select {}
	}()
	return <-errc
}

func initCOMAndMonitor() error {
	hr, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded)
	switch {
	case succeeded(hr):
		// Leave COM initialized on this thread for the monitor lifetime.
	case hr == sFalse, uint32(hr) == rpcEChangedMode:
	default:
		return fmt.Errorf("CoInitializeEx: 0x%x", uint32(hr))
	}

	if err := mfplat.Load(); err == nil {
		_, _, _ = procMFStartup.Call(mfVersion, mfStartupLite)
	}
	if err := mfsensorgroup.Load(); err != nil {
		return fmt.Errorf("load mfsensorgroup.dll: %w", err)
	}

	var mon *comObj
	hr, _, _ = procCreateMonitor.Call(uintptr(unsafe.Pointer(cbObj)), uintptr(unsafe.Pointer(&mon)))
	if !succeeded(hr) || mon == nil {
		return fmt.Errorf("MFCreateSensorActivityMonitor: 0x%x", uint32(hr))
	}
	if hr := call(mon, 3); !succeeded(hr) {
		release(mon)
		return fmt.Errorf("IMFSensorActivityMonitor.Start: 0x%x", uint32(hr))
	}
	activityMon = mon
	return nil
}

func setStreaming(names []string) {
	cacheMu.Lock()
	cache = append([]string(nil), names...)
	cacheMu.Unlock()
	firstOnce.Do(func() { close(firstCh) })
}

func currentStreaming() []string {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return append([]string(nil), cache...)
}

func waitFirst(d time.Duration) {
	select {
	case <-firstCh:
	case <-time.After(d):
	}
}

func cbQueryInterface(this *callback, iid *windows.GUID, ppv *unsafe.Pointer) uintptr {
	if this == nil || iid == nil {
		return ePointer
	}
	if ppv == nil {
		return ePointer
	}
	if *iid != iidIUnknown && *iid != iidCallback {
		*ppv = nil
		return eNoInterface
	}
	*ppv = unsafe.Pointer(this)
	cbAddRef(this)
	return 0
}

func cbAddRef(this *callback) uintptr {
	if this == nil {
		return 0
	}
	return uintptr(atomic.AddInt32(&this.refs, 1))
}

func cbRelease(this *callback) uintptr {
	if this == nil {
		return 0
	}
	n := atomic.AddInt32(&this.refs, -1)
	if n < 0 {
		return 0
	}
	return uintptr(n)
}

func cbOnActivitiesReport(_ *callback, report *comObj) uintptr {
	setStreaming(collectStreaming(report))
	return 0
}

func collectStreaming(report *comObj) []string {
	if report == nil {
		return nil
	}
	var count uint32
	if !succeeded(call(report, 3, uintptr(unsafe.Pointer(&count)))) {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for i := uint32(0); i < count; i++ {
		var act *comObj
		if !succeeded(call(report, 4, uintptr(i), uintptr(unsafe.Pointer(&act)))) || act == nil {
			continue
		}
		for _, n := range streamingOnSensor(act) {
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			out = append(out, n)
		}
		release(act)
	}
	slices.Sort(out)
	return out
}

func streamingOnSensor(act *comObj) []string {
	friendly := readMFString(act, 3)
	symlink := readMFString(act, 4)
	if isMicrophoneSensor(friendly, symlink) {
		return nil
	}
	var n uint32
	if !succeeded(call(act, 5, uintptr(unsafe.Pointer(&n)))) {
		return nil
	}
	var names []string
	for i := uint32(0); i < n; i++ {
		var proc *comObj
		if !succeeded(call(act, 6, uintptr(i), uintptr(unsafe.Pointer(&proc)))) || proc == nil {
			continue
		}
		name, ok := streamingExe(proc)
		release(proc)
		if ok {
			names = append(names, name)
		}
	}
	return names
}

func streamingExe(proc *comObj) (string, bool) {
	var streaming int32
	if !succeeded(call(proc, 4, uintptr(unsafe.Pointer(&streaming)))) || streaming == 0 {
		return "", false
	}
	var pid uint32
	if !succeeded(call(proc, 3, uintptr(unsafe.Pointer(&pid)))) || pid == 0 {
		return "", false
	}
	name, err := processBase(pid)
	if err != nil || name == "" {
		return "", false
	}
	return name, true
}

func readMFString(obj *comObj, slot uintptr) string {
	const n = 1024
	buf := make([]uint16, n)
	var written uint32
	hr := call(obj, slot, uintptr(unsafe.Pointer(&buf[0])), uintptr(n), uintptr(unsafe.Pointer(&written)))
	if !succeeded(hr) {
		return ""
	}
	return windows.UTF16ToString(buf)
}

func processBase(pid uint32) (string, error) {
	h, err := windows.OpenProcess(processQueryLimited, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)

	n := uint32(32768)
	buf := make([]uint16, n)
	hr, _, callErr := queryFullProcess.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if hr == 0 {
		if callErr != nil {
			return "", callErr
		}
		return "", fmt.Errorf("QueryFullProcessImageName")
	}
	return filepath.Base(windows.UTF16ToString(buf[:n])), nil
}

func call(obj *comObj, slot uintptr, args ...uintptr) uintptr {
	if obj == nil || obj.vtbl == nil {
		return ePointer
	}
	fn := *(*uintptr)(unsafe.Add(unsafe.Pointer(obj.vtbl), slot*unsafe.Sizeof(uintptr(0))))
	all := make([]uintptr, 0, 1+len(args))
	all = append(all, uintptr(unsafe.Pointer(obj)))
	all = append(all, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	return r
}

func release(obj *comObj) {
	if obj != nil {
		call(obj, 2)
	}
}

func succeeded(hr uintptr) bool {
	return int32(hr) >= 0
}

func mustGUID(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}
