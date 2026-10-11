package main

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var readConsoleInput = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")

// INPUT_RECORD with its 16-byte union represented as KEY_EVENT_RECORD.
// Other event types are ignored before reading the key fields.
type inputRecord struct {
	EventType                               uint16
	_                                       uint16
	KeyDown                                 int32
	RepeatCount, VirtualKey, ScanCode, Char uint16
	ControlKeyState                         uint32
}

type control struct {
	stop        bool
	togglePause bool
	volumeSteps int
}

func (r inputRecord) control() control {
	if r.EventType != 1 || r.KeyDown == 0 {
		return control{}
	}
	steps := int(max(1, r.RepeatCount))
	if r.ControlKeyState&0x0003 != 0 { // LEFT_ALT_PRESSED | RIGHT_ALT_PRESSED
		steps *= 10
	}
	switch r.VirtualKey {
	case 0x0D: // VK_RETURN
		return control{stop: true}
	case 0x20: // VK_SPACE
		return control{togglePause: true}
	case 0x26: // VK_UP
		return control{volumeSteps: steps}
	case 0x28: // VK_DOWN
		return control{volumeSteps: -steps}
	}
	return control{}
}

type consoleInput struct {
	handle       windows.Handle
	previousMode uint32
	spaceDown    bool
}

func openConsoleInput() (*consoleInput, error) {
	c := &consoleInput{handle: windows.Handle(os.Stdin.Fd())}
	if err := windows.GetConsoleMode(c.handle, &c.previousMode); err != nil {
		// Redirected stdin is not an interactive console. Do not wait for input.
		return nil, nil
	}
	// Keep Ctrl+C handling enabled, prevent echo/line buffering, and prevent
	// Quick Edit selection from suspending playback. Restore on every exit path.
	mode := c.previousMode | windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_EXTENDED_FLAGS
	mode &^= windows.ENABLE_LINE_INPUT | windows.ENABLE_ECHO_INPUT | windows.ENABLE_QUICK_EDIT_MODE | windows.ENABLE_VIRTUAL_TERMINAL_INPUT
	if err := windows.SetConsoleMode(c.handle, mode); err != nil {
		return nil, fmt.Errorf("enable keyboard controls: %w", err)
	}
	return c, nil
}

func (c *consoleInput) close() error {
	if c == nil {
		return nil
	}
	if err := windows.SetConsoleMode(c.handle, c.previousMode); err != nil {
		return fmt.Errorf("restore console input: %w", err)
	}
	return nil
}

func (c *consoleInput) poll() ([]control, error) {
	if c == nil {
		return nil, nil
	}
	var count uint32
	if err := windows.GetNumberOfConsoleInputEvents(c.handle, &count); err != nil {
		return nil, fmt.Errorf("poll keyboard: %w", err)
	}
	if count == 0 {
		return nil, nil
	}
	var records [32]inputRecord
	var read uint32
	ok, _, err := readConsoleInput.Call(uintptr(c.handle), uintptr(unsafe.Pointer(&records[0])), uintptr(min(count, uint32(len(records)))), uintptr(unsafe.Pointer(&read)))
	if ok == 0 {
		return nil, fmt.Errorf("read keyboard: %w", err)
	}
	var controls []control
	for _, record := range records[:read] {
		// Space is a toggle, so holding it must not repeatedly pause/resume.
		if record.EventType == 1 && record.VirtualKey == 0x20 {
			wasDown := c.spaceDown
			c.spaceDown = record.KeyDown != 0
			if wasDown || !c.spaceDown {
				continue
			}
		}
		if event := record.control(); event.stop || event.togglePause || event.volumeSteps != 0 {
			controls = append(controls, event)
		}
	}
	return controls, nil
}
