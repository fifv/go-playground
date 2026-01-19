package main

import (
	"fmt"
	"log"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
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

	{
		ports, err := enumerator.GetDetailedPortsList()
		if err != nil {
			log.Fatal(err)
		}
		if len(ports) == 0 {
			fmt.Println("No serial ports found!")
			return
		}
		for _, port := range ports {
			fmt.Printf("Found port: %s\n", port.Name)
			fmt.Printf("   Product: %s\n", port.Product)
			if port.IsUSB {
				fmt.Printf("   USB ID     %s:%s\n", port.VID, port.PID)
				fmt.Printf("   USB serial %s\n", port.SerialNumber)
			}
		}
	}

	var port serial.Port
	for {
		port, err = serial.Open("COM21", &serial.Mode{BaudRate: 115200})
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println("Connected To COM  ")
			break
		}
		time.Sleep(time.Millisecond * 300)
	}

	n, err := port.Write([]byte("fasdfa"))
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
