package namesys

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	record "github.com/libp2p/go-libp2p-record"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	dshelp "github.com/ipfs/go-ipfs-ds-help"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("pubsub-valuestore")

// Pubsub is the minimal subset of the pubsub interface required by the pubsub
// value store. This way, users can wrap the underlying pubsub implementation
// without re-exporting/implementing the entire interface.
type Pubsub interface {
	RegisterTopicValidator(topic string, validator interface{}, opts ...pubsub.ValidatorOpt) error
	Join(topic string, opts ...pubsub.TopicOpt) (*pubsub.Topic, error)
}

type watchGroup struct {
	// Note: this chan must be buffered, see notifyWatchers
	listeners map[chan []byte]struct{}
}

type PubsubValueStore struct {
	ctx context.Context
	ds  ds.Datastore
	ps  Pubsub

	host  host.Host
	fetch *fetchProtocol

	rebroadcastInitialDelay time.Duration
	rebroadcastInterval     time.Duration

	// Map of keys to topics
	mx     sync.Mutex
	topics map[string]*topicInfo

	watchLk  sync.Mutex
	watching map[string]*watchGroup

	Validator record.Validator
}

type topicInfo struct {
	topic *pubsub.Topic
	evts  *pubsub.TopicEventHandler
	sub   *pubsub.Subscription

	cancel   context.CancelFunc
	finished chan struct{}

	dbWriteMx sync.Mutex
}

// KeyToTopic converts a binary record key to a pubsub topic key.
func KeyToTopic(key string) string {
	// Record-store keys are arbitrary binary. However, pubsub requires UTF-8 string topic IDs.
	// Encodes to "/record/base64url(key)"
	return "/record/" + base64.RawURLEncoding.EncodeToString([]byte(key))
}

// Option is a function that configures a PubsubValueStore during initialization
type Option func(*PubsubValueStore) error

// NewPubsubValueStore constructs a new ValueStore that gets and receives records through pubsub.
func NewPubsubValueStore(ctx context.Context, host host.Host, ps Pubsub, validator record.Validator, opts ...Option) (*PubsubValueStore, error) {
	psValueStore := &PubsubValueStore{
		ctx: ctx,

		ds:                      dssync.MutexWrap(ds.NewMapDatastore()),
		ps:                      ps,
		host:                    host,
		rebroadcastInitialDelay: 100 * time.Millisecond,
		rebroadcastInterval:     time.Minute * 10,

		topics:   make(map[string]*topicInfo),
		watching: make(map[string]*watchGroup),

		Validator: validator,
	}

	for _, opt := range opts {
		err := opt(psValueStore)
		if err != nil {
			return nil, err
		}
	}

	psValueStore.fetch = newFetchProtocol(ctx, host, psValueStore.getLocal)

	go psValueStore.rebroadcast(ctx)

	return psValueStore, nil
}

// PutValue publishes a record through pubsub
func (p *PubsubValueStore) PutValue(ctx context.Context, key string, value []byte, opts ...routing.Option) error {
	if err := p.Subscribe(key); err != nil {
		return err
	}

	log.Debugf("PubsubPublish: publish value for key", key)

	p.mx.Lock()
	ti, ok := p.topics[key]
	p.mx.Unlock()
	if !ok {
		return errors.New("could not find topic handle")
	}

	ti.dbWriteMx.Lock()
	defer ti.dbWriteMx.Unlock()
	recCmp, err := p.putLocal(ctx, ti, key, value)
	if err != nil {
		return err
	}
	if recCmp < 0 {
		return nil
	}

	select {
	case err := <-p.psPublishChannel(ctx, ti.topic, value):
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// compare compares the input value with the current value.
// First return value is 0 if equal, greater than 0 if better, less than 0 if worse.
// Second return value is true if valid.
//
func (p *PubsubValueStore) compare(ctx context.Context, key string, val []byte) (int, bool) {
	if p.Validator.Validate(key, val) != nil {
		return -1, false
	}

	old, err := p.getLocal(ctx, key)
	if err != nil {
		// If the old one is invalid, the new one is *always* better.
		return 1, true
	}

	// Same record is not better
	if old != nil && bytes.Equal(old, val) {
		return 0, true
	}

	i, err := p.Validator.Select(key, [][]byte{val, old})
	if err == nil && i == 0 {
		return 1, true
	}
	return -1, true
}

func (p *PubsubValueStore) Subscribe(key string) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	// see if we already have a pubsub subscription; if not, subscribe
	_, ok := p.topics[key]
	if ok {
		return nil
	}

	topic := KeyToTopic(key)

	// Ignore the error. We have to check again anyways to make sure the
	// record hasn't expired.
	//
	// Also, make sure to do this *before* subscribing.
	myID := p.host.ID()
	_ = p.ps.RegisterTopicValidator(topic, func(
		ctx context.Context,
		src peer.ID,
		msg *pubsub.Message,
	) pubsub.ValidationResult {
		cmp, valid := p.compare(ctx, key, msg.GetData())
		if !valid {
			return pubsub.ValidationReject
		}

		if cmp > 0 || cmp == 0 && src == myID {
			return pubsub.ValidationAccept
		}
		return pubsub.ValidationIgnore
	})

	ti, err := p.createTopicHandler(topic)
	if err != nil {
		return err
	}

	p.topics[key] = ti
	ctx, cancel := context.WithCancel(p.ctx)
	ti.cancel = cancel

	go p.handleSubscription(ctx, ti, key)

	log.Debugf("PubsubResolve: subscribed to %s", key)

	return nil
}

// createTopicHandler creates an internal topic object. Must be called with p.mx held
func (p *PubsubValueStore) createTopicHandler(topic string) (*topicInfo, error) {
	t, err := p.ps.Join(topic)
	if err != nil {
		return nil, err
	}

	sub, err := t.Subscribe()
	if err != nil {
		_ = t.Close()
		return nil, err
	}

	evts, err := t.EventHandler()
	if err != nil {
		sub.Cancel()
		_ = t.Close()
	}

	ti := &topicInfo{
		topic:    t,
		evts:     evts,
		sub:      sub,
		finished: make(chan struct{}, 1),
	}

	return ti, nil
}

func (p *PubsubValueStore) rebroadcast(ctx context.Context) {
	select {
	case <-time.After(p.rebroadcastInitialDelay):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(p.rebroadcastInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mx.Lock()
			keys := make([]string, 0, len(p.topics))
			topics := make([]*topicInfo, 0, len(p.topics))
			for k, ti := range p.topics {
				keys = append(keys, k)
				topics = append(topics, ti)
			}
			p.mx.Unlock()
			if len(topics) > 0 {
				for i, k := range keys {
					val, err := p.getLocal(ctx, k)
					if err == nil {
						topic := topics[i].topic
						select {
						case <-p.psPublishChannel(ctx, topic, val):
						case <-ctx.Done():
							return
						}
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *PubsubValueStore) psPublishChannel(ctx context.Context, topic *pubsub.Topic, value []byte) chan error {
	done := make(chan error, 1)
	go func() {
		done <- topic.Publish(ctx, value)
	}()
	return done
}

// putLocal tries to put the key-value pair into the local datastore
// Requires that the ti.dbWriteMx is held when called
// Returns true if the value is better then what is currently in the datastore
// Returns any errors from putting the data in the datastore
func (p *PubsubValueStore) putLocal(ctx context.Context, ti *topicInfo, key string, value []byte) (int, error) {
	cmp, valid := p.compare(ctx, key, value)
	if valid && cmp > 0 {
		return cmp, p.ds.Put(ctx, dshelp.NewKeyFromBinary([]byte(key)), value)
	}
	return cmp, nil
}

func (p *PubsubValueStore) getLocal(ctx context.Context, key string) ([]byte, error) {
	val, err := p.ds.Get(ctx, dshelp.NewKeyFromBinary([]byte(key)))
	if err != nil {
		// Don't invalidate due to ds errors.
		if err == ds.ErrNotFound {
			err = routing.ErrNotFound
		}
		return nil, err
	}

	// If the old one is invalid, the new one is *always* better.
	if err := p.Validator.Validate(key, val); err != nil {
		return nil, err
	}
	return val, nil
}

func (p *PubsubValueStore) GetValue(ctx context.Context, key string, opts ...routing.Option) ([]byte, error) {
	if err := p.Subscribe(key); err != nil {
		return nil, err
	}

	return p.getLocal(ctx, key)
}

func (p *PubsubValueStore) SearchValue(ctx context.Context, key string, opts ...routing.Option) (<-chan []byte, error) {
	if err := p.Subscribe(key); err != nil {
		return nil, err
	}

	p.watchLk.Lock()
	defer p.watchLk.Unlock()

	out := make(chan []byte, 1)
	lv, err := p.getLocal(ctx, key)
	if err == nil {
		out <- lv
		close(out)
		return out, nil
	}

	wg, ok := p.watching[key]
	if !ok {
		wg = &watchGroup{
			listeners: map[chan []byte]struct{}{},
		}
		p.watching[key] = wg
	}

	proxy := make(chan []byte, 1)

	ctx, cancel := context.WithCancel(ctx)
	wg.listeners[proxy] = struct{}{}

	go func() {
		defer func() {
			cancel()

			p.watchLk.Lock()
			delete(wg.listeners, proxy)

			if _, ok := p.watching[key]; len(wg.listeners) == 0 && ok {
				delete(p.watching, key)
			}
			p.watchLk.Unlock()

			close(out)
		}()

		for {
			select {
			case val, ok := <-proxy:
				if !ok {
					return
				}

				// outCh is buffered, so we just put the value or swap it for the newer one
				select {
				case out <- val:
				case <-out:
					out <- val
				}

				// 1 is good enough
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// GetSubscriptions retrieves a list of active topic subscriptions
func (p *PubsubValueStore) GetSubscriptions() []string {
	p.mx.Lock()
	defer p.mx.Unlock()

	var res []string
	for sub := range p.topics {
		res = append(res, sub)
	}

	return res
}

// Cancel cancels a topic subscription; returns true if an active
// subscription was canceled
func (p *PubsubValueStore) Cancel(name string) (bool, error) {
	p.mx.Lock()
	defer p.mx.Unlock()

	p.watchLk.Lock()
	if _, wok := p.watching[name]; wok {
		p.watchLk.Unlock()
		return false, fmt.Errorf("key has active subscriptions")
	}
	p.watchLk.Unlock()

	ti, ok := p.topics[name]
	if ok {
		p.closeTopic(name, ti)
		<-ti.finished
	}

	return ok, nil
}

// closeTopic must be called under the PubSubValueStore's mutex
func (p *PubsubValueStore) closeTopic(key string, ti *topicInfo) {
	ti.cancel()
	ti.sub.Cancel()
	ti.evts.Cancel()
	_ = ti.topic.Close()
	delete(p.topics, key)
}

func (p *PubsubValueStore) handleSubscription(ctx context.Context, ti *topicInfo, key string) {
	defer func() {
		close(ti.finished)

		p.mx.Lock()
		defer p.mx.Unlock()

		p.closeTopic(key, ti)
	}()

	newMsg := make(chan []byte)
	go func() {
		defer close(newMsg)
		for {
			data, err := p.handleNewMsgs(ctx, ti.sub, key)
			if err != nil {
				return
			}
			select {
			case newMsg <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	newPeerData := make(chan []byte)
	go func() {
		defer close(newPeerData)
		for {
			data, err := p.handleNewPeer(ctx, ti.evts, key)
			if err == nil {
				if data != nil {
					select {
					case newPeerData <- data:
					case <-ctx.Done():
						return
					}
				}
			} else {
				select {
				case <-ctx.Done():
					return
				default:
					log.Errorf("PubsubPeerJoin: error interacting with new peer: %s", err)
				}
			}
		}
	}()

	for {
		var data []byte
		var ok bool
		select {
		case data, ok = <-newMsg:
			if !ok {
				return
			}
		case data, ok = <-newPeerData:
			if !ok {
				return
			}
		case <-ctx.Done():
			return
		}

		ti.dbWriteMx.Lock()
		recCmp, err := p.putLocal(ctx, ti, key, data)
		ti.dbWriteMx.Unlock()
		if recCmp > 0 {
			if err != nil {
				log.Warnf("PubsubResolve: error writing update for %s: %s", key, err)
			}
			p.notifyWatchers(key, data)
		}
	}
}

func (p *PubsubValueStore) handleNewMsgs(ctx context.Context, sub *pubsub.Subscription, key string) ([]byte, error) {
	msg, err := sub.Next(ctx)
	if err != nil {
		if err != context.Canceled {
			log.Warnf("PubsubResolve: subscription error in %s: %s", key, err.Error())
		}
		return nil, err
	}
	return msg.GetData(), nil
}

func (p *PubsubValueStore) handleNewPeer(ctx context.Context, peerEvtHandler *pubsub.TopicEventHandler, key string) ([]byte, error) {
	for ctx.Err() == nil {
		peerEvt, err := peerEvtHandler.NextPeerEvent(ctx)
		if err != nil {
			if err != context.Canceled {
				log.Warnf("PubsubNewPeer: subscription error in %s: %s", key, err.Error())
			}
			return nil, err
		}

		if peerEvt.Type != pubsub.PeerJoin {
			continue
		}

		pid := peerEvt.Peer
		value, err := p.fetch.Fetch(ctx, pid, key)
		if err == nil {
			return value, nil
		}
		log.Debugf("failed to fetch latest pubsub value for key '%s' from peer '%s': %s", key, pid, err)
	}
	return nil, ctx.Err()
}

func (p *PubsubValueStore) notifyWatchers(key string, data []byte) {
	p.watchLk.Lock()
	defer p.watchLk.Unlock()
	sg, ok := p.watching[key]
	if !ok {
		return
	}

	for watcher := range sg.listeners {
		select {
		case <-watcher:
			watcher <- data
		case watcher <- data:
		}
	}
}

func WithRebroadcastInterval(duration time.Duration) Option {
	return func(store *PubsubValueStore) error {
		store.rebroadcastInterval = duration
		return nil
	}
}

func WithRebroadcastInitialDelay(duration time.Duration) Option {
	return func(store *PubsubValueStore) error {
		store.rebroadcastInitialDelay = duration
		return nil
	}
}

// WithDatastore returns an option that overrides the default datastore.
func WithDatastore(datastore ds.Datastore) Option {
	return func(store *PubsubValueStore) error {
		store.ds = datastore
		return nil
	}
}
