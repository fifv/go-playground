package main

import "github.com/gonutz/w32/v2"

func main() {
	// w32.MessageBox(0, "Hello", "Title", w32.MB_OK)
    w32.PostMessage(w32.HWND_BROADCAST, w32.WM_SYSCOMMAND, w32.SC_MONITORPOWER, 2);

}