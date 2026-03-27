package main

import (
	"bytes"
	"fmt"
	"time"
	"runtime"

	"math/rand"

	"encoding/gob"
	"encoding/binary"
)

type Message struct {
	Type string
	Content string
}

type Particle struct {
	ID uint64
	X, V [3]float32
}

func main() {
	var network bytes.Buffer
	gobEnc := gob.NewEncoder(&network)
	gobDec := gob.NewDecoder(&network)

	RunBenchmark("gob", 100*1000, 10, gobEnc.Encode, gobDec.Decode)

	binEnc := func(v any) error {
		return binary.Write(&network, binary.LittleEndian, v)
	}
	binDec := func(v any) error {
		return binary.Read(&network, binary.LittleEndian, v)
	}
	
	RunBenchmark("binary", 100*1000, 10, binEnc, binDec)
}

type EncDec func(v any) error

func RunBenchmark(name string, N, n int, enc, dec EncDec) {
	var (
		ms0, ms1, ms2 runtime.MemStats
		msTotal0, msTotal1 runtime.MemStats
		tTotal0, tTotal1 float64
	)
	
	in, out :=  ParticleBuffers(N)
	enc(in)
	dec(out)
	
	for i := 0; i < n; i++ {
		runtime.ReadMemStats(&ms0)
		
		t0 := time.Now()
		err := enc(in)
		if err != nil { panic(err.Error()) }
		dt0 := time.Since(t0)

		runtime.ReadMemStats(&ms1)
		
		t1 := time.Now()
		err = dec(&out)
		if err != nil { panic(err.Error()) }
		dt1 := time.Since(t1)

		runtime.ReadMemStats(&ms2)
		runtime.ReadMemStats(&ms2)

		tTotal0 += dt0.Seconds()
		tTotal1 += dt1.Seconds()
		
		dms0 := SubMemStats(ms1, ms0)
		dms1 := SubMemStats(ms2, ms1)
		
		msTotal0 = AddMemStats(msTotal0, dms0)
		msTotal1 = AddMemStats(msTotal1, dms1)
	}

	fmt.Printf("%10s N: %6d n: %6d: enc: %9.3g s/op dec: %9.3g s/op\n",
		name, N, n, tTotal0/float64(n), tTotal1/float64(n))
	for i := range msTotal0.BySize {
		if int(msTotal0.BySize[i].Mallocs) > n/2 ||
			int(msTotal1.BySize[i].Mallocs) > n/2 {
			
			fmt.Printf("%9.3g B: %9.3g %9.3g\n",
				float64(ms0.BySize[i].Size),
				float64(msTotal0.BySize[i].Mallocs)/float64(n),
				float64(msTotal1.BySize[i].Mallocs)/float64(n),
			)
		}
	}
}

func AddMemStats(ms0, ms1 runtime.MemStats) runtime.MemStats {
	var out runtime.MemStats
	for i := range ms0.BySize {
		out.BySize[i].Size = ms0.BySize[i].Size
		out.BySize[i].Mallocs = ms0.BySize[i].Mallocs + ms1.BySize[i].Mallocs
		out.BySize[i].Frees = ms0.BySize[i].Frees + ms1.BySize[i].Frees
	}
	return out
}

func SubMemStats(ms0, ms1 runtime.MemStats) runtime.MemStats {
	var out runtime.MemStats
	for i := range ms0.BySize {
		out.BySize[i].Size = ms0.BySize[i].Size
		out.BySize[i].Mallocs = ms0.BySize[i].Mallocs - ms1.BySize[i].Mallocs
		out.BySize[i].Frees = ms0.BySize[i].Frees - ms1.BySize[i].Frees
	}
	return out
}

func ParticleBuffers(N int) (in, out []Particle) {
	in, out = make([]Particle, N), make([]Particle, N)
	for i := range N {
		in[i].ID = rand.Uint64()
		for k := range 3 {
			in[i].X[k] = rand.Float32()
			in[i].V[k] = rand.Float32()
		}
	}

	return in, out
}

func ClearParticles(p []Particle) {
	for i := range p {
		p[i] = Particle{ }
	}
}
