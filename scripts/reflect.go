package main

import (
	"fmt"
	"reflect"
)

type Particle struct {
	X, V [3]float32
	ID int64
}

type Runtime struct {
	Data map[string]interface{ }
}

func main() {

	var x1 []float32
	var x2 []float64
	var x3 [][3]float64
	var x4 []Particle
	
	Get(&x1)
	Get(&x2)
	Get(&x3)
	Get(&x4)
}

func CheckedGet(name string, x interface{ }) error {
	if reflect.TypeOf(x).Kind() != reflect.Pointer {
		return fmt.Errorf("")
	}
}

