package main

import (
	"io"
	"net"
	"sync"

	// "sync"
	"time"

	"github.com/charmbracelet/log"
)

func main() {
	// var wg sync.WaitGroup
	// wg.Go(tcpServer)
	// wg.Go(tcpClient)
	// wg.Wait()

	go tcpServer()
	tcpClient()
	time.Sleep(time.Millisecond * 100)
}

func tcpServer() {
	listener, err := net.Listen("tcp", ":8888")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		log.Infof("Accepted connection from %v", conn.RemoteAddr())

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1<<28)
	totalRxBytes := uint64(0)

	for {
		n, err := conn.Read(buf)
		if n > 0 {
			totalRxBytes += uint64(n)
		}
		if err != nil {
			if err == io.EOF {
				log.Infof("Received: %v KB", totalRxBytes/1024)
			} else {
				log.Infof("Error: %v", err)
			}
			return
		}
	}
}

func tcpClient() {
	conn, err := net.Dial("tcp", "localhost:8888")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	/**
	 * payload size affect speed significantly, large is fast
	 */
	payload := make([]byte, 1<<20)

	var wg sync.WaitGroup
	t1 := time.Now()

	/**
	 * seems conn.Write() has lock, multiple goroutines don't help, even slower
	 */
	for range 16 {
		wg.Go(func() {
		})
	}
	for {
		_, err := conn.Write(payload)
		if err != nil {
			panic(err)
		}
		if time.Since(t1).Milliseconds() > 3000 {
			break
		}
	}
	wg.Wait()
}
