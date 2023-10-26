package hooks

import (
	"github.com/hannahhoward/go-pubsub"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

// IncomingBlockHooks is a set of incoming block hooks that can be processed
type IncomingBlockHooks struct {
	pubSub *pubsub.PubSub
}

type internalBlockHookEvent struct {
	p        peer.ID
	response graphsync.ResponseData
	block    graphsync.BlockData
	rha      *updateHookActions
}

func blockHookDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalBlockHookEvent)
	hook := subscriberFn.(graphsync.OnIncomingBlockHook)
	hook(ie.p, ie.response, ie.block, ie.rha)
	return ie.rha.err
}

// NewBlockHooks returns a new list of incoming request hooks
func NewBlockHooks() *IncomingBlockHooks {
	return &IncomingBlockHooks{pubSub: pubsub.New(blockHookDispatcher)}
}

// Register registers an extension to process incoming responses
func (ibh *IncomingBlockHooks) Register(hook graphsync.OnIncomingBlockHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(ibh.pubSub.Subscribe(hook))
}

// ProcessBlockHooks runs response hooks against an incoming response
func (ibh *IncomingBlockHooks) ProcessBlockHooks(p peer.ID, response graphsync.ResponseData, block graphsync.BlockData) UpdateResult {
	rha := &updateHookActions{}
	_ = ibh.pubSub.Publish(internalBlockHookEvent{p, response, block, rha})
	return rha.result()
}
