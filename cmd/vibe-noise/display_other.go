//go:build !windows

package main

import (
	"golang.org/x/term"
	"os"
)

func enableTerminalOutput(file *os.File) (bool, func()) {
	return term.IsTerminal(int(file.Fd())), func() {}
}
