package fullrt

import (
	"fmt"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

type config struct {
	dhtOpts []kaddht.Option
}

func (cfg *config) apply(opts ...Option) error {
	for i, o := range opts {
		if err := o(cfg); err != nil {
			return fmt.Errorf("fullrt dht option %d failed: %w", i, err)
		}
	}
	return nil
}

type Option func(opt *config) error

func DHTOption(opts ...kaddht.Option) Option {
	return func(c *config) error {
		c.dhtOpts = append(c.dhtOpts, opts...)
		return nil
	}
}
