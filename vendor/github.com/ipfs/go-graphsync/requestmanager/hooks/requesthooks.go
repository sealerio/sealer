package hooks

import (
	"github.com/hannahhoward/go-pubsub"
	"github.com/ipld/go-ipld-prime/traversal"
	peer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

// OutgoingRequestHooks is a set of incoming request hooks that can be processed
type OutgoingRequestHooks struct {
	pubSub *pubsub.PubSub
}

type internalRequestHookEvent struct {
	p           peer.ID
	request     graphsync.RequestData
	hookActions *requestHookActions
}

func requestHooksDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalRequestHookEvent)
	hook := subscriberFn.(graphsync.OnOutgoingRequestHook)
	hook(ie.p, ie.request, ie.hookActions)
	return nil
}

// NewRequestHooks returns a new list of incoming request hooks
func NewRequestHooks() *OutgoingRequestHooks {
	return &OutgoingRequestHooks{
		pubSub: pubsub.New(requestHooksDispatcher),
	}
}

// Register registers an extension to process outgoing requests
func (orh *OutgoingRequestHooks) Register(hook graphsync.OnOutgoingRequestHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(orh.pubSub.Subscribe(hook))
}

// RequestResult is the outcome of running requesthooks
type RequestResult struct {
	PersistenceOption string
	CustomChooser     traversal.LinkTargetNodePrototypeChooser
}

// ProcessRequestHooks runs request hooks against an outgoing request
func (orh *OutgoingRequestHooks) ProcessRequestHooks(p peer.ID, request graphsync.RequestData) RequestResult {
	rha := &requestHookActions{}
	_ = orh.pubSub.Publish(internalRequestHookEvent{p, request, rha})
	return rha.result()
}

type requestHookActions struct {
	persistenceOption  string
	nodeBuilderChooser traversal.LinkTargetNodePrototypeChooser
}

func (rha *requestHookActions) result() RequestResult {
	return RequestResult{
		PersistenceOption: rha.persistenceOption,
		CustomChooser:     rha.nodeBuilderChooser,
	}
}

func (rha *requestHookActions) UsePersistenceOption(name string) {
	rha.persistenceOption = name
}

func (rha *requestHookActions) UseLinkTargetNodePrototypeChooser(nodeBuilderChooser traversal.LinkTargetNodePrototypeChooser) {
	rha.nodeBuilderChooser = nodeBuilderChooser
}
