//go:build darwin

package live

import (
	"bytes"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	kAudioObjectSystemObject                     = 1
	kAudioObjectPropertyElementMain              = 0
	kAudioObjectPropertyScopeGlobal              = 0x676C6F62 // 'glob'
	kAudioHardwarePropertyDevices                = 0x64657623 // 'dev#'
	kAudioDevicePropertyStreams                  = 0x73746D23 // 'stm#'
	kAudioDevicePropertyScopeInput               = 0x696E7074 // 'inpt'
	kAudioDevicePropertyDeviceIsRunningSomewhere = 0x72756E6E // 'runn'
	kAudioObjectPropertyName                     = 0x6C6E616D // 'lnam'
	kCMIODevicePropertyDeviceIsRunningSomewhere  = 0x676F696E // 'goin'
	kCFStringEncodingUTF8                        = 0x08000100
)

type propertyAddress struct {
	Selector uint32
	Scope    uint32
	Element  uint32
}

type objectGetPropertyData func(object uint32, address *propertyAddress, qualSize uint32, qual unsafe.Pointer, dataSize *uint32, data unsafe.Pointer) int32

type objectGetPropertyDataSize func(object uint32, address *propertyAddress, qualSize uint32, qual unsafe.Pointer, dataSize *uint32) int32

var (
	halOnce      sync.Once
	halErr       error
	audioGet     objectGetPropertyData
	audioGetSz   objectGetPropertyDataSize
	cmioGet      objectGetPropertyData
	cmioGetSz    objectGetPropertyDataSize
	cfGetCString func(str uintptr, buf *byte, size int32, encoding uint32) uint8
	cfRelease    func(cf uintptr)
)

func loadHAL() error {
	halOnce.Do(func() {
		audio, err := purego.Dlopen("/System/Library/Frameworks/CoreAudio.framework/CoreAudio", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			halErr = fmt.Errorf("coreaudio: %w", err)
			return
		}
		purego.RegisterLibFunc(&audioGet, audio, "AudioObjectGetPropertyData")
		purego.RegisterLibFunc(&audioGetSz, audio, "AudioObjectGetPropertyDataSize")

		cmio, err := purego.Dlopen("/System/Library/Frameworks/CoreMediaIO.framework/CoreMediaIO", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			halErr = fmt.Errorf("coremediaio: %w", err)
			return
		}
		purego.RegisterLibFunc(&cmioGet, cmio, "CMIOObjectGetPropertyData")
		purego.RegisterLibFunc(&cmioGetSz, cmio, "CMIOObjectGetPropertyDataSize")

		cf, err := purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			halErr = fmt.Errorf("corefoundation: %w", err)
			return
		}
		purego.RegisterLibFunc(&cfGetCString, cf, "CFStringGetCString")
		purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")
	})
	return halErr
}

func listObjectIDs(getSz objectGetPropertyDataSize, get objectGetPropertyData) ([]uint32, error) {
	addr := propertyAddress{Selector: kAudioHardwarePropertyDevices, Scope: kAudioObjectPropertyScopeGlobal, Element: kAudioObjectPropertyElementMain}
	var size uint32
	if st := getSz(kAudioObjectSystemObject, &addr, 0, nil, &size); st != 0 {
		return nil, fmt.Errorf("property size: status %d", st)
	}
	if size == 0 || size%4 != 0 {
		return nil, nil
	}
	buf := make([]uint32, size/4)
	if st := get(kAudioObjectSystemObject, &addr, 0, nil, &size, unsafe.Pointer(&buf[0])); st != 0 {
		return nil, fmt.Errorf("property data: status %d", st)
	}
	return buf[:size/4], nil
}

func objectUint32(get objectGetPropertyData, id uint32, selector, scope uint32) (uint32, bool) {
	addr := propertyAddress{Selector: selector, Scope: scope, Element: kAudioObjectPropertyElementMain}
	var val uint32
	size := uint32(4)
	if st := get(id, &addr, 0, nil, &size, unsafe.Pointer(&val)); st != 0 {
		return 0, false
	}
	return val, true
}

func objectHasStreams(getSz objectGetPropertyDataSize, id uint32, scope uint32) bool {
	addr := propertyAddress{Selector: kAudioDevicePropertyStreams, Scope: scope, Element: kAudioObjectPropertyElementMain}
	var size uint32
	if st := getSz(id, &addr, 0, nil, &size); st != 0 {
		return false
	}
	return size > 0
}

func objectName(get objectGetPropertyData, id uint32) string {
	if cfGetCString == nil {
		return ""
	}
	addr := propertyAddress{Selector: kAudioObjectPropertyName, Scope: kAudioObjectPropertyScopeGlobal, Element: kAudioObjectPropertyElementMain}
	var cf uintptr
	size := uint32(unsafe.Sizeof(cf))
	if st := get(id, &addr, 0, nil, &size, unsafe.Pointer(&cf)); st != 0 || cf == 0 {
		return ""
	}
	defer cfRelease(cf)
	var buf [256]byte
	if cfGetCString(cf, &buf[0], int32(len(buf)), kCFStringEncodingUTF8) == 0 {
		return ""
	}
	n := bytes.IndexByte(buf[:], 0)
	if n < 0 {
		n = len(buf)
	}
	return string(buf[:n])
}

func runningInputNames() ([]string, error) {
	if err := loadHAL(); err != nil {
		return nil, err
	}
	ids, err := listObjectIDs(audioGetSz, audioGet)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, id := range ids {
		if !objectHasStreams(audioGetSz, id, kAudioDevicePropertyScopeInput) {
			continue
		}
		running, ok := objectUint32(audioGet, id, kAudioDevicePropertyDeviceIsRunningSomewhere, kAudioObjectPropertyScopeGlobal)
		if !ok || running == 0 {
			continue
		}
		name := objectName(audioGet, id)
		if name == "" {
			name = "microphone"
		}
		names = append(names, name)
	}
	return uniqueNames(names), nil
}

func runningCameraNames() ([]string, error) {
	if err := loadHAL(); err != nil {
		return nil, err
	}
	ids, err := listObjectIDs(cmioGetSz, cmioGet)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, id := range ids {
		running, ok := objectUint32(cmioGet, id, kCMIODevicePropertyDeviceIsRunningSomewhere, kAudioObjectPropertyScopeGlobal)
		if !ok || running == 0 {
			continue
		}
		name := objectName(cmioGet, id)
		if name == "" {
			name = "camera"
		}
		names = append(names, name)
	}
	return uniqueNames(names), nil
}
