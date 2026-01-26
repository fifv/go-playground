package main

import (
	"fmt"
	"github.com/jacobsa/go-serial/serial"
	// "github.com/Plantiga/simple-go-serial/serial"
	"log"
)

func main() {
	// Set up options.
	options := serial.OpenOptions{
		PortName:              "COM21",
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       10000000,
		InterCharacterTimeout: 10000000,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	// Make sure to close it later.
	defer port.Close()

	// Write 4 bytes to the port.
	for {

		b := make([]byte, 1024)
		n, err := port.Read(b)
		if err != nil {
			log.Fatalf("port.Write: %v", err)
		}
		fmt.Println("Read", n, "bytes.", b[:n])
	}

}
