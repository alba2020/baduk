package main

import (
	"encoding/binary"
	"math"
	"math/rand"

	"atari/engine"
)

type Agent struct {
	Weights  []float64
	Score    int
	AvgTurns float64
}

func NewAgent() *Agent {
	a := &Agent{Weights: make([]float64, engine.TotalParam)}
	for i := range a.Weights {
		a.Weights[i] = rand.NormFloat64() * math.Sqrt(2.0/9.0)
	}
	return a
}

func (a *Agent) Clone() *Agent {
	clone := &Agent{Weights: make([]float64, engine.TotalParam), Score: 0, AvgTurns: 0}
	copy(clone.Weights, a.Weights)
	return clone
}

func (a *Agent) ToBytes() []byte {
	buf := make([]byte, engine.TotalParam*8)
	for i, w := range a.Weights {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], math.Float64bits(w))
	}
	return buf
}

func (a *Agent) FromBytes(buf []byte) bool {
	if len(buf) != engine.TotalParam*8 {
		return false
	}
	for i := 0; i < engine.TotalParam; i++ {
		bits := binary.LittleEndian.Uint64(buf[i*8 : (i+1)*8])
		a.Weights[i] = math.Float64frombits(bits)
	}
	return true
}
