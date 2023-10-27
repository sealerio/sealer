package hooks

import (
	"errors"

	"github.com/hannahhoward/go-pubsub"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/traversal"
	peer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

// PersistenceOptions is an interface for getting loaders by name
type PersistenceOptions interface {
	GetLinkSystem(name string) (ipld.LinkSystem, bool)
}

// IncomingRequestQueuedHooks is a set of incoming request queued hooks that can be processed.
type IncomingRequestQueuedHooks struct {
	pubSub *pubsub.PubSub
}

type internalRequestQueuedHookEvent struct {
	p       peer.ID
	request graphsync.RequestData
}

func requestQueuedHookDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalRequestQueuedHookEvent)
	hook := subscriberFn.(graphsync.OnIncomingRequestQueuedHook)
	hook(ie.p, ie.request)
	return nil
}

// Register registers an extension to process new incoming requests.
func (rqh *IncomingRequestQueuedHooks) Register(hook graphsync.OnIncomingRequestQueuedHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(rqh.pubSub.Subscribe(hook))
}

// NewRequestQueuedHooks returns a new list of incoming request queued hooks.
func NewRequestQueuedHooks() *IncomingRequestQueuedHooks {
	return &IncomingRequestQueuedHooks{
		pubSub: pubsub.New(requestQueuedHookDispatcher),
	}
}

// ProcessRequestQueuedHooks runs request hooks against an incoming queued request.
func (rqh *IncomingRequestQueuedHooks) ProcessRequestQueuedHooks(p peer.ID, request graphsync.RequestData) {
	_ = rqh.pubSub.Publish(internalRequestQueuedHookEvent{p, request})
}

// IncomingRequestHooks is a set of incoming request hooks that can be processed
type IncomingRequestHooks struct {
	persistenceOptions PersistenceOptions
	pubSub             *pubsub.PubSub
}

type internalRequestHookEvent struct {
	p       peer.ID
	request graphsync.RequestData
	rha     *requestHookActions
}

func requestHookDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalRequestHookEvent)
	hook := subscriberFn.(graphsync.OnIncomingRequestHook)
	hook(ie.p, ie.request, ie.rha)
	return ie.rha.err
}

// NewRequestHooks returns a new list of incoming request hooks
func NewRequestHooks(persistenceOptions PersistenceOptions) *IncomingRequestHooks {
	return &IncomingRequestHooks{
		persistenceOptions: persistenceOptions,
		pubSub:             pubsub.New(requestHookDispatcher),
	}
}

// Register registers an extension to process new incoming requests
func (irh *IncomingRequestHooks) Register(hook graphsync.OnIncomingRequestHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(irh.pubSub.Subscribe(hook))
}

// RequestResult is the outcome of running requesthooks
type RequestResult struct {
	IsValidated      bool
	IsPaused         bool
	CustomLinkSystem ipld.LinkSystem
	CustomChooser    traversal.LinkTargetNodePrototypeChooser
	Err              error
	Extensions       []graphsync.ExtensionData
}

// ProcessRequestHooks runs request hooks against an incoming request
func (irh *IncomingRequestHooks) ProcessRequestHooks(p peer.ID, request graphsync.RequestData) RequestResult {
	ha := &requestHookActions{
		persistenceOptions: irh.persistenceOptions,
	}
	_ = irh.pubSub.Publish(internalRequestHookEvent{p, request, ha})
	return ha.result()
}

type requestHookActions struct {
	persistenceOptions PersistenceOptions
	isValidated        bool
	isPaused           bool
	err                error
	linkSystem         ipld.LinkSystem
	chooser            traversal.LinkTargetNodePrototypeChooser
	extensions         []graphsync.ExtensionData
}

func (ha *requestHookActions) result() RequestResult {
	return RequestResult{
		IsValidated:      ha.isValidated,
		IsPaused:         ha.isPaused,
		CustomLinkSystem: ha.linkSystem,
		CustomChooser:    ha.chooser,
		Err:              ha.err,
		Extensions:       ha.extensions,
	}
}

func (ha *requestHookActions) SendExtensionData(ext graphsync.ExtensionData) {
	ha.extensions = append(ha.extensions, ext)
}

func (ha *requestHookActions) TerminateWithError(err error) {
	ha.err = err
}

func (ha *requestHookActions) ValidateRequest() {
	ha.isValidated = true
}

func (ha *requestHookActions) UsePersistenceOption(name string) {
	linkSystem, ok := ha.persistenceOptions.GetLinkSystem(name)
	if !ok {
		ha.TerminateWithError(errors.New("unknown loader option"))
		return
	}
	ha.linkSystem = linkSystem
}

func (ha *requestHookActions) UseLinkTargetNodePrototypeChooser(chooser traversal.LinkTargetNodePrototypeChooser) {
	ha.chooser = chooser
}

func (ha *requestHookActions) PauseResponse() {
	ha.isPaused = true
}
