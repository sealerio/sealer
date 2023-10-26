package autorelay

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"

	ma "github.com/multiformats/go-multiaddr"
)

var (
	// this is purposefully long to require some node stability before advertising as a relay
	AdvertiseBootDelay = 15 * time.Minute
	AdvertiseTTL       = 30 * time.Minute
)

// Advertise advertises this node as a libp2p relay.
func Advertise(ctx context.Context, advertise discovery.Advertiser) {
	go func() {
		select {
		case <-time.After(AdvertiseBootDelay):
			go func() {
				for {
					ttl, err := advertise.Advertise(ctx, RelayRendezvous, discovery.TTL(AdvertiseTTL))
					if err != nil {
						log.Debugf("Error advertising %s: %s", RelayRendezvous, err.Error())
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
		case <-ctx.Done():
		}
	}()
}

// Filter filters out all relay addresses.
func Filter(addrs []ma.Multiaddr) []ma.Multiaddr {
	raddrs := make([]ma.Multiaddr, 0, len(addrs))
	for _, addr := range addrs {
		if isRelayAddr(addr) {
			continue
		}
		raddrs = append(raddrs, addr)
	}
	return raddrs
}
