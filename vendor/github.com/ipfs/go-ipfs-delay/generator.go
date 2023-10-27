package delay

import (
	"math/rand"
	"time"
)

// Generator provides an interface for generating wait times
type Generator interface {
	NextWaitTime(time.Duration) time.Duration
}

var sharedRNG = rand.New(rand.NewSource(time.Now().UnixNano()))

// VariableNormalGenerator makes delays that following a normal distribution
func VariableNormalGenerator(std time.Duration, rng *rand.Rand) Generator {
	if rng == nil {
		rng = sharedRNG
	}

	return &variableNormal{
		std: std,
		rng: rng,
	}
}

type variableNormal struct {
	std time.Duration
	rng *rand.Rand
}

func (d *variableNormal) NextWaitTime(t time.Duration) time.Duration {
	return time.Duration(d.rng.NormFloat64()*float64(d.std)) + t
}

// VariableUniformGenerator generates delays following a uniform distribution
func VariableUniformGenerator(d time.Duration, rng *rand.Rand) Generator {
	if rng == nil {
		rng = sharedRNG
	}

	return &variableUniform{
		d:   d,
		rng: rng,
	}
}

type variableUniform struct {
	d   time.Duration // max delta
	rng *rand.Rand
}

func (d *variableUniform) NextWaitTime(t time.Duration) time.Duration {
	return time.Duration(d.rng.Float64()*float64(d.d)) + t
}

type fixed struct{}

// FixedGenerator returns a delay with fixed latency
func FixedGenerator() Generator {
	return &fixed{}
}

func (d *fixed) NextWaitTime(t time.Duration) time.Duration {
	return t
}
