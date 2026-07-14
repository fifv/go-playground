package main

import "golang.org/x/sys/windows"

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procFreeConsole       = kernel32.NewProc("FreeConsole")
	procAttachConsole     = kernel32.NewProc("AttachConsole")
	procGenerateCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

func AttachConsole(pid int) error {
	r1, _, err := procAttachConsole.Call(uintptr(pid))
	if r1 == 0 {
		return err
	}
	return nil
}

func FreeConsole() error {
	r1, _, err := procFreeConsole.Call()
	if r1 == 0 {
		return err
	}
	return nil
}
