package main

import "fmt"

func main() {
	for i := range 10 {
		fmt.Println("current", i)
		/**
		 * defer runs at the end of the surrounding function,
		 * not the surrounding block.
		 */
		defer func() {
			fmt.Println("defer", i)
		}()
	}
}
