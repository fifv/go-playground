package main

import (
	// "log"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
)

func init() {
	syscall.NewLazyDLL("user32.dll").NewProc("SetProcessDPIAware").Call()

	/**
	 * This works!!! Fix Window Randomly stuck
	 * calls on the beginning of main() also works
	 */
	runtime.LockOSThread() // This is the magic fix
}

var (
	moduser32            = syscall.NewLazyDLL("user32.dll")
	dwmapi               = syscall.NewLazyDLL("dwmapi.dll")
	modwingdi            = syscall.NewLazyDLL("gdi32.dll")
	procCreateSolidBrush = modwingdi.NewProc("CreateSolidBrush")
	procSetClassLongPtr  = moduser32.NewProc("SetClassLongPtrW")
	procSetClassLong     = moduser32.NewProc("SetClassLongW")
)

// Define RGB manually if w32.RGB is undefined
func RGB(r, g, b byte) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}
func main() {
	instance := w32.GetModuleHandle("")

	// 1. Create a Dark Brush (Dark Gray/Black)

	// darkBackgroundBrush := w32.CreateSolidBrush(RGB(32, 32, 32))

	className := "DarkWindow"
	var wc w32.WNDCLASSEX
	wc.Size = uint32(unsafe.Sizeof(wc))
	wc.Style = w32.CS_HREDRAW | w32.CS_VREDRAW
	wc.WndProc = syscall.NewCallback(wndProc)
	wc.Instance = instance
	wc.Cursor = w32.LoadCursor(0, w32.MakeIntResource(w32.IDC_ARROW))

	// 2. Assign the dark brush to the background
	wc.Background = w32.CreateSolidBrush(RGB(0, 0, 0)) /* darkBackgroundBrush */
	// wc.Background = w32.COLOR_BTNFACE + 1
	wc.ClassName = syscall.StringToUTF16Ptr(className)

	w32.RegisterClassEx(&wc)

	hwnd := w32.CreateWindowEx(
		0,
		// w32.WS_EX_CONTROLPARENT | w32.WS_EX_APPWINDOW,
		/* !!!!!!!!!!!!! This is WS_EX_COMPOSITED, not in w32, it works!!! */
		// w32.WS_EX_COMPOSITED,
		syscall.StringToUTF16Ptr(className),
		syscall.StringToUTF16Ptr("Go w32 Dark Mode"),
		// w32.WS_OVERLAPPEDWINDOW|w32.WS_VISIBLE,
		w32.WS_OVERLAPPEDWINDOW,
		w32.CW_USEDEFAULT, w32.CW_USEDEFAULT, w32.CW_USEDEFAULT, w32.CW_USEDEFAULT,
		0, 0, instance, nil,
	)

	setImmersiveDarkMode(hwnd)
	// SetBackgroundColour(uintptr(hwnd), 33, 33, 33)
	// SetBackgroundColour(uintptr(hwnd), 0, 0, 0)
	/* !!! SetWindowPos before ShowWindow also can elimiate the white flash */
	CenterWindow(hwnd)
	w32.ShowWindow(hwnd, 1)

	var msg w32.MSG
	for w32.GetMessage(&msg, 0, 0, 0) != 0 {
		w32.TranslateMessage(&msg)
		w32.DispatchMessage(&msg)
	}
}

func wndProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	// log.Println("msg:", msg)

	switch msg {
	case w32.WM_CREATE:
		/* wtf it works?! */
		/* !!! you can also call SetWindowPos before ShowWindow, it also elimiate the white flash */
		// w32.SetWindowPos(hwnd, w32.HWND(0), 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_FRAMECHANGED)
		return w32.DefWindowProc(hwnd, msg, wParam, lParam)
	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)
		return 0
	case w32.WM_PAINT:
		return w32.DefWindowProc(hwnd, msg, wParam, lParam)
	case w32.WM_ERASEBKGND:
		return w32.DefWindowProc(hwnd, msg, wParam, lParam)

	default:
		return w32.DefWindowProc(hwnd, msg, wParam, lParam)
	}
}

func setImmersiveDarkMode(hwnd w32.HWND) {
	// DWMWA_USE_IMMERSIVE_DARK_MODE attribute
	const DWMWA_USE_IMMERSIVE_DARK_MODE = 20
	isDark := 1

	proc := dwmapi.NewProc("DwmSetWindowAttribute")

	proc.Call(
		uintptr(hwnd),
		uintptr(DWMWA_USE_IMMERSIVE_DARK_MODE),
		uintptr(unsafe.Pointer(&isDark)),
		uintptr(4),
	)
}

func SetBackgroundColour(hwnd uintptr, r, g, b uint8) {
	const (
		GCLP_HBRBACKGROUND int32 = -10
	)

	col := RGB(r, g, b)
	hbrush, _, _ := procCreateSolidBrush.Call(uintptr(col))
	setClassLongPtr(hwnd, GCLP_HBRBACKGROUND, hbrush)
}

func setClassLongPtr(hwnd uintptr, param int32, val uintptr) bool {
	proc := procSetClassLongPtr
	if strconv.IntSize == 32 {
		/*
			https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setclasslongptrw
			Note: 	To write code that is compatible with both 32-bit and 64-bit Windows, use SetClassLongPtr.
					When compiling for 32-bit Windows, SetClassLongPtr is defined as a call to the SetClassLong function

			=> We have to do this dynamically when directly calling the DLL procedures
		*/
		proc = procSetClassLong
	}

	ret, _, _ := proc.Call(
		hwnd,
		uintptr(param),
		val,
	)
	return ret != 0
}

func CenterWindow(hwnd w32.HWND) {

	// windowInfo := getWindowInfo(hwnd)
	// frameless := false

	info := getMonitorInfo(hwnd)
	workRect := info.RcWork
	screenMiddleW := workRect.Left + (workRect.Right-workRect.Left)/2
	screenMiddleH := workRect.Top + (workRect.Bottom-workRect.Top)/2
	var winRect *w32.RECT
	// if !frameless {
	winRect = w32.GetWindowRect(hwnd)
	// } else {
	// winRect = w32.GetClientRect(hwnd)
	// }
	winWidth := winRect.Right - winRect.Left
	winHeight := winRect.Bottom - winRect.Top
	windowX := screenMiddleW - (winWidth / 2)
	windowY := screenMiddleH - (winHeight / 2)
	w32.SetWindowPos(hwnd, w32.HWND_TOP, int(windowX), int(windowY), int(winWidth), int(winHeight), w32.SWP_NOSIZE)
}

//	func getWindowInfo(hwnd w32.HWND) *w32.WINDOWINFO {
//		var info w32.WINDOWINFO
//		info.CbSize = uint32(unsafe.Sizeof(info))
//		w32.GetWindowInfo(hwnd, &info)
//		return &info
//	}
func getMonitorInfo(hwnd w32.HWND) *w32.MONITORINFO {
	currentMonitor := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	var info w32.MONITORINFO
	info.CbSize = uint32(unsafe.Sizeof(info))
	w32.GetMonitorInfo(currentMonitor, &info)
	return &info
}
