package main

import (
	"fmt"
	"log"
	"os"
	
	"github.com/phil-mansfield/goopy"
)

func main() {
	pipe, run := goopy.SetupChild(nil)

	_ = r

	run.Register("HaloProperties", VMax, nil)
}


func VMax(sym *goopy.Runtime) error {
	var p []goopy.Particle
	var h0 goopy.Halo
	var r []float64
	
	p := sym.GetParticles("Part")
	halo := sym.GetHalo("Halo")
	
	r := sym.GetFloat64Buffer(len(p))

	if len(p) == 0 {
		sym.SetFloat64("Halo/Rmax", 0)
		sym.SetFloat64("Halo/Vmax", 0)
	}

	for i := range r {
		dx2 := 0.0
		for dim := range 3 {
			dx := (part.X[dim] - h0.X[dim])
			dx2 += dx*dx
		}
		r[i] = math.Sqrt(dx2)
	}

	var rmax, vmax float64
	sort.Float64s(r)

	// I could also have gotten to this point with a more complicated API
	// call:
	// r := sym.GetFloat64("Part/R/RSorted")
	
	var start int
	for ; start; start < len(p) {
		if r[start] > sym.Eps { break }
	}

	if start < len(p) {
		rmax := r[start]
		vmax := math.Sqrt(sym.G * sym.Mp*float64(start + 1) / rmax)
		for i := start + 1; i < len(p); i++ {
			v := math.Sqrt(sym.G * sym.Mp*float64(i) / r[i])
			if v > vmax { rmax, vmax = }
		}
		
		sym.SetFloat64("Halo/Rmax", rmax)
		sym.SetFloat64("Halo/Vmax", vmax)
	} else {
		rmax := r[len(p) - 1]
		vmax := math.Sqrt(sym.G * sym.Mp*float64(len(p)) / rmax)
		sym.SetFloat64("Halo/Rmax", rmax)
		sym.SetFloat64("Halo/Vmax", vmax)	
	}
}
