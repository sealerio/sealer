package hooks

import (
	"github.com/hannahhoward/go-pubsub"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

// ErrPaused indicates a request should stop processing, but only cause it's paused
type ErrPaused struct{}

func (e ErrPaused) Error() string { return "request has been paused" }

// IncomingResponseHooks is a set of incoming response hooks that can be processed
type IncomingResponseHooks struct {
	pubSub *pubsub.PubSub
}

type internalResponseHookEvent struct {
	p        peer.ID
	response graphsync.ResponseData
	rha      *updateHookActions
}

func responseHookDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalResponseHookEvent)
	hook := subscriberFn.(graphsync.OnIncomingResponseHook)
	hook(ie.p, ie.response, ie.rha)
	return ie.rha.err
}

// NewResponseHooks returns a new list of incoming request hooks
func NewResponseHooks() *IncomingResponseHooks {
	return &IncomingResponseHooks{pubSub: pubsub.New(responseHookDispatcher)}
}

// Register registers an extension to process incoming responses
func (irh *IncomingResponseHooks) Register(hook graphsync.OnIncomingResponseHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(irh.pubSub.Subscribe(hook))
}

// UpdateResult is the outcome of running response hooks
type UpdateResult struct {
	Err        error
	Extensions []graphsync.ExtensionData
}

// ProcessResponseHooks runs response hooks against an incoming response
func (irh *IncomingResponseHooks) ProcessResponseHooks(p peer.ID, response graphsync.ResponseData) UpdateResult {
	rha := &updateHookActions{}
	_ = irh.pubSub.Publish(internalResponseHookEvent{p, response, rha})
	return rha.result()
}

type updateHookActions struct {
	err        error
	extensions []graphsync.ExtensionData
}

func (rha *updateHookActions) result() UpdateResult {
	return UpdateResult{
		Err:        rha.err,
		Extensions: rha.extensions,
	}
}

func (rha *updateHookActions) TerminateWithError(err error) {
	rha.err = err
}

func (rha *updateHookActions) UpdateRequestWithExtensions(extensions ...graphsync.ExtensionData) {
	rha.extensions = append(rha.extensions, extensions...)
}

func (rha *updateHookActions) PauseRequest() {
	rha.err = ErrPaused{}
}
