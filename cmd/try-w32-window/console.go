package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	kernel32          = windows.NewLazySystemDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procAllocConsole  = kernel32.NewProc("AllocConsole")
)

const ATTACH_PARENT_PROCESS = ^uint32(0)

func CreateConsole() {
	procAllocConsole.Call()

	f, _ := os.OpenFile("CONOUT$", os.O_WRONLY, 0644)
	os.Stdout = f
	os.Stderr = f
}
func AttachConsole(dwParentProcess uint32) (ok bool) {
	r0, _, _ := syscall.Syscall(procAttachConsole.Addr(), 1, uintptr(dwParentProcess), 0, 0)
	ok = bool(r0 != 0)
	return
}

func TryAttachToParentConsole() {

	// r1, _, err := procAttachConsole.Call(ATTACH_PARENT_PROCESS)
	// if r1 != 0 {
	// 	fmt.Println("AttachConsole Successfully")
	// } else {
	// 	fmt.Println("AttachConsole Failed")
	// }
	// return err

	ok := AttachConsole(ATTACH_PARENT_PROCESS)
	if ok {
		fmt.Println("Okay, attached")
	}
}