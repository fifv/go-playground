package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

func main() {
	fmt.Println("hi, 123456789123456789012312312323frfff34444444f4")
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
			// return
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

	comPath := "COM31"
	baudRate := 115200
	if len(os.Args) >= 2 {
		comPath = os.Args[1]
	}
	if len(os.Args) >= 3 {
		baudRate, err = strconv.Atoi(os.Args[2])
		if err != nil {
			panic("wrong arg 2, should be baudRate: int")
		}
	}

	fmt.Println("--------------------")
	var port serial.Port
	for {
		port, err = serial.Open(comPath, &serial.Mode{BaudRate: baudRate})
		// port, err = serial.Open("COM31", &serial.Mode{BaudRate: 115200})
		// port, err = serial.Open("COM5", &serial.Mode{BaudRate: 921600})
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println("Connected To", comPath)
			break
		}
		time.Sleep(time.Millisecond * 300)
	}
	fmt.Println("--------------------")

	if len(os.Args) >= 4 {
		txBuff := make([]byte, 0, len(os.Args)-3)
		for _, arg := range os.Args[3:] {
			byteArg := strings.TrimPrefix(strings.TrimPrefix(arg, "0x"), "0X")
			v, err := strconv.ParseUint(byteArg, 16, 8)
			if err != nil {
				log.Fatalf("wrong data byte %q, should be hex byte like AA or 55", arg)
			}
			txBuff = append(txBuff, byte(v))
		}

		n, err := WriteAllSerialPort(port, txBuff)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Sent %v bytes: % x\n", n, txBuff)
		return
	}

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
		// fmt.Printf("<<<<<<[%v] %v\n", len(rxBuff), strconv.QuoteToASCII(string(rxBuff)))
		fmt.Printf("<<<<<<[%v] %v\n", len(rxBuff), fmt.Sprintf("% x", rxBuff))
	}
}

func WriteAllSerialPort(port serial.Port, buff []byte) (int, error) {
	total := 0
	for total < len(buff) {
		n, err := port.Write(buff[total:])
		total += n
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, fmt.Errorf("serial write made no progress after %d/%d bytes", total, len(buff))
		}
	}
	return total, nil
}

//   gfosjstjust some test that the air is working.... working... asdfsdfsd
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
		port.SetReadTimeout(serial.NoTimeout)
		readCount, err := port.Read(tmpReadBuf)
		/* here n==0 may means EOF?  */
		if err != nil || readCount == 0 {
			return nil, err
		}
		resultBuf = append(resultBuf, tmpReadBuf[:readCount]...)
	}

	/* all next reads, until idle (i.e. timeout) */
	/* if timeout, will got n==0, err==nil */
	if runtime.GOOS == "windows" {
		port.SetReadTimeout(time.Millisecond * 1)
	} else {
		port.SetReadTimeout(time.Microsecond * 10)
	}
	// if err != nil {
	// 	panic("the timeout value is not valid")
	// }
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
