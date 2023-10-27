package hooks

import (
	"github.com/hannahhoward/go-pubsub"
	peer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

// RequestUpdatedHooks manages and runs hooks for request updates
type RequestUpdatedHooks struct {
	pubSub *pubsub.PubSub
}

type internalRequestUpdateEvent struct {
	p       peer.ID
	request graphsync.RequestData
	update  graphsync.RequestData
	uha     *updateHookActions
}

func updateHookDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalRequestUpdateEvent)
	hook := subscriberFn.(graphsync.OnRequestUpdatedHook)
	hook(ie.p, ie.request, ie.update, ie.uha)
	return ie.uha.err
}

// NewUpdateHooks returns a new list of request updated hooks
func NewUpdateHooks() *RequestUpdatedHooks {
	return &RequestUpdatedHooks{pubSub: pubsub.New(updateHookDispatcher)}
}

// Register registers an hook to process updates to requests
func (ruh *RequestUpdatedHooks) Register(hook graphsync.OnRequestUpdatedHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(ruh.pubSub.Subscribe(hook))
}

// UpdateResult is the result of running update hooks
type UpdateResult struct {
	Err        error
	Unpause    bool
	Extensions []graphsync.ExtensionData
}

// ProcessUpdateHooks runs request hooks against an incoming request
func (ruh *RequestUpdatedHooks) ProcessUpdateHooks(p peer.ID, request graphsync.RequestData, update graphsync.RequestData) UpdateResult {
	ha := &updateHookActions{}
	_ = ruh.pubSub.Publish(internalRequestUpdateEvent{p, request, update, ha})
	return ha.result()
}

type updateHookActions struct {
	err        error
	unpause    bool
	extensions []graphsync.ExtensionData
}

func (uha *updateHookActions) result() UpdateResult {
	return UpdateResult{uha.err, uha.unpause, uha.extensions}
}

func (uha *updateHookActions) SendExtensionData(data graphsync.ExtensionData) {
	uha.extensions = append(uha.extensions, data)
}

func (uha *updateHookActions) TerminateWithError(err error) {
	uha.err = err
}

func (uha *updateHookActions) UnpauseResponse() {
	uha.unpause = true
}
