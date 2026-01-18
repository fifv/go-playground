package main

import (
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"
)

func main() {
	t1 := time.Now()
	ports, err := serial.GetPortsList()
	fmt.Println(time.Since(t1).Milliseconds(), "ms", "GetPortsList")
	if err != nil {
		log.Fatal(err)
	}
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}

	port, err := serial.Open("COM21", &serial.Mode{BaudRate: 115200})
	if err != nil {
		log.Fatal(err)
	}

	n, err := port.Write([]byte("asfdf112fffffffff"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sent %v bytes\n", n)

	buff := make([]byte, 100)
	for {
		n, err := port.Read(buff)
		if err != nil {
			log.Fatal(err)
			break
		}
		if n == 0 {
			fmt.Println("\nEOF")
			break
		}
		fmt.Printf("<<<<<<1 %v\n", string(buff[:n]))
	}
}
