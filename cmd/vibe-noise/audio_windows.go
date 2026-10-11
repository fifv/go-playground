package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winmm                  = windows.NewLazySystemDLL("winmm.dll")
	waveOutGetNumDevs      = winmm.NewProc("waveOutGetNumDevs")
	waveOutGetDevCaps      = winmm.NewProc("waveOutGetDevCapsW")
	waveOutOpen            = winmm.NewProc("waveOutOpen")
	waveOutPause           = winmm.NewProc("waveOutPause")
	waveOutRestart         = winmm.NewProc("waveOutRestart")
	waveOutPrepareHeader   = winmm.NewProc("waveOutPrepareHeader")
	waveOutUnprepareHeader = winmm.NewProc("waveOutUnprepareHeader")
	waveOutWrite           = winmm.NewProc("waveOutWrite")
	waveOutReset           = winmm.NewProc("waveOutReset")
	waveOutClose           = winmm.NewProc("waveOutClose")
	waveOutGetErrorText    = winmm.NewProc("waveOutGetErrorTextW")
)

type waveFormat struct {
	Tag, Channels                        uint16
	SamplesPerSec, AvgBytesPerSec        uint32
	BlockAlign, BitsPerSample, ExtraSize uint16
}

type waveCaps struct {
	Manufacturer, Product uint16
	Version               uint32
	Name                  [32]uint16
	Formats               uint32
	Channels, Reserved    uint16
	Support               uint32
}

type waveHeader struct {
	Data                        *int16
	BufferLength, BytesRecorded uint32
	User                        uintptr
	Flags, Loops                uint32
	Next                        *waveHeader
	Reserved                    uintptr
}

const (
	waveMapper   = uintptr(0xffffffff)
	whdrDone     = 1
	bufferFrames = sampleRate / 20 // Four 50 ms buffers per device.
)

type outputDevice struct {
	id   uintptr
	name string
}

type audioBuffer struct {
	header   waveHeader
	samples  []int16
	volume   float64
	pin      runtime.Pinner
	prepared bool
}

type player struct {
	device     outputDevice
	handle     uintptr
	buffers    []*audioBuffer
	generator  *generator
	nextBuffer int
}

// On a driver cleanup failure, retain pinned buffers until process exit rather
// than release memory that the driver may still be using.
var retainedBuffers []*audioBuffer

//go:uintptrescapes
func mmCall(proc *windows.LazyProc, args ...uintptr) error {
	code, _, _ := proc.Call(args...)
	if code == 0 {
		return nil
	}
	var message [256]uint16
	waveOutGetErrorText.Call(code, uintptr(unsafe.Pointer(&message[0])), uintptr(len(message)))
	return fmt.Errorf("%s: %s (code %d)", proc.Name, windows.UTF16ToString(message[:]), code)
}

func outputDevices() ([]outputDevice, error) {
	count, _, _ := waveOutGetNumDevs.Call()
	var devices []outputDevice
	for id := uintptr(0); id < count; id++ {
		var caps waveCaps
		if err := mmCall(waveOutGetDevCaps, id, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps)); err != nil {
			printDiagnostic("SKIPPED", fmt.Sprintf("Device %d: %v", id, err))
			continue
		}
		devices = append(devices, outputDevice{id, windows.UTF16ToString(caps.Name[:])})
	}
	if len(devices) == 0 {
		return nil, errors.New("no audio output devices available")
	}
	return devices, nil
}

func openPlayer(device outputDevice, o options, seed uint64) (_ *player, err error) {
	p := &player{device: device, generator: newGenerator(o, seed)}
	format := waveFormat{Tag: 1, Channels: 2, SamplesPerSec: sampleRate, AvgBytesPerSec: sampleRate * 4, BlockAlign: 4, BitsPerSample: 16}
	if err = mmCall(waveOutOpen, uintptr(unsafe.Pointer(&p.handle)), device.id, uintptr(unsafe.Pointer(&format)), 0, 0, 0); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, p.close())
		}
	}()
	if err = mmCall(waveOutPause, p.handle); err != nil {
		return nil, err
	}
	for range 4 {
		b := &audioBuffer{samples: make([]int16, bufferFrames*2)}
		b.header.Data = &b.samples[0]
		b.header.BufferLength = uint32(len(b.samples) * 2)
		// WinMM retains both pointers after the syscall returns.
		b.pin.Pin(&b.samples[0])
		b.pin.Pin(&b.header)
		p.buffers = append(p.buffers, b)
		if err = mmCall(waveOutPrepareHeader, p.handle, uintptr(unsafe.Pointer(&b.header)), unsafe.Sizeof(b.header)); err != nil {
			return nil, err
		}
		b.prepared = true
		if err = p.write(b); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *player) write(b *audioBuffer) error {
	p.generator.fill(b.samples)
	b.volume = p.generator.volume
	return mmCall(waveOutWrite, p.handle, uintptr(unsafe.Pointer(&b.header)), unsafe.Sizeof(b.header))
}

// prepareResume runs while paused. A queued block already contains its gain,
// so changing the generator alone cannot change what WinMM plays on resume.
func (p *player) prepareResume() error {
	stale := false
	for _, b := range p.buffers {
		if b.header.Flags&whdrDone == 0 && b.volume != p.generator.volume {
			stale = true
			break
		}
	}
	if !stale {
		return nil
	}
	// Reset returns ownership of every queued buffer. Pause again before
	// writing, since the first write after reset would otherwise start audio.
	if err := mmCall(waveOutReset, p.handle); err != nil {
		return err
	}
	if err := mmCall(waveOutPause, p.handle); err != nil {
		return err
	}
	p.nextBuffer = 0
	for _, b := range p.buffers {
		if err := p.write(b); err != nil {
			return err
		}
	}
	return nil
}

func (p *player) setPaused(paused bool) error {
	if paused {
		return mmCall(waveOutPause, p.handle)
	}
	if err := p.prepareResume(); err != nil {
		return err
	}
	return mmCall(waveOutRestart, p.handle)
}

func (p *player) refill() error {
	// Preserve queue order even if several buffers finish between polls.
	for range len(p.buffers) {
		b := p.buffers[p.nextBuffer]
		if b.header.Flags&whdrDone == 0 {
			break
		}
		if err := p.write(b); err != nil {
			return err
		}
		p.nextBuffer = (p.nextBuffer + 1) % len(p.buffers)
	}
	return nil
}

func (p *player) close() error {
	var errs []error
	if err := mmCall(waveOutReset, p.handle); err != nil {
		errs = append(errs, err)
	}
	for _, b := range p.buffers {
		if b.prepared {
			if err := mmCall(waveOutUnprepareHeader, p.handle, uintptr(unsafe.Pointer(&b.header)), unsafe.Sizeof(b.header)); err != nil {
				errs = append(errs, err)
				retainedBuffers = append(retainedBuffers, b)
				continue
			}
		}
		b.pin.Unpin()
	}
	if err := mmCall(waveOutClose, p.handle); err != nil {
		errs = append(errs, err)
	}
	p.buffers = nil
	return errors.Join(errs...)
}

func play(ctx context.Context, o options) (err error) {
	ui := newTerminalUI(os.Stdout)
	var started time.Time
	defer func() {
		if !started.IsZero() {
			ui.stopped(started, err)
		}
		ui.restore()
	}()
	devices := []outputDevice{{waveMapper, "current Windows default output"}}
	if o.device == "all" || o.list {
		devices, err = outputDevices()
		if err != nil {
			return err
		}
	}
	resolveOutputNames(devices)
	if o.list {
		var names []string
		for _, d := range devices {
			names = append(names, fmt.Sprintf("%2d  %s", d.id, d.name))
		}
		ui.devices(names)
		return nil
	}
	keyboard, err := openConsoleInput()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, keyboard.close()) }()
	var players []*player
	defer func() {
		for _, p := range players {
			err = errors.Join(err, p.close())
		}
	}()
	// Every device renders the same sequence independently, allowing different
	// hardware clock rates without sharing mutable generator state.
	seed := rand.Uint64()
	for _, d := range devices {
		if ctx.Err() != nil {
			return nil
		}
		p, openErr := openPlayer(d, o, seed)
		if openErr != nil {
			printDiagnostic("SKIPPED", fmt.Sprintf("%s: %v", d.name, openErr))
			continue
		}
		players = append(players, p)
	}
	if len(players) == 0 {
		return errors.New("could not open any audio output device")
	}
	var names []string
	for _, p := range players {
		if err := mmCall(waveOutRestart, p.handle); err != nil {
			return fmt.Errorf("start %s: %w", p.device.name, err)
		}
		names = append(names, p.device.name)
	}
	ui.playback(o, names, keyboard != nil)
	started = time.Now()
	if o.duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.duration)
		defer cancel()
	}
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	paused := false
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			controls, err := keyboard.poll()
			if err != nil {
				return err
			}
			for _, event := range controls {
				if event.stop {
					return nil
				}
				if event.togglePause {
					paused = !paused
					for _, p := range players {
						if err := p.setPaused(paused); err != nil {
							return fmt.Errorf("change playback state for %s: %w", p.device.name, err)
						}
					}
					ui.paused = paused
				}
				o.volume = adjustVolume(o.volume, event.volumeSteps)
				for _, p := range players {
					p.generator.volume = o.volume
				}
				ui.volume(o.volume)
			}
			if paused {
				continue
			}
			for _, p := range players {
				if err := p.refill(); err != nil {
					return fmt.Errorf("play %s: %w", p.device.name, err)
				}
			}
		}
	}
}
