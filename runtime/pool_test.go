package runtime

import (
	r "runtime"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	pool := NewPool(2)
	println(r.NumGoroutine())
	for i := 0; i < 10; i++ {
		pool.Add(1)
		go func(n int) {
			time.Sleep(time.Second)
			println(r.NumGoroutine(), n)
			pool.Done()
		}(i)
	}
	pool.Wait()
	println(r.NumGoroutine())
}
