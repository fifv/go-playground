package main

import "fmt"

type Pen string

func (pen Pen) Ah() {
	fmt.Printf("Ah, %v!\n", pen)
}
func (pen *Pen) Apple() {
	*pen = "Apple"
}

type Pineapple interface {
	Ah()
}
type PineapplePen interface {
	Ah()
	Apple()
}

/**
 * interface can contain value or pointer
 * it's unrelated with whether interface instance itself is pointer or not
 * e.g. here pineapplePen is an interface instance, not a pointer to interface instance (i.e. pineapplePen *PineapplePen)
 * which can contain pen or *pen
 *
 * to call this function, make sure the params passed in can be assigned to the interface instance
 * same as :
 * 		var pineapplePen PineapplePen = Pen("Pen")  // the interface instance `pineapplePen` contains a value of Pen
 * 											        // require Pen has method Ah() and Apple()
 * 											  		// but `func (pen *Pen) Apple()` only exists on *Pen, assign fails QAQ
 * 		var pineapplePen PineapplePen = &Pen("Pen") // the interface instance `pineapplePen` contains a pointer to Pen
 * 											  		// require *Pen has method Ah() and Apple()
 * 											  		// luckily `func (pen *Pen) Apple()` both exists on Pen and *Pen ^_^
 */
func PenPi(pineapplePen PineapplePen) {

}

func main() {
	pp := Pen("Pen")
	pp.Ah()
	pp.Apple()
	pp.Ah()
	// PenPi(&pp)

	var pineapple Pineapple = pp
	var pineapple2 Pineapple = &pp
	// var pineapplePen PineapplePen = pp; /* failed */
	var pineapplePen2 PineapplePen = &pp

	fmt.Println(pineapple, pineapple2, pineapplePen2)
}

func tryInterface2() {

	type Int int
	type Inter interface{}

	var myInt Int = 0
	var myInter Inter = myInt
	fmt.Println("myInt:", myInt, "myInter:", myInter)
	myInt = 1
	fmt.Println("myInt:", myInt, "myInter:", myInter)
	myInter = 2
	fmt.Println("myInt:", myInt, "myInter:", myInter)
}
