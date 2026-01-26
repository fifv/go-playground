package main

import (
	"fmt"
	"time"
)

/**
 * Write to closed channel cause panic
 */
func main() {

	fmt.Println("Hi")
	doneCh := make(chan int)
	defer close(doneCh)
	go doneIt(doneCh)
	go doneIt(doneCh)
	go doneIt(doneCh)
	<-doneCh
	close(doneCh)
	fmt.Println("Done!")
	time.Sleep(time.Second)
}

func doneIt(doneCh chan int) {
	time.Sleep(time.Second)
	fmt.Println("Write To Done...")
	doneCh <- 1
	fmt.Println("Write To Done!")

}
