package main

import "fmt"

func main() {
	mm := map[string]int{}
	mm["k1"] = 1
	mm["k2"] = 2
	fmt.Println(mm["k3"])
	val, ok := mm["k3"]
	if !ok {
		fmt.Println("no 3!", val)
	}
	if val == 3 {
		fmt.Println("3!", val)
	}

	fmt.Println(mm)
}
