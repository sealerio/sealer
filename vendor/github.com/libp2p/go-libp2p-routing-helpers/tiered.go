package routinghelpers

import (
	"context"
	"io"

	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"

	multierror "github.com/hashicorp/go-multierror"
	cid "github.com/ipfs/go-cid"
	record "github.com/libp2p/go-libp2p-record"
)

// Tiered is like the Parallel except that GetValue and FindPeer
// are called in series.
type Tiered struct {
	Routers   []routing.Routing
	Validator record.Validator
}

// PutValue puts the given key to all sub-routers in parallel. It succeeds as
// long as putting to at least one sub-router succeeds, but it waits for all
// puts to terminate.
func (r Tiered) PutValue(ctx context.Context, key string, value []byte, opts ...routing.Option) error {
	return Parallel{Routers: r.Routers}.PutValue(ctx, key, value, opts...)
}

func (r Tiered) get(ctx context.Context, do func(routing.Routing) (interface{}, error)) (interface{}, error) {
	var errs []error
	for _, ri := range r.Routers {
		val, err := do(ri)
		switch err {
		case nil:
			return val, nil
		case routing.ErrNotFound, routing.ErrNotSupported:
			continue
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		errs = append(errs, err)
	}
	switch len(errs) {
	case 0:
		return nil, routing.ErrNotFound
	case 1:
		return nil, errs[0]
	default:
		return nil, &multierror.Error{Errors: errs}
	}
}

// GetValue sequentially searches each sub-router for the given key, returning
// the value from the first sub-router to complete the query.
func (r Tiered) GetValue(ctx context.Context, key string, opts ...routing.Option) ([]byte, error) {
	valInt, err := r.get(ctx, func(ri routing.Routing) (interface{}, error) {
		return ri.GetValue(ctx, key, opts...)
	})
	val, _ := valInt.([]byte)
	return val, err
}

// SearchValue searches all sub-routers for the given key in parallel,
// returning results in monotonically increasing "freshness" from all
// sub-routers.
func (r Tiered) SearchValue(ctx context.Context, key string, opts ...routing.Option) (<-chan []byte, error) {
	return Parallel{Routers: r.Routers, Validator: r.Validator}.SearchValue(ctx, key, opts...)
}

// GetPublicKey sequentially searches each sub-router for the the public key,
// returning the first result.
func (r Tiered) GetPublicKey(ctx context.Context, p peer.ID) (ci.PubKey, error) {
	vInt, err := r.get(ctx, func(ri routing.Routing) (interface{}, error) {
		return routing.GetPublicKey(ri, ctx, p)
	})
	val, _ := vInt.(ci.PubKey)
	return val, err
}

// Provide announces that this peer provides the content in question to all
// sub-routers in parallel. Provide returns success as long as a single
// sub-router succeeds, but still waits for all sub-routers to finish before
// returning.
func (r Tiered) Provide(ctx context.Context, c cid.Cid, local bool) error {
	return Parallel{Routers: r.Routers}.Provide(ctx, c, local)
}

// FindProvidersAsync searches all sub-routers in parallel for peers who are
// able to provide a given key.
//
// If count > 0, it returns at most count providers. If count == 0, it returns
// an unbounded number of providers.
func (r Tiered) FindProvidersAsync(ctx context.Context, c cid.Cid, count int) <-chan peer.AddrInfo {
	return Parallel{Routers: r.Routers}.FindProvidersAsync(ctx, c, count)
}

// FindPeer sequentially searches for given peer using each sub-router,
// returning the first result.
func (r Tiered) FindPeer(ctx context.Context, p peer.ID) (peer.AddrInfo, error) {
	valInt, err := r.get(ctx, func(ri routing.Routing) (interface{}, error) {
		return ri.FindPeer(ctx, p)
	})
	val, _ := valInt.(peer.AddrInfo)
	return val, err
}

// Bootstrap signals all the sub-routers to bootstrap.
func (r Tiered) Bootstrap(ctx context.Context) error {
	return Parallel{Routers: r.Routers}.Bootstrap(ctx)
}

// Close closes all sub-routers that implement the io.Closer interface.
func (r Tiered) Close() error {
	var me multierror.Error
	for _, router := range r.Routers {
		if closer, ok := router.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				me.Errors = append(me.Errors, err)
			}
		}
	}
	return me.ErrorOrNil()
}

var _ routing.Routing = Tiered{}
