package main

import (
	"fmt"
	"log"
	"strconv"
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
		// port, err = serial.Open("COM5", &serial.Mode{BaudRate: 921600})
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println("Connected To COM  ")

			break
		}
		time.Sleep(time.Millisecond * 300)
	}

	// n, err := port.Write([]byte("fasdfa"))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("Sent %v bytes\n", n)

	for {
		// buff := make([]byte, 2048)
		/**
		 * port.Read(buff) must use a []byte with length > 0 (capacity won't be used)
		 */
		// n, err := port.Read(buff)
		// rxBuff, err := ReadSerialPortToIdle(port)
		rxBuff, err := ReadSerialPortToIdle(port)
		if err != nil {
			if portErr, ok := err.(*serial.PortError); ok {
				/* Cast the error to serial.PortError and check the detail */
				if portErr.Code() == serial.PortClosed {
					fmt.Println("Closed!!!")
					break
				} else {
					fmt.Println("Err", err)
				}
			} else {
				/* Maybe the error is not serial.PortError */
				log.Fatal(err)
			}

			break
		}
		if len(rxBuff) == 0 {
			fmt.Println("\nEOF")
			break
		}
		fmt.Printf("<<<<<<[%v] %v\n", len(rxBuff), strconv.QuoteToASCII(string(rxBuff)))
	}
}

/**
 * 
 * the "perfert" readToIdle
 * 1. abuse allocations, so arbitary number of bytes is okay, with beautiful allocated []byte as result
 * 2. no poll, block until data come
 * 3. win32 will return only 1 byte on waking, and if data exceeds readBuf, multiple Read() with timeout is used
 * 
 * by @Fifv
 */
func ReadSerialPortToIdle(port serial.Port) ([]byte, error) {
	tmpReadBuf := make([]byte, 1024)
	resultBuf := make([]byte, 0, 1024)

	/* first read, with block */
	{
		/* block, until first bytes come */
		port.SetReadTimeout(-1)
		readCount, err := port.Read(tmpReadBuf)
		/* here n==0 may means EOF?  */
		if err != nil || readCount == 0 {
			return nil, err
		}
		resultBuf = append(resultBuf, tmpReadBuf[:readCount]...)
	}

	/* all next reads, until idle (i.e. timeout) */
	/* if timeout, will got n==0, err==nil */
	port.SetReadTimeout(time.Microsecond * 10)
	for {
		readCount, err := port.Read(tmpReadBuf)
		if err != nil {
			return nil, err
		}
		/* timeout, means already readToIdle */
		if readCount == 0 {
			return resultBuf, nil
		}
		/* still reading some data */
		resultBuf = append(resultBuf, tmpReadBuf[:readCount]...)
	}
}
