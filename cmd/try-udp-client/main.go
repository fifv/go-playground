package main

import (
	// "bufio"
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
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
type SomeCenterPoint struct {
	point  CenterPoint
	exists bool
}

func main() {
	zntdDisplay9Files()
}
func zntdSendDummyPoints() {
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
func zntdDisplay9Files() {
	var wg sync.WaitGroup

	for i := range 9 {
		wg.Go(func() {
			// sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/normal-5/cut_8%v_processed.json", i), i)
			// sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/normal-5-2/cut_8%v_processed.json", i), i)
			// sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/speed-26/cut_8%v_processed.json", i), i)
			// sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/free-26/cut_8%v_processed.json", i), i)
			// sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/extreme-5/cut_8%v_processed.json", i), i)
			// sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/extreme/cut_8%v_processed.json", i), i)
			sendPoints(fmt.Sprintf("D:/Downloads/钉钉/location/normal-5(1)/cut_8%v_processed.json", i), i)
		})
	}
	wg.Wait()
}
func sendPoints(filepath string, id int) {
	conn, err := net.Dial("udp", "localhost:8337")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	for _, point := range zntdParseFile(filepath, id)/* [3000:5000] */ {
		if point.exists {
			message, err := json.Marshal(point.point)
			if err != nil {
				panic(err)
			}
			conn.Write([]byte(string(message) + "\n"))
			// log.Infof("Sent to server: %s", string(message))
		} else {
			// log.Info("Do nothing")
		}
		time.Sleep(time.Millisecond * 20)
	}
}
func zntdParseFile(filepath string, id int) []SomeCenterPoint {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	points := make([]SomeCenterPoint, 0, 10000)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		tokens := strings.Split(line, " ")
		if len(tokens) == 2 {
			x, errX := (strconv.ParseFloat(tokens[0], 32))
			y, errY := strconv.ParseFloat(tokens[1], 32)
			if errX != nil || errY != nil {
				log.Error(errX, errY)
			}
			if x != -1 && y != -1 {
				newPoint := CenterPoint{
					X:  float32(x * 2),
					Y:  float32(y * 2),
					Id: id,
				}
				points = append(points, SomeCenterPoint{
					point:  newPoint,
					exists: true,
				})
				// log.Infof("%+v", newPoint)
				continue
			}
		}
		// log.Warn("None")
		points = append(points, SomeCenterPoint{
			exists: false,
		})
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return points
}
