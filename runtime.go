package goopy

import (
	"os"
	"reflect"
)

type ChildRuntime struct {
	Pipe *Pipe
	Eps, Mp, G float64
}

func NewChildRuntime(p *Pipe) *ChildRuntime {
	return &Runtime{
		Pipe: p,
		
		i64: map[string][]int64{ }, i32: map[string][]int32{ },
		f64: map[string][]float64{ }, f32: map[string][]float32{ },
		v64: map[string][][3]float64{ }, v32: map[string][][3]float32{ },
	}
}

func Get(name string, x interface{ }, start, end int) {
	t := reflect.TypeOf(x)
}

func Set(name string, x interface{ }, start int) {
}

type ParentRuntime struct {
	i64 map[string][]int64
	i32 map[string][]int32
	f64 map[string][]float64
	f32 map[string][]float32
	v64 map[string][][3]float64
	v32 map[string][][3]float32
}
