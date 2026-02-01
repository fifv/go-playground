package main

import (
	"fmt"
	"reflect"
	"slices"
)

type MyStruct struct {
	name string
	age int
}
type MyStruct2 struct {
	name string
	data []int
}

/**
 * ### map
 * `nil map` can't be assigned
 * `nil map` can be read with any index, return 0
 * 
 * map initialized with `make(map[string]int)`
 * 
 * `initialized map` can be read with any index, if not exist then return 0
 * `initialized map` can be set with any index
 */

/**
 * ### slice
 * `nil slice` has len == 0
 * `nil slice` can't be read indexed when out-of-bound
 * `nil slice` can't be set indexed when out-of-bound
 * 
 * slice initialized with `make([]int)`
 * 
 * `initialized slice` can't be read indexed when out-of-bound
 * `initialized slice` can't be set indexed when out-of-bound
 */
func main() {
	s1 := []MyStruct{{name: "asdf"}, {}}
	s2 := []MyStruct{{name: "asdf"}, {}}
	fmt.Println(slices.Equal(s1, s2))
	fmt.Println(struct{}{} == struct{}{})

	s3 := []MyStruct2{{name: "asdf", data: []int{}}}
	s4 := []MyStruct2{{name: "asdf", data: []int{}}}
	fmt.Println(reflect.DeepEqual(s3, s4))
	/* can't set index out of range */
	// s3[1] = MyStruct2{name: "wtf"}
	/* can't read index out of range */
	// fmt.Println(s2[2])

	var nilSlice []byte

	/* it's okay, got 0 */
	fmt.Println(nilSlice == nil, len(nilSlice))
	nilSlice = append(nilSlice, byte(1))
	fmt.Println(nilSlice == nil, len(nilSlice))

	/* nil map == nil, and can't be assigned, otherwise panic */
	var mm map[string]int = make(map[string]int)
	fmt.Println(mm == nil, mm, mm["k2"])
	mm["k3"] = 1
	fmt.Println(mm == nil, mm)

	/* try map is value or >pointer< */
	mmm := map[string]int{}
	mmm["asdf"] = 1
	mmm2 := mmm
	fmt.Printf("%+v\n", mmm)
	mmm2["asdf"] = 2
	fmt.Printf("%+v\n", mmm)
	modifyTheMap(mmm2)
	fmt.Printf("%+v\n", mmm)

	/* compare struct */
	cm := map[MyStruct]struct{}{}
	cm[MyStruct{name: "Student", age: 24}] = struct{}{}
	fmt.Printf("%+v\n", cm)
	v, ok := cm[MyStruct{name: "Student", age: 24}]
	fmt.Printf("%+v %+v\n", v, ok)



}

func modifyTheMap(theMap map[string]int)  {
	theMap["aaa"] = 3
}