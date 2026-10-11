package main

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	waveOutMessage        = winmm.NewProc("waveOutMessage")
	ole32                 = windows.NewLazySystemDLL("ole32.dll")
	coCreateInstance      = ole32.NewProc("CoCreateInstance")
	propVariantClear      = ole32.NewProc("PropVariantClear")
	clsidDeviceEnumerator = windows.GUID{Data1: 0xbcde0395, Data2: 0xe52f, Data3: 0x467c, Data4: [8]byte{0x8e, 0x3d, 0xc4, 0x57, 0x92, 0x91, 0x69, 0x2e}}
	iidDeviceEnumerator   = windows.GUID{Data1: 0xa95664d2, Data2: 0x9614, Data3: 0x4f35, Data4: [8]byte{0xa7, 0x46, 0xde, 0x8d, 0xb6, 0x36, 0x17, 0xe6}}
	friendlyNameKey       = struct {
		Format windows.GUID
		ID     uint32
	}{windows.GUID{Data1: 0xa45c254e, Data2: 0xdf1c, Data3: 0x4efd, Data4: [8]byte{0x80, 0x20, 0x67, 0xd1, 0x46, 0xa8, 0x50, 0xe0}}, 14}
)

// These three COM interfaces have at most eight vtable entries. Only the
// documented IMMDeviceEnumerator, IMMDevice and IPropertyStore slots are used.
type audioCOMObject struct{ vtable *[8]uintptr }

//go:uintptrescapes
func (o *audioCOMObject) call(slot int, args ...uintptr) bool {
	params := append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)
	hr, _, _ := syscall.SyscallN(o.vtable[slot], params...)
	runtime.KeepAlive(o)
	return int32(hr) >= 0
}

func (o *audioCOMObject) release() { o.call(2) }

// WinMM's WAVEOUTCAPS name has only 32 UTF-16 characters. Resolve the exact
// endpoint ID through WinMM, then read its unabridged friendly name via Core
// Audio. Keep legacy names if an older driver does not support the lookup.
// https://learn.microsoft.com/en-us/windows/win32/coreaudio/device-properties
func resolveOutputNames(devices []outputDevice) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err != nil && err != syscall.Errno(1) {
		return
	} // S_FALSE is also success.
	defer windows.CoUninitialize()
	var enumerator *audioCOMObject
	hr, _, _ := coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidDeviceEnumerator)), 0, 1, uintptr(unsafe.Pointer(&iidDeviceEnumerator)), uintptr(unsafe.Pointer(&enumerator)))
	if int32(hr) < 0 || enumerator == nil {
		return
	}
	defer enumerator.release()
	for i := range devices {
		if name := endpointName(enumerator, devices[i].id); name != "" {
			devices[i].name = name
		}
	}
}

func endpointName(enumerator *audioCOMObject, id uintptr) string {
	var endpoint *audioCOMObject
	if id == waveMapper {
		// IMMDeviceEnumerator::GetDefaultAudioEndpoint(eRender, eConsole).
		if !enumerator.call(4, 0, 0, uintptr(unsafe.Pointer(&endpoint))) {
			return ""
		}
	} else {
		const queryID, queryIDSize = 0x0811, 0x0812
		var size uintptr // Byte count, including the UTF-16 terminator.
		if err := mmCall(waveOutMessage, id, queryIDSize, uintptr(unsafe.Pointer(&size)), 0); err != nil || size < 2 || size > 65536 || size%2 != 0 {
			return ""
		}
		identifier := make([]uint16, size/2)
		if err := mmCall(waveOutMessage, id, queryID, uintptr(unsafe.Pointer(&identifier[0])), size); err != nil {
			return ""
		}
		// IMMDeviceEnumerator::GetDevice uses the endpoint ID, not a name prefix.
		if !enumerator.call(5, uintptr(unsafe.Pointer(&identifier[0])), uintptr(unsafe.Pointer(&endpoint))) {
			return ""
		}
	}
	if endpoint == nil {
		return ""
	}
	defer endpoint.release()
	var properties *audioCOMObject
	if !endpoint.call(4, 0, uintptr(unsafe.Pointer(&properties))) || properties == nil {
		return ""
	} // OpenPropertyStore(STGM_READ).
	defer properties.release()
	// PROPVARIANT: 8-byte header followed by a union (two pointer-sized words).
	var value struct {
		Type     uint16
		Reserved [3]uint16
		Text     *uint16
		Unused   uintptr
	}
	defer propVariantClear.Call(uintptr(unsafe.Pointer(&value)))
	if !properties.call(5, uintptr(unsafe.Pointer(&friendlyNameKey)), uintptr(unsafe.Pointer(&value))) || value.Type != 31 || value.Text == nil {
		return ""
	} // GetValue, VT_LPWSTR.
	return windows.UTF16PtrToString(value.Text)
}
