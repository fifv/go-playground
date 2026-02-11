package main

import (
	"bufio"
	"encoding/json"
	"net"

	// "strings"

	"github.com/charmbracelet/log"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("Error accepting conn: ", err)
			continue
		}
		log.Info("Accepted connection from ", conn.RemoteAddr())

		go handleConnection(conn)
	}
}

type CenterPoint struct {
	Id int     `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	log.Infof("Handling connection from %v", conn.RemoteAddr())

	rxBytesCount := 0
	reader := bufio.NewReader(conn)
	/**
	 * defer a func makes the captured value read when executed, not when deferred
	 */
	defer func() {
		log.Infof("Connection from %v closed. Total bytes received: %d", conn.RemoteAddr(), rxBytesCount)
	}()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Errorf("Error reading from connection: %v", err)
			return
		}

		log.Infof("Received: %v", line)

		var point CenterPoint
		err = json.Unmarshal([]byte(line), &point)
		if err != nil {
			log.Errorf("Error unmarshaling JSON: %v", err)
			return
		}
		log.Infof("Parsed CenterPoint: %+v", point)

		// rxBytesCount += len(line)
		// _, err = conn.Write([]byte("Echo: " + strings.ToUpper(line)))
		// if err != nil {
		// 	log.Errorf("Error writing to connection: %v", err)
		// 	return
		// }
	}
}
