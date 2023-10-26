package routinghelpers

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"

	"github.com/ipfs/go-cid"
)

// Null is a router that doesn't do anything.
type Null struct{}

// PutValue always returns ErrNotSupported
func (nr Null) PutValue(context.Context, string, []byte, ...routing.Option) error {
	return routing.ErrNotSupported
}

// GetValue always returns ErrNotFound
func (nr Null) GetValue(context.Context, string, ...routing.Option) ([]byte, error) {
	return nil, routing.ErrNotFound
}

// SearchValue always returns ErrNotFound
func (nr Null) SearchValue(ctx context.Context, key string, opts ...routing.Option) (<-chan []byte, error) {
	return nil, routing.ErrNotFound
}

// Provide always returns ErrNotSupported
func (nr Null) Provide(context.Context, cid.Cid, bool) error {
	return routing.ErrNotSupported
}

// FindProvidersAsync always returns a closed channel
func (nr Null) FindProvidersAsync(context.Context, cid.Cid, int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo)
	close(ch)
	return ch
}

// FindPeer always returns ErrNotFound
func (nr Null) FindPeer(context.Context, peer.ID) (peer.AddrInfo, error) {
	return peer.AddrInfo{}, routing.ErrNotFound
}

// Bootstrap always succeeds instantly
func (nr Null) Bootstrap(context.Context) error {
	return nil
}

// Close always succeeds instantly
func (nr Null) Close() error {
	return nil
}

var _ routing.Routing = Null{}
