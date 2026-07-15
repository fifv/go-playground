//go:build windows

/**
 * Fully vibed loopback to websocket
 */
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/go-ole/go-ole"
	"github.com/gorilla/websocket"
	"github.com/moutend/go-wca/pkg/wca"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	var clientsMu sync.RWMutex
	clients := make(map[*websocket.Conn]struct{})

	http.HandleFunc("/wscom", handleWebSocket(clients, &clientsMu))

	payloadCh := make(chan []byte, 32)

	go func() {
		for {
			payload := <-payloadCh

			clientsMu.RLock()
			for conn := range clients {
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					_ = conn.Close()
					delete(clients, conn)
				}
			}
			clientsMu.RUnlock()
		}
	}()
	go func() {
		if err := captureLoopback(func(payload []byte) {
			msg, err := constructWsComMessage(payload)
			if err != nil {
				log.Error(err)
				return
			}
			payloadCh <- msg
		}); err != nil {
			log.Fatal(err)
		}
	}()

	log.Info("WebSocket: ws://127.0.0.1:3304/wscom")
	log.Fatal(http.ListenAndServe("127.0.0.1:3304", nil))
}

func handleWebSocket(clients map[*websocket.Conn]struct{}, clientsMu *sync.RWMutex) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(writer, req, nil)
		if err != nil {
			log.Debug("WebSocket upgrade:", err)
			return
		}

		clientsMu.Lock()
		clients[conn] = struct{}{}
		clientsMu.Unlock()

		log.Debugf("connected: %v", conn.RemoteAddr())

		// Wait for disconnect.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}

		clientsMu.Lock()
		delete(clients, conn)
		clientsMu.Unlock()

		_ = conn.Close()
		log.Debugf("disconnected: %v", conn.RemoteAddr())
	}
}

func constructWsComMessage(data []byte) ([]byte, error) {
	integerData := make([]int, len(data))
	for i, value := range data {
		integerData[i] = int(value)
	}

	payload, err := json.Marshal(struct {
		Event string `json:"event"`
		Data  struct {
			Payload []int `json:"payload"`
		} `json:"data"`
	}{
		Event: "com-rx",
		Data: struct {
			Payload []int `json:"payload"`
		}{
			Payload: integerData,
		},
	})
	if err != nil {
		log.Debug("encode WebSocket message:", err)
		return nil, err
	}

	return payload, nil
}

func captureLoopback(output func([]byte)) error {
	// COM initialization and WASAPI calls must remain on the same OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return fmt.Errorf("CoInitializeEx: %w", err)
	}
	defer ole.CoUninitialize()

	var enumerator *wca.IMMDeviceEnumerator

	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&enumerator,
	); err != nil {
		return fmt.Errorf("create device enumerator: %w", err)
	}
	defer enumerator.Release()

	var device *wca.IMMDevice

	if err := enumerator.GetDefaultAudioEndpoint(
		wca.ERender,
		wca.EMultimedia,
		&device,
	); err != nil {
		return fmt.Errorf("get default output device: %w", err)
	}
	defer device.Release()

	var audioClient *wca.IAudioClient

	if err := device.Activate(
		wca.IID_IAudioClient,
		wca.CLSCTX_ALL,
		nil,
		&audioClient,
	); err != nil {
		return fmt.Errorf("activate audio client: %w", err)
	}
	defer audioClient.Release()

	var format *wca.WAVEFORMATEX

	if err := audioClient.GetMixFormat(&format); err != nil {
		return fmt.Errorf("get mix format: %w", err)
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(format)))

	log.Infof(
		"audio: %d Hz, %d channels, %d bits, block=%d bytes",
		format.NSamplesPerSec,
		format.NChannels,
		format.WBitsPerSample,
		format.NBlockAlign,
	)
	if format.NChannels == 0 || format.NBlockAlign%format.NChannels != 0 {
		return fmt.Errorf(
			"unsupported audio format: %d channels, block alignment %d",
			format.NChannels,
			format.NBlockAlign,
		)
	}
	bytesPerSample := int(format.NBlockAlign / format.NChannels)

	const bufferDuration = wca.REFERENCE_TIME(10000000) // 1 second

	if err := audioClient.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		wca.AUDCLNT_STREAMFLAGS_LOOPBACK,
		bufferDuration,
		0,
		format,
		nil,
	); err != nil {
		return fmt.Errorf("initialize loopback: %w", err)
	}

	var captureClient *wca.IAudioCaptureClient

	if err := audioClient.GetService(
		wca.IID_IAudioCaptureClient,
		&captureClient,
	); err != nil {
		return fmt.Errorf("get capture client: %w", err)
	}
	defer captureClient.Release()

	if err := audioClient.Start(); err != nil {
		return fmt.Errorf("start capture: %w", err)
	}
	defer audioClient.Stop()

	log.Info("WASAPI loopback capture started")

	for {
		var packetFrames uint32

		if err := captureClient.GetNextPacketSize(&packetFrames); err != nil {
			return fmt.Errorf("get next packet size: %w", err)
		}

		if packetFrames == 0 {
			time.Sleep(2 * time.Millisecond)
			continue
		}

		for packetFrames > 0 {
			var (
				data           *byte
				frames         uint32
				flags          uint32
				devicePosition uint64
				qpcPosition    uint64
			)

			if err := captureClient.GetBuffer(
				&data,
				&frames,
				&flags,
				&devicePosition,
				&qpcPosition,
			); err != nil {
				return fmt.Errorf("capture GetBuffer: %w", err)
			}

			frameSize := int(format.NBlockAlign)
			byteCount := int(frames) * frameSize
			packet := make([]byte, int(frames)*bytesPerSample)

			// AUDCLNT_BUFFERFLAGS_SILENT means data may be nil.
			if flags&wca.AUDCLNT_BUFFERFLAGS_SILENT == 0 && data != nil {
				src := unsafe.Slice(data, byteCount)
				for frame := 0; frame < int(frames); frame++ {
					srcStart := frame * frameSize
					dstStart := frame * bytesPerSample
					copy(
						packet[dstStart:dstStart+bytesPerSample],
						src[srcStart:srcStart+bytesPerSample],
					)
				}
			}

			if err := captureClient.ReleaseBuffer(frames); err != nil {
				return fmt.Errorf("capture ReleaseBuffer: %w", err)
			}

			if len(packet) > 0 {
				output(packet)
			}

			if err := captureClient.GetNextPacketSize(&packetFrames); err != nil {
				return fmt.Errorf("get next packet size: %w", err)
			}
		}
	}
}

func init() {
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.SetCallerFormatter(func(file string, line int, fn string) string {
		cwd, _ := os.Getwd()
		rel, _ := filepath.Rel(cwd, file)
		return fmt.Sprintf("%s:%d", rel, line)
	})
	styles := log.DefaultStyles()
	styles.Timestamp = lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")) /* dark grey */
	log.SetStyles(styles)
}
