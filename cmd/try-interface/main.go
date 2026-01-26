package main

import "fmt"

type Int int
type Inter interface{}


func main() {
	var myInt Int = 0
	var myInter Inter = myInt
	fmt.Println("myInt:", myInt, "myInter:", myInter)
	myInt = 1
	fmt.Println("myInt:", myInt, "myInter:", myInter)
	myInter = 2
	fmt.Println("myInt:", myInt, "myInter:", myInter)
}
