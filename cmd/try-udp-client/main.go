package main

import (
	// "bufio"
	"encoding/json"
	"math/rand"
	"time"

	// "fmt"
	"net"

	"github.com/charmbracelet/log"
)

type CenterPoint struct {
	Id int     `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
}

func main() {
	conn, err := net.Dial("udp", "localhost:8337")
	if err != nil {
		panic(err)
	}
	defer conn.Close()


	for i := range 300 {
		point := CenterPoint{
			Id: rand.Intn(9),
			X:  (rand.Float32()*100 + float32(i*4)) * 2,
			Y:  (rand.Float32()*100 + float32(i*4)) * 1,
		}
		message, err := json.Marshal(point)
		if err != nil {
			panic(err)
		}
		conn.Write([]byte(string(message) + "\n"))
		log.Infof("Sent to server: %s", string(message))
		time.Sleep(time.Millisecond * 30)
	}
}