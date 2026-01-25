package main

import (
	// "fmt"
	"fmt"
	"log"

	"runtime"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
	"github.com/wailsapp/go-webview2/pkg/edge"
	// "github.com/wailsapp/wails/v2/internal/frontend/desktop/windows"
	// "github.com/wailsapp/wails/v2/internal/frontend/desktop/windows/winc"
)

/**
 * TODO\: 2026.01.18 24:25 1. resize doesn't work 2. while flickering (seem chromium itself)
 * TODO\: 2026.01.25 22:51 1. resize doesn't work
 */

var (
	g_chromium *edge.Chromium
)

func main() {
	hwnd := createWindow()
	chromium := edge.NewChromium()
	g_chromium = chromium

	chromium.MessageCallback = func(message string, sender *edge.ICoreWebView2, args *edge.ICoreWebView2WebMessageReceivedEventArgs) {
		fmt.Println("MessageCallback", message)
	}
	chromium.MessageWithAdditionalObjectsCallback = func(message string, sender *edge.ICoreWebView2, args *edge.ICoreWebView2WebMessageReceivedEventArgs) {
		fmt.Println("MessageWithAdditionalObjectsCallback", message)
	}
	chromium.WebResourceRequestedCallback = func(request *edge.ICoreWebView2WebResourceRequest, args *edge.ICoreWebView2WebResourceRequestedEventArgs) {
		fmt.Println("WebResourceRequestedCallback")
	}
	chromium.NavigationCompletedCallback = func(sender *edge.ICoreWebView2, args *edge.ICoreWebView2NavigationCompletedEventArgs) {

		/* WORKS! */
		// Hack to make it visible: https://github.com/MicrosoftEdge/WebView2Feedback/issues/1077#issuecomment-825375026
		err := chromium.Hide()
		if err != nil {
			log.Fatal(err)
		}
		err = chromium.Show()
		if err != nil {
			log.Fatal(err)
		}

		w32.ShowWindow(hwnd, 1)

	}
	chromium.AcceleratorKeyCallback = func(vkey uint) bool {
		fmt.Println("AcceleratorKeyCallback")
		return false
	}
	chromium.ProcessFailedCallback = func(sender *edge.ICoreWebView2, args *edge.ICoreWebView2ProcessFailedEventArgs) {
		log.Fatal("ProcessFailedCallback")
	}

	chromium.Embed(uintptr(hwnd))

	chromium.Resize()
	settings, err := chromium.GetSettings()
	if err != nil {
		log.Fatal(err)
	}

	err = settings.PutIsStatusBarEnabled(false)
	if err != nil {
		log.Fatal(err)
	}
	err = settings.PutAreBrowserAcceleratorKeysEnabled(false)
	if err != nil {
		log.Fatal(err)
	}

	// if f.debug && f.frontendOptions.Debug.OpenInspectorOnStartup {
	// chromium.OpenDevToolsWindow()
	// }

	// Setup focus event handler

	// Set background colour
	// f.WindowSetBackgroundColour(f.frontendOptions.BackgroundColour)
	setChromiumBackground(chromium, 255, 0, 0, true)

	chromium.SetGlobalPermission(edge.CoreWebView2PermissionStateAllow)
	chromium.AddWebResourceRequestedFilter("*", edge.COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL)
	// chromium.Navigate("https://google.com")
	// chromium.Navigate("http://localhost:4173/")
	chromium.Navigate("http://localhost:3000/")

	runMsgLoop()
}

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
func createWindow() w32.HWND {
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
	// wc.Background = w32.CreateSolidBrush(RGB(0, 0, 0)) /* darkBackgroundBrush */
	wc.Background = w32.CreateSolidBrush(RGB(33, 37, 43)) /* darkBackgroundBrush */
	// wc.Background = w32.CreateSolidBrush(RGB(233, 0, 0)) /* darkBackgroundBrush */
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
	Center(hwnd)
	// w32.ShowWindow(hwnd, 1)

	return hwnd
}
func runMsgLoop() {
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
		w32.SetWindowPos(hwnd, w32.HWND(0), 0, 0, 0, 0, w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_FRAMECHANGED)
		return w32.DefWindowProc(hwnd, msg, wParam, lParam)
	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)
		return 0
	case w32.WM_MOVE, w32.WM_MOVING:
		if g_chromium != nil {
			g_chromium.NotifyParentWindowPositionChanged()
			log.Println("changed!")
		} else {
			log.Println("oh nil")
		}
	case w32.WM_SIZE:
		/* just works */
		g_chromium.Resize()
	case 0x02E0: /* w32.WM_DPICHANGED */
		log.Println("w32.WM_DPICHANGED")
	case w32.WM_NCCALCSIZE:
		log.Println("w32.WM_NCCALCSIZE")

		// Disable the standard frame by allowing the client area to take the full
		// window size.
		// See: https://docs.microsoft.com/en-us/windows/win32/winmsg/wm-nccalcsize#remarks
		// This hides the titlebar and also disables the resizing from user interaction because the standard frame is not
		// shown. We still need the WS_THICKFRAME style to enable resizing from the frontend.
		// if wParam != 0 {
		// 	rgrc := (*w32.RECT)(unsafe.Pointer(lParam))
		// 	if w.Form.IsFullScreen() {
		// 		// In Full-Screen mode we don't need to adjust anything
		// 		w.SetPadding(edge.Rect{})
		// 	} else if w.IsMaximised() {
		// 		// If the window is maximized we must adjust the client area to the work area of the monitor. Otherwise
		// 		// some content goes beyond the visible part of the monitor.
		// 		// Make sure to use the provided RECT to get the monitor, because during maximizig there might be
		// 		// a wrong monitor returned in multi screen mode when using MonitorFromWindow.
		// 		// See: https://github.com/MicrosoftEdge/WebView2Feedback/issues/2549
		// 		monitor := w32.MonitorFromRect(rgrc, w32.MONITOR_DEFAULTTONULL)

		// 		var monitorInfo w32.MONITORINFO
		// 		monitorInfo.CbSize = uint32(unsafe.Sizeof(monitorInfo))
		// 		if monitor != 0 && w32.GetMonitorInfo(monitor, &monitorInfo) {
		// 			*rgrc = monitorInfo.RcWork

		// 			maxWidth := w.frontendOptions.MaxWidth
		// 			maxHeight := w.frontendOptions.MaxHeight
		// 			if maxWidth > 0 || maxHeight > 0 {
		// 				var dpiX, dpiY uint
		// 				GetDPIForMonitor(monitor, w32.MDT_EFFECTIVE_DPI, &dpiX, &dpiY)

		// 				maxWidth := int32(ScaleWithDPI(maxWidth, dpiX))
		// 				if maxWidth > 0 && rgrc.Right-rgrc.Left > maxWidth {
		// 					rgrc.Right = rgrc.Left + maxWidth
		// 				}

		// 				maxHeight := int32(ScaleWithDPI(maxHeight, dpiY))
		// 				if maxHeight > 0 && rgrc.Bottom-rgrc.Top > maxHeight {
		// 					rgrc.Bottom = rgrc.Top + maxHeight
		// 				}
		// 			}
		// 		}
		// 		SetPadding(edge.Rect{})
		// 	} else {
		// 		// This is needed to workaround the resize flickering in frameless mode with WindowDecorations
		// 		// See: https://stackoverflow.com/a/6558508
		// 		// The workaround originally suggests to decrese the bottom 1px, but that seems to bring up a thin
		// 		// white line on some Windows-Versions, due to DrawBackground using also this reduces ClientSize.
		// 		// Increasing the bottom also worksaround the flickering but we would loose 1px of the WebView content
		// 		// therefore let's pad the content with 1px at the bottom.
		// 		rgrc.Bottom += 1
		// 		SetPadding(edge.Rect{Bottom: 1})
		// 	}
		// 	return 0
		// }
	default:
		return w32.DefWindowProc(hwnd, msg, wParam, lParam)
	}
	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}
func SetPadding(padding edge.Rect) {
	// Skip SetPadding if window is being minimized to prevent flickering
	// 如果窗口正在最小化,跳过设置padding以防止闪烁
	// if w.isMinimizing {
	// 	return
	// }
	if g_chromium != nil {
		g_chromium.SetPadding(padding)
	}
}
func ScaleWithDPI(pixels int, dpi uint) int {
	return (pixels * int(dpi)) / 96
}

type MONITOR_DPI_TYPE int32

func GetDPIForMonitor(hmonitor w32.HMONITOR, dpiType MONITOR_DPI_TYPE, dpiX *uint, dpiY *uint) uintptr {
	ret, _, _ := syscall.NewLazyDLL("shcore.dll").NewProc("GetDpiForMonitor").Call(
		uintptr(hmonitor),
		uintptr(dpiType),
		uintptr(unsafe.Pointer(dpiX)),
		uintptr(unsafe.Pointer(dpiY)))

	return ret
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

func Center(hwnd w32.HWND) {

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

/**
 * why no effect..?
 */
func setChromiumBackground(chromium *edge.Chromium, r uint8, g uint8, b uint8, webviewIsTransparent bool) {
	controller := chromium.GetController()
	controller2 := controller.GetICoreWebView2Controller2()

	backgroundCol := edge.COREWEBVIEW2_COLOR{
		A: 255,
		R: r,
		G: g,
		B: b,
	}

	// WebView2 only has 0 and 255 as valid values.
	if backgroundCol.A > 0 && backgroundCol.A < 255 {
		backgroundCol.A = 255
	}

	if webviewIsTransparent {
		backgroundCol.A = 0
	}

	err := controller2.PutDefaultBackgroundColor(backgroundCol)
	if err != nil {
		log.Fatal(err)
	}

}
