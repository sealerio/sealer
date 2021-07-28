package client

import "time"

type clientOpt struct {
	timeout time.Duration
}

type Option func(opt *clientOpt) error

func WithTimeOut(d time.Duration) Option {
	return func(opt *clientOpt) error {
		opt.timeout = d
		return nil
	}
}
