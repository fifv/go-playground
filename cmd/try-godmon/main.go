/**
 * Something like nodemon / air
 */

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	// "github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows"
)

/**
 * This written by ChatGPT, seems works, but after attaching to child process, the console is lost
 */
func main() {
	for {
		cmd := exec.Command("go", "run", "./cmd/try-serial")

		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		fmt.Println("Starting child...")
		if err := cmd.Start(); err != nil {
			panic(err)
		}

		time.Sleep(3 * time.Second)

		fmt.Println("Sending Ctrl+Break...")
		err := sendCtrlBreak(cmd.Process.Pid)
		if err != nil {
			fmt.Println("Signal error:", err)
		}

		err = cmd.Wait()
		fmt.Println("Child exited:", err)

		time.Sleep(time.Second)
	}
}

func sendCtrlBreak(pid int) error {
	// Detach from current console
	FreeConsole()

	// Attach to target console
	AttachConsole(pid)

	defer func() {
		FreeConsole()
		AttachConsole(os.Getpid())
	}()

	return windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, 0)
}
