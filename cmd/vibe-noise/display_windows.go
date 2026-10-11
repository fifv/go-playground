package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func enableTerminalOutput(file *os.File) (bool, func()) {
	handle := windows.Handle(file.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false, func() {}
	}
	if err := windows.SetConsoleMode(handle, mode|windows.ENABLE_PROCESSED_OUTPUT|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return false, func() {}
	}
	return true, func() { _ = windows.SetConsoleMode(handle, mode) }
}
