//go:build windows

package wasapi

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/scallister/call-detect/internal/detect"
)

const (
	coinitMultithreaded = 0x0
	clsctxAll           = 0x17
	eRender             = 0
	eCapture            = 1
	deviceStateActive   = 0x1
	sessionStateActive  = 1
	sFalse              = 1
	processQueryLimited = 0x1000
)

var (
	ole32    = windows.NewLazySystemDLL("ole32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procCoInitializeEx       = ole32.NewProc("CoInitializeEx")
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
	procCoCreateInstance     = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
	queryFullProcessImage    = kernel32.NewProc("QueryFullProcessImageNameW")
	clsidMMDeviceEnumerator  = mustGUID("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	iidIMMDeviceEnumerator   = mustGUID("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	iidIAudioSessionManager2 = mustGUID("{77AA99A0-1BD6-484F-8BC7-2C654C9A9B6F}")
	iidIAudioSessionControl2 = mustGUID("{BFB7FF88-7239-4FC9-8FA2-07C950BE9C6D}")
)

type comObj struct{ vtbl *iUnknownVtbl }

type iUnknownVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
}

func listSessions() detect.Audio {
	hr, _, _ := procCoInitializeEx.Call(0, coinitMultithreaded)
	switch {
	case succeeded(hr) && hr != sFalse:
		defer procCoUninitialize.Call()
	case hr == sFalse, uint32(hr) == 0x80010106: // RPC_E_CHANGED_MODE
		// COM already initialized on this thread.
	default:
		return detect.Audio{Err: fmt.Errorf("CoInitializeEx: 0x%x", uint32(hr))}
	}

	var enum *comObj
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)),
		0,
		clsctxAll,
		uintptr(unsafe.Pointer(&iidIMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enum)),
	)
	if !succeeded(hr) || enum == nil {
		return detect.Audio{Err: fmt.Errorf("MMDeviceEnumerator: 0x%x", uint32(hr))}
	}
	defer release(enum)

	capture, err := appsOnFlow(enum, eCapture)
	if err != nil {
		return detect.Audio{Err: err}
	}
	render, err := appsOnFlow(enum, eRender)
	if err != nil {
		return detect.Audio{Err: err}
	}
	return detect.Audio{Capture: capture, Render: render}
}

func appsOnFlow(enum *comObj, flow uint32) ([]string, error) {
	var coll *comObj
	hr := call(enum, 3, uintptr(flow), deviceStateActive, uintptr(unsafe.Pointer(&coll)))
	if !succeeded(hr) || coll == nil {
		return nil, fmt.Errorf("EnumAudioEndpoints(%d): 0x%x", flow, uint32(hr))
	}
	defer release(coll)

	var count uint32
	hr = call(coll, 3, uintptr(unsafe.Pointer(&count)))
	if !succeeded(hr) {
		return nil, fmt.Errorf("GetCount: 0x%x", uint32(hr))
	}

	seen := map[string]struct{}{}
	var names []string
	for i := uint32(0); i < count; i++ {
		var dev *comObj
		hr = call(coll, 4, uintptr(i), uintptr(unsafe.Pointer(&dev)))
		if !succeeded(hr) || dev == nil {
			continue
		}
		for _, n := range activeAppsOnDevice(dev) {
			if _, ok := seen[n]; ok {
				continue
			}
			seen[n] = struct{}{}
			names = append(names, n)
		}
		release(dev)
	}
	return names, nil
}

func activeAppsOnDevice(dev *comObj) []string {
	var mgr *comObj
	hr := call(dev, 3,
		uintptr(unsafe.Pointer(&iidIAudioSessionManager2)),
		clsctxAll,
		0,
		uintptr(unsafe.Pointer(&mgr)),
	)
	if !succeeded(hr) || mgr == nil {
		return nil
	}
	defer release(mgr)

	var sessions *comObj
	// IAudioSessionManager2::GetSessionEnumerator is slot 5
	// (IUnknown 0-2, IAudioSessionManager 3-4).
	hr = call(mgr, 5, uintptr(unsafe.Pointer(&sessions)))
	if !succeeded(hr) || sessions == nil {
		return nil
	}
	defer release(sessions)

	var n int32
	hr = call(sessions, 3, uintptr(unsafe.Pointer(&n)))
	if !succeeded(hr) {
		return nil
	}

	var names []string
	for i := int32(0); i < n; i++ {
		var ctl *comObj
		hr = call(sessions, 4, uintptr(i), uintptr(unsafe.Pointer(&ctl)))
		if !succeeded(hr) || ctl == nil {
			continue
		}
		if name, ok := activeSessionExe(ctl); ok {
			names = append(names, name)
		}
		release(ctl)
	}
	return names
}

func activeSessionExe(ctl *comObj) (string, bool) {
	var state uint32
	if hr := call(ctl, 3, uintptr(unsafe.Pointer(&state))); !succeeded(hr) || state != sessionStateActive {
		return "", false
	}

	var ctl2 *comObj
	hr := call(ctl, 0, uintptr(unsafe.Pointer(&iidIAudioSessionControl2)), uintptr(unsafe.Pointer(&ctl2)))
	if !succeeded(hr) || ctl2 == nil {
		return "", false
	}
	defer release(ctl2)

	if call(ctl2, 15) == 0 {
		// IsSystemSoundsSession (slot 15) returns S_OK for system sounds.
		return "", false
	}

	var pid uint32
	if hr := call(ctl2, 14, uintptr(unsafe.Pointer(&pid))); !succeeded(hr) || pid == 0 {
		return "", false
	}
	name, err := processBase(pid)
	if err != nil || name == "" {
		return "", false
	}
	return name, true
}

func processBase(pid uint32) (string, error) {
	h, err := windows.OpenProcess(processQueryLimited, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)

	n := uint32(32768)
	buf := make([]uint16, n)
	hr, _, callErr := queryFullProcessImage.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
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
		return 0x80004003 // E_POINTER
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
