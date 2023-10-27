package delay

import (
	"math/rand"
	"sync"
	"time"
)

// D (Delay) makes it easy to add (threadsafe) configurable delays to other
// objects.
type D interface {
	Set(time.Duration) time.Duration
	Wait()
	NextWaitTime() time.Duration
	Get() time.Duration
}

type delay struct {
	l         sync.RWMutex
	t         time.Duration
	generator Generator
}

func (d *delay) Set(t time.Duration) time.Duration {
	d.l.Lock()
	defer d.l.Unlock()
	prev := d.t
	d.t = t
	return prev
}

func (d *delay) Wait() {
	d.l.RLock()
	defer d.l.RUnlock()
	time.Sleep(d.generator.NextWaitTime(d.t))
}

func (d *delay) NextWaitTime() time.Duration {
	d.l.Lock()
	defer d.l.Unlock()
	return d.generator.NextWaitTime(d.t)
}

func (d *delay) Get() time.Duration {
	d.l.Lock()
	defer d.l.Unlock()
	return d.t
}

// Delay generates a generic delay form a t, a sleeper, and a generator
func Delay(t time.Duration, generator Generator) D {
	return &delay{
		t:         t,
		generator: generator,
	}
}

// Fixed returns a delay with fixed latency
func Fixed(t time.Duration) D {
	return Delay(t, FixedGenerator())
}

// VariableUniform is a delay following a uniform distribution
// Notice that to implement the D interface Set can only change the minimum delay
// the delta is set only at initialization
func VariableUniform(t, d time.Duration, rng *rand.Rand) D {
	return Delay(t, VariableUniformGenerator(d, rng))
}

// VariableNormal is a delay following a normal distribution
// Notice that to implement the D interface Set can only change the mean delay
// the standard deviation is set only at initialization
func VariableNormal(t, std time.Duration, rng *rand.Rand) D {
	return Delay(t, VariableNormalGenerator(std, rng))
}
