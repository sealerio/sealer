package simple

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-cidutil"
	"github.com/ipfs/go-fetcher"
	fetcherhelpers "github.com/ipfs/go-fetcher/helpers"
	blocks "github.com/ipfs/go-ipfs-blockstore"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-verifcid"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/libp2p/go-libp2p-core/routing"
)

var logR = logging.Logger("reprovider.simple")

// ErrClosed is returned by Trigger when operating on a closed reprovider.
var ErrClosed = errors.New("reprovider service stopped")

// KeyChanFunc is function streaming CIDs to pass to content routing
type KeyChanFunc func(context.Context) (<-chan cid.Cid, error)

// Reprovider reannounces blocks to the network
type Reprovider struct {
	// Reprovider context. Cancel to stop, then wait on closedCh.
	ctx      context.Context
	cancel   context.CancelFunc
	closedCh chan struct{}

	// Trigger triggers a reprovide.
	trigger chan chan<- error

	// The routing system to provide values through
	rsys routing.ContentRouting

	keyProvider KeyChanFunc

	tick time.Duration
}

// NewReprovider creates new Reprovider instance.
func NewReprovider(ctx context.Context, reprovideInterval time.Duration, rsys routing.ContentRouting, keyProvider KeyChanFunc) *Reprovider {
	ctx, cancel := context.WithCancel(ctx)
	return &Reprovider{
		ctx:      ctx,
		cancel:   cancel,
		closedCh: make(chan struct{}),
		trigger:  make(chan chan<- error),

		rsys:        rsys,
		keyProvider: keyProvider,
		tick:        reprovideInterval,
	}
}

// Close the reprovider
func (rp *Reprovider) Close() error {
	rp.cancel()
	<-rp.closedCh
	return nil
}

// Run re-provides keys with 'tick' interval or when triggered
func (rp *Reprovider) Run() {
	defer close(rp.closedCh)

	var initialReprovideCh, reprovideCh <-chan time.Time

	// If reproviding is enabled (non-zero)
	if rp.tick > 0 {
		reprovideTicker := time.NewTicker(rp.tick)
		defer reprovideTicker.Stop()
		reprovideCh = reprovideTicker.C

		// If the reprovide ticker is larger than a minute (likely),
		// provide once after we've been up a minute.
		//
		// Don't provide _immediately_ as we might be just about to stop.
		if rp.tick > time.Minute {
			initialReprovideTimer := time.NewTimer(time.Minute)
			defer initialReprovideTimer.Stop()

			initialReprovideCh = initialReprovideTimer.C
		}
	}

	var done chan<- error
	for rp.ctx.Err() == nil {
		select {
		case <-initialReprovideCh:
		case <-reprovideCh:
		case done = <-rp.trigger:
		case <-rp.ctx.Done():
			return
		}

		err := rp.Reprovide()

		// only log if we've hit an actual error, otherwise just tell the client we're shutting down
		if rp.ctx.Err() != nil {
			err = ErrClosed
		} else if err != nil {
			logR.Errorf("failed to reprovide: %s", err)
		}

		if done != nil {
			if err != nil {
				done <- err
			}
			close(done)
		}
	}
}

// Reprovide registers all keys given by rp.keyProvider to libp2p content routing
func (rp *Reprovider) Reprovide() error {
	keychan, err := rp.keyProvider(rp.ctx)
	if err != nil {
		return fmt.Errorf("failed to get key chan: %s", err)
	}
	for c := range keychan {
		// hash security
		if err := verifcid.ValidateCid(c); err != nil {
			logR.Errorf("insecure hash in reprovider, %s (%s)", c, err)
			continue
		}
		op := func() error {
			err := rp.rsys.Provide(rp.ctx, c, true)
			if err != nil {
				logR.Debugf("Failed to provide key: %s", err)
			}
			return err
		}

		err := backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), rp.ctx))
		if err != nil {
			logR.Debugf("Providing failed after number of retries: %s", err)
			return err
		}
	}
	return nil
}

// Trigger starts the reprovision process in rp.Run and waits for it to finish.
//
// Returns an error if a reprovide is already in progress.
func (rp *Reprovider) Trigger(ctx context.Context) error {
	resultCh := make(chan error, 1)
	select {
	case rp.trigger <- resultCh:
	default:
		return fmt.Errorf("reprovider is already running")
	}

	select {
	case err := <-resultCh:
		return err
	case <-rp.ctx.Done():
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Strategies

// NewBlockstoreProvider returns key provider using bstore.AllKeysChan
func NewBlockstoreProvider(bstore blocks.Blockstore) KeyChanFunc {
	return func(ctx context.Context) (<-chan cid.Cid, error) {
		return bstore.AllKeysChan(ctx)
	}
}

// Pinner interface defines how the simple.Reprovider wants to interact
// with a Pinning service
type Pinner interface {
	DirectKeys(ctx context.Context) ([]cid.Cid, error)
	RecursiveKeys(ctx context.Context) ([]cid.Cid, error)
}

// NewPinnedProvider returns provider supplying pinned keys
func NewPinnedProvider(onlyRoots bool, pinning Pinner, fetchConfig fetcher.Factory) KeyChanFunc {
	return func(ctx context.Context) (<-chan cid.Cid, error) {
		set, err := pinSet(ctx, pinning, fetchConfig, onlyRoots)
		if err != nil {
			return nil, err
		}

		outCh := make(chan cid.Cid)
		go func() {
			defer close(outCh)
			for c := range set.New {
				select {
				case <-ctx.Done():
					return
				case outCh <- c:
				}
			}

		}()

		return outCh, nil
	}
}

func pinSet(ctx context.Context, pinning Pinner, fetchConfig fetcher.Factory, onlyRoots bool) (*cidutil.StreamingSet, error) {
	set := cidutil.NewStreamingSet()

	go func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		defer close(set.New)

		dkeys, err := pinning.DirectKeys(ctx)
		if err != nil {
			logR.Errorf("reprovide direct pins: %s", err)
			return
		}
		for _, key := range dkeys {
			set.Visitor(ctx)(key)
		}

		rkeys, err := pinning.RecursiveKeys(ctx)
		if err != nil {
			logR.Errorf("reprovide indirect pins: %s", err)
			return
		}

		session := fetchConfig.NewSession(ctx)
		for _, key := range rkeys {
			set.Visitor(ctx)(key)
			if !onlyRoots {
				err := fetcherhelpers.BlockAll(ctx, session, cidlink.Link{Cid: key}, func(res fetcher.FetchResult) error {
					clink, ok := res.LastBlockLink.(cidlink.Link)
					if ok {
						set.Visitor(ctx)(clink.Cid)
					}
					return nil
				})
				if err != nil {
					logR.Errorf("reprovide indirect pins: %s", err)
					return
				}
			}
		}
	}()

	return set, nil
}
