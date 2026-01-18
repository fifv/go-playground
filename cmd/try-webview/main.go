package main

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"

	// "log"
	"github.com/jchv/go-webview2"
	// "github.com/gonutz/w32"
)

func main() {
	fmt.Println("asdf")
	applicationInit()

	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     true,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Minimal webview example",
			Width:  0,
			Height: 0,
			// Width:  1280,
			// Height: 720,
			IconId: 2, // icon resource id
			Center: false,
		},
	})
	if w == nil {
		log.Fatalln("Failed to load webview.")
	}
	defer w.Destroy()
	SetTheme(uintptr(w.Window()), true)
	// w.SetSize(1280, 720, webview2.HintNone)
	// w32.ShowWindow(w32.HWND(w.Window()), w32.SW_MAXIMIZE)

	w.Navigate("https://google.com")
	w.Run()
}

func applicationInit() error {
	status, r, err := syscall.NewLazyDLL("user32.dll").NewProc("SetProcessDPIAware").Call()
	if status == 0 {
		return fmt.Errorf("exit status %d: %v %v", status, r, err)
	}
	return nil
}

type DWMWINDOWATTRIBUTE int32

func dwmSetWindowAttribute(hwnd uintptr, dwAttribute DWMWINDOWATTRIBUTE, pvAttribute unsafe.Pointer, cbAttribute uintptr) {
	ret, _, err := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute").Call(
		hwnd,
		uintptr(dwAttribute),
		uintptr(pvAttribute),
		cbAttribute)
	if ret != 0 {
		_ = err
		// println(err.Error())
	}
}
func SetTheme(hwnd uintptr, useDarkMode bool) {
	const DwmwaUseImmersiveDarkMode DWMWINDOWATTRIBUTE = 20
	attr := DwmwaUseImmersiveDarkMode
	var winDark int32
	if useDarkMode {
		winDark = 1
	}
	dwmSetWindowAttribute(hwnd, attr, unsafe.Pointer(&winDark), unsafe.Sizeof(winDark))

}
