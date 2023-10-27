// Package offline implements Routing with a client which
// is only able to perform offline operations.
package offline

import (
	"bytes"
	"context"
	"errors"
	"time"

	proto "github.com/gogo/protobuf/proto"
	cid "github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	dshelp "github.com/ipfs/go-ipfs-ds-help"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"

	record "github.com/libp2p/go-libp2p-record"
	pb "github.com/libp2p/go-libp2p-record/pb"
)

// ErrOffline is returned when trying to perform operations that
// require connectivity.
var ErrOffline = errors.New("routing system in offline mode")

// NewOfflineRouter returns an Routing implementation which only performs
// offline operations. It allows to Put and Get signed dht
// records to and from the local datastore.
func NewOfflineRouter(dstore ds.Datastore, validator record.Validator) routing.Routing {
	return &offlineRouting{
		datastore: dstore,
		validator: validator,
	}
}

// offlineRouting implements the Routing interface,
// but only provides the capability to Put and Get signed dht
// records to and from the local datastore.
type offlineRouting struct {
	datastore ds.Datastore
	validator record.Validator
}

func (c *offlineRouting) PutValue(ctx context.Context, key string, val []byte, _ ...routing.Option) error {
	if err := c.validator.Validate(key, val); err != nil {
		return err
	}
	if old, err := c.GetValue(ctx, key); err == nil {
		// be idempotent to be nice.
		if bytes.Equal(old, val) {
			return nil
		}
		// check to see if the older record is better
		i, err := c.validator.Select(key, [][]byte{val, old})
		if err != nil {
			// this shouldn't happen for validated records.
			return err
		}
		if i != 0 {
			return errors.New("can't replace a newer record with an older one")
		}
	}
	rec := record.MakePutRecord(key, val)
	data, err := proto.Marshal(rec)
	if err != nil {
		return err
	}

	return c.datastore.Put(ctx, dshelp.NewKeyFromBinary([]byte(key)), data)
}

func (c *offlineRouting) GetValue(ctx context.Context, key string, _ ...routing.Option) ([]byte, error) {
	buf, err := c.datastore.Get(ctx, dshelp.NewKeyFromBinary([]byte(key)))
	if err != nil {
		return nil, err
	}

	rec := new(pb.Record)
	err = proto.Unmarshal(buf, rec)
	if err != nil {
		return nil, err
	}
	val := rec.GetValue()

	err = c.validator.Validate(key, val)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (c *offlineRouting) SearchValue(ctx context.Context, key string, _ ...routing.Option) (<-chan []byte, error) {
	out := make(chan []byte, 1)
	go func() {
		defer close(out)
		v, err := c.GetValue(ctx, key)
		if err == nil {
			out <- v
		}
	}()
	return out, nil
}

func (c *offlineRouting) FindPeer(ctx context.Context, pid peer.ID) (peer.AddrInfo, error) {
	return peer.AddrInfo{}, ErrOffline
}

func (c *offlineRouting) FindProvidersAsync(ctx context.Context, k cid.Cid, max int) <-chan peer.AddrInfo {
	out := make(chan peer.AddrInfo)
	close(out)
	return out
}

func (c *offlineRouting) Provide(_ context.Context, k cid.Cid, _ bool) error {
	return ErrOffline
}

func (c *offlineRouting) Ping(ctx context.Context, p peer.ID) (time.Duration, error) {
	return 0, ErrOffline
}

func (c *offlineRouting) Bootstrap(context.Context) error {
	return nil
}

// ensure offlineRouting matches the Routing interface
var _ routing.Routing = &offlineRouting{}
