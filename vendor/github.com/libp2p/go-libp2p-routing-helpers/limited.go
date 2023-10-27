package routinghelpers

import (
	"context"
	"io"
	"strings"

	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
)

// LimitedValueStore limits the internal value store to the given namespaces.
type LimitedValueStore struct {
	routing.ValueStore
	Namespaces []string
}

// GetPublicKey returns the public key for the given peer, if and only if this
// router supports the /pk namespace. Otherwise, it returns routing.ErrNotFound.
func (lvs *LimitedValueStore) GetPublicKey(ctx context.Context, p peer.ID) (ci.PubKey, error) {
	for _, ns := range lvs.Namespaces {
		if ns == "pk" {
			return routing.GetPublicKey(lvs.ValueStore, ctx, p)
		}
	}
	return nil, routing.ErrNotFound
}

// PutValue puts the given key in the underlying value store if the namespace
// is supported. Otherwise, it returns routing.ErrNotSupported.
func (lvs *LimitedValueStore) PutValue(ctx context.Context, key string, value []byte, opts ...routing.Option) error {
	if !lvs.KeySupported(key) {
		return routing.ErrNotSupported
	}
	return lvs.ValueStore.PutValue(ctx, key, value, opts...)
}

// KeySupported returns true if the passed key is supported by this value store.
func (lvs *LimitedValueStore) KeySupported(key string) bool {
	if len(key) < 3 {
		return false
	}
	if key[0] != '/' {
		return false
	}
	key = key[1:]
	for _, ns := range lvs.Namespaces {
		if len(ns) < len(key) && strings.HasPrefix(key, ns) && key[len(ns)] == '/' {
			return true
		}
	}
	return false
}

// GetValue retrieves the given key from the underlying value store if the namespace
// is supported. Otherwise, it returns routing.ErrNotFound.
func (lvs *LimitedValueStore) GetValue(ctx context.Context, key string, opts ...routing.Option) ([]byte, error) {
	if !lvs.KeySupported(key) {
		return nil, routing.ErrNotFound
	}
	return lvs.ValueStore.GetValue(ctx, key, opts...)
}

// SearchValue searches the underlying value store for the given key if the
// namespace is supported, returning results in monotonically increasing
// "freshness". Otherwise, it returns an empty, closed channel to indicate that
// the value wasn't found.
func (lvs *LimitedValueStore) SearchValue(ctx context.Context, key string, opts ...routing.Option) (<-chan []byte, error) {
	if !lvs.KeySupported(key) {
		out := make(chan []byte)
		close(out)
		return out, nil
	}
	return lvs.ValueStore.SearchValue(ctx, key, opts...)
}

// Bootstrap signals the underlying value store to get into the "bootstrapped"
// state, if it implements the Bootstrap interface.
func (lvs *LimitedValueStore) Bootstrap(ctx context.Context) error {
	if bs, ok := lvs.ValueStore.(Bootstrap); ok {
		return bs.Bootstrap(ctx)
	}
	return nil
}

// Close closest the underlying value store if it implements the io.Closer
// interface.
func (lvs *LimitedValueStore) Close() error {
	if closer, ok := lvs.ValueStore.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

var _ routing.PubKeyFetcher = (*LimitedValueStore)(nil)
var _ routing.ValueStore = (*LimitedValueStore)(nil)
var _ Bootstrap = (*LimitedValueStore)(nil)
