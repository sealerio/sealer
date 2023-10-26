package discovery

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"

	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("discovery")

// FindPeers is a utility function that synchronously collects peers from a Discoverer.
func FindPeers(ctx context.Context, d discovery.Discoverer, ns string, opts ...discovery.Option) ([]peer.AddrInfo, error) {
	var res []peer.AddrInfo

	ch, err := d.FindPeers(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}

	for pi := range ch {
		res = append(res, pi)
	}

	return res, nil
}

// Advertise is a utility function that persistently advertises a service through an Advertiser.
func Advertise(ctx context.Context, a discovery.Advertiser, ns string, opts ...discovery.Option) {
	go func() {
		for {
			ttl, err := a.Advertise(ctx, ns, opts...)
			if err != nil {
				log.Debugf("Error advertising %s: %s", ns, err.Error())
				if ctx.Err() != nil {
					return
				}

				select {
				case <-time.After(2 * time.Minute):
					continue
				case <-ctx.Done():
					return
				}
			}

			wait := 7 * ttl / 8
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return
			}
		}
	}()
}
