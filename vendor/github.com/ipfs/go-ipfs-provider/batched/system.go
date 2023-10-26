package batched

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	provider "github.com/ipfs/go-ipfs-provider"
	"github.com/ipfs/go-ipfs-provider/queue"
	"github.com/ipfs/go-ipfs-provider/simple"
	logging "github.com/ipfs/go-log"
	"github.com/ipfs/go-verifcid"
	"github.com/multiformats/go-multihash"
)

var log = logging.Logger("provider.batched")

type BatchProvidingSystem struct {
	ctx     context.Context
	close   context.CancelFunc
	closewg sync.WaitGroup

	reprovideInterval        time.Duration
	initalReprovideDelay     time.Duration
	initialReprovideDelaySet bool

	rsys        provideMany
	keyProvider simple.KeyChanFunc

	q  *queue.Queue
	ds datastore.Batching

	reprovideCh chan cid.Cid

	totalProvides, lastReprovideBatchSize     int
	avgProvideDuration, lastReprovideDuration time.Duration
}

var _ provider.System = (*BatchProvidingSystem)(nil)

type provideMany interface {
	ProvideMany(ctx context.Context, keys []multihash.Multihash) error
	Ready() bool
}

// Option defines the functional option type that can be used to configure
// BatchProvidingSystem instances
type Option func(system *BatchProvidingSystem) error

var lastReprovideKey = datastore.NewKey("/provider/reprovide/lastreprovide")

func New(provider provideMany, q *queue.Queue, opts ...Option) (*BatchProvidingSystem, error) {
	s := &BatchProvidingSystem{
		reprovideInterval: time.Hour * 24,
		rsys:              provider,
		keyProvider:       nil,
		q:                 q,
		ds:                datastore.NewMapDatastore(),
		reprovideCh:       make(chan cid.Cid),
	}

	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}

	// Setup default behavior for the initial reprovide delay
	//
	// If the reprovide ticker is larger than a minute (likely),
	// provide once after we've been up a minute.
	//
	// Don't provide _immediately_ as we might be just about to stop.
	if !s.initialReprovideDelaySet && s.reprovideInterval > time.Minute {
		s.initalReprovideDelay = time.Minute
		s.initialReprovideDelaySet = true
	}

	if s.keyProvider == nil {
		s.keyProvider = func(ctx context.Context) (<-chan cid.Cid, error) {
			ch := make(chan cid.Cid)
			close(ch)
			return ch, nil
		}
	}

	// This is after the options processing so we do not have to worry about leaking a context if there is an
	// initialization error processing the options
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.close = cancel

	return s, nil
}

func Datastore(batching datastore.Batching) Option {
	return func(system *BatchProvidingSystem) error {
		system.ds = batching
		return nil
	}
}

func ReproviderInterval(duration time.Duration) Option {
	return func(system *BatchProvidingSystem) error {
		system.reprovideInterval = duration
		return nil
	}
}

func KeyProvider(fn simple.KeyChanFunc) Option {
	return func(system *BatchProvidingSystem) error {
		system.keyProvider = fn
		return nil
	}
}

func initialReprovideDelay(duration time.Duration) Option {
	return func(system *BatchProvidingSystem) error {
		system.initialReprovideDelaySet = true
		system.initalReprovideDelay = duration
		return nil
	}
}

func (s *BatchProvidingSystem) Run() {
	// how long we wait between the first provider we hear about and batching up the provides to send out
	const pauseDetectionThreshold = time.Millisecond * 500
	// how long we are willing to collect providers for the batch after we receive the first one
	const maxCollectionDuration = time.Minute * 10

	provCh := s.q.Dequeue()

	s.closewg.Add(1)
	go func() {
		defer s.closewg.Done()

		m := make(map[cid.Cid]struct{})

		// setup stopped timers
		maxCollectionDurationTimer := time.NewTimer(time.Hour)
		pauseDetectTimer := time.NewTimer(time.Hour)
		stopAndEmptyTimer(maxCollectionDurationTimer)
		stopAndEmptyTimer(pauseDetectTimer)

		// make sure timers are cleaned up
		defer maxCollectionDurationTimer.Stop()
		defer pauseDetectTimer.Stop()

		resetTimersAfterReceivingProvide := func() {
			firstProvide := len(m) == 0
			if firstProvide {
				// after receiving the first provider start up the timers
				maxCollectionDurationTimer.Reset(maxCollectionDuration)
				pauseDetectTimer.Reset(pauseDetectionThreshold)
			} else {
				// otherwise just do a full restart of the pause timer
				stopAndEmptyTimer(pauseDetectTimer)
				pauseDetectTimer.Reset(pauseDetectionThreshold)
			}
		}

		for {
			performedReprovide := false

			// at the start of every loop the maxCollectionDurationTimer and pauseDetectTimer should be already be
			// stopped and have empty channels
		loop:
			for {
				select {
				case <-maxCollectionDurationTimer.C:
					// if this timer has fired then the pause timer has started so let's stop and empty it
					stopAndEmptyTimer(pauseDetectTimer)
					break loop
				default:
				}

				select {
				case c := <-provCh:
					resetTimersAfterReceivingProvide()
					m[c] = struct{}{}
					continue
				default:
				}

				select {
				case c := <-provCh:
					resetTimersAfterReceivingProvide()
					m[c] = struct{}{}
				case c := <-s.reprovideCh:
					resetTimersAfterReceivingProvide()
					m[c] = struct{}{}
					performedReprovide = true
				case <-pauseDetectTimer.C:
					// if this timer has fired then the max collection timer has started so let's stop and empty it
					stopAndEmptyTimer(maxCollectionDurationTimer)
					break loop
				case <-maxCollectionDurationTimer.C:
					// if this timer has fired then the pause timer has started so let's stop and empty it
					stopAndEmptyTimer(pauseDetectTimer)
					break loop
				case <-s.ctx.Done():
					return
				}
			}

			if len(m) == 0 {
				continue
			}

			keys := make([]multihash.Multihash, 0, len(m))
			for c := range m {
				delete(m, c)

				// hash security
				if err := verifcid.ValidateCid(c); err != nil {
					log.Errorf("insecure hash in reprovider, %s (%s)", c, err)
					continue
				}

				keys = append(keys, c.Hash())
			}

			// in case after removing all the invalid CIDs there are no valid ones left
			if len(keys) == 0 {
				continue
			}

			for !s.rsys.Ready() {
				log.Debugf("reprovider system not ready")
				select {
				case <-time.After(time.Minute):
				case <-s.ctx.Done():
					return
				}
			}

			log.Debugf("starting provide of %d keys", len(keys))
			start := time.Now()
			err := s.rsys.ProvideMany(s.ctx, keys)
			if err != nil {
				log.Debugf("providing failed %v", err)
				continue
			}
			dur := time.Since(start)

			totalProvideTime := int64(s.totalProvides) * int64(s.avgProvideDuration)
			recentAvgProvideDuration := time.Duration(int64(dur) / int64(len(keys)))
			s.avgProvideDuration = time.Duration((totalProvideTime + int64(dur)) / int64(s.totalProvides+len(keys)))
			s.totalProvides += len(keys)

			log.Debugf("finished providing of %d keys. It took %v with an average of %v per provide", len(keys), dur, recentAvgProvideDuration)

			if performedReprovide {
				s.lastReprovideBatchSize = len(keys)
				s.lastReprovideDuration = dur

				if err := s.ds.Put(s.ctx, lastReprovideKey, storeTime(time.Now())); err != nil {
					log.Errorf("could not store last reprovide time: %v", err)
				}
				if err := s.ds.Sync(s.ctx, lastReprovideKey); err != nil {
					log.Errorf("could not perform sync of last reprovide time: %v", err)
				}
			}
		}
	}()

	s.closewg.Add(1)
	go func() {
		defer s.closewg.Done()

		var initialReprovideCh, reprovideCh <-chan time.Time

		// If reproviding is enabled (non-zero)
		if s.reprovideInterval > 0 {
			reprovideTicker := time.NewTicker(s.reprovideInterval)
			defer reprovideTicker.Stop()
			reprovideCh = reprovideTicker.C

			// if there is a non-zero initial reprovide time that was set in the initializer or if the fallback has been
			if s.initialReprovideDelaySet {
				initialReprovideTimer := time.NewTimer(s.initalReprovideDelay)
				defer initialReprovideTimer.Stop()

				initialReprovideCh = initialReprovideTimer.C
			}
		}

		for s.ctx.Err() == nil {
			select {
			case <-initialReprovideCh:
			case <-reprovideCh:
			case <-s.ctx.Done():
				return
			}

			err := s.reprovide(s.ctx, false)

			// only log if we've hit an actual error, otherwise just tell the client we're shutting down
			if s.ctx.Err() == nil && err != nil {
				log.Errorf("failed to reprovide: %s", err)
			}
		}
	}()
}

func stopAndEmptyTimer(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}

func storeTime(t time.Time) []byte {
	val := []byte(fmt.Sprintf("%d", t.UnixNano()))
	return val
}

func parseTime(b []byte) (time.Time, error) {
	tns, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, tns), nil
}

func (s *BatchProvidingSystem) Close() error {
	s.close()
	err := s.q.Close()
	s.closewg.Wait()
	return err
}

func (s *BatchProvidingSystem) Provide(cid cid.Cid) error {
	return s.q.Enqueue(cid)
}

func (s *BatchProvidingSystem) Reprovide(ctx context.Context) error {
	return s.reprovide(ctx, true)
}

func (s *BatchProvidingSystem) reprovide(ctx context.Context, force bool) error {
	if !s.shouldReprovide() && !force {
		return nil
	}

	kch, err := s.keyProvider(ctx)
	if err != nil {
		return err
	}

reprovideCidLoop:
	for {
		select {
		case c, ok := <-kch:
			if !ok {
				break reprovideCidLoop
			}

			select {
			case s.reprovideCh <- c:
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (s *BatchProvidingSystem) getLastReprovideTime() (time.Time, error) {
	val, err := s.ds.Get(s.ctx, lastReprovideKey)
	if errors.Is(err, datastore.ErrNotFound) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get last reprovide time")
	}

	t, err := parseTime(val)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not decode last reprovide time, got %q", string(val))
	}

	return t, nil
}

func (s *BatchProvidingSystem) shouldReprovide() bool {
	t, err := s.getLastReprovideTime()
	if err != nil {
		log.Debugf("getting last reprovide time failed: %s", err)
		return false
	}

	if time.Since(t) < time.Duration(float64(s.reprovideInterval)*0.5) {
		return false
	}
	return true
}

type BatchedProviderStats struct {
	TotalProvides, LastReprovideBatchSize     int
	AvgProvideDuration, LastReprovideDuration time.Duration
}

// Stat returns various stats about this provider system
func (s *BatchProvidingSystem) Stat(ctx context.Context) (BatchedProviderStats, error) {
	// TODO: Does it matter that there is no locking around the total+average values?
	return BatchedProviderStats{
		TotalProvides:          s.totalProvides,
		LastReprovideBatchSize: s.lastReprovideBatchSize,
		AvgProvideDuration:     s.avgProvideDuration,
		LastReprovideDuration:  s.lastReprovideDuration,
	}, nil
}
