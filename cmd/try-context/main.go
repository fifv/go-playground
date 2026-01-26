package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx := context.Background()
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		fmt.Println("goroutine start")
		time.Sleep(time.Second * 2)
		fmt.Println("before cancel")
		cancel()
		fmt.Println("after cancel")
	}()

	for {
		select {
		case <-newCtx.Done():
			fmt.Println("done")
			return
		default:
			fmt.Println("default")
		}
		time.Sleep(time.Second)
	}
}
