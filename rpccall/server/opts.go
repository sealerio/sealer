package server

type serverOption struct {
	port int
}

type Option func(opt *serverOption) error

func WithPort(port int) Option {
	return func(opt *serverOption) error {
		opt.port = port
		return nil
	}
}
