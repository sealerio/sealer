package internal

import (
	"context"
)

type CtxMutex chan struct{}

func NewCtxMutex() CtxMutex {
	return make(CtxMutex, 1)
}

func (m CtxMutex) Lock(ctx context.Context) error {
	select {
	case m <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m CtxMutex) Unlock() {
	select {
	case <-m:
	default:
		panic("not locked")
	}
}
