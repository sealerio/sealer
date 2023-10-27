package hooks

import (
	"github.com/hannahhoward/go-pubsub"
	peer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
)

// ErrPaused indicates a request should stop processing, but only cause it's paused
type ErrPaused struct{}

func (e ErrPaused) Error() string { return "request has been paused" }

// OutgoingBlockHooks is a set of outgoing block hooks that can be processed
type OutgoingBlockHooks struct {
	pubSub *pubsub.PubSub
}

type internalBlockHookEvent struct {
	p       peer.ID
	request graphsync.RequestData
	block   graphsync.BlockData
	bha     *blockHookActions
}

func blockHookDispatcher(event pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie := event.(internalBlockHookEvent)
	hook := subscriberFn.(graphsync.OnOutgoingBlockHook)
	hook(ie.p, ie.request, ie.block, ie.bha)
	return ie.bha.err
}

// NewBlockHooks returns a new list of outgoing block hooks
func NewBlockHooks() *OutgoingBlockHooks {
	return &OutgoingBlockHooks{pubSub: pubsub.New(blockHookDispatcher)}
}

// Register registers an hook to process outgoing blocks in a response
func (obh *OutgoingBlockHooks) Register(hook graphsync.OnOutgoingBlockHook) graphsync.UnregisterHookFunc {
	return graphsync.UnregisterHookFunc(obh.pubSub.Subscribe(hook))
}

// BlockResult is the result of processing block hooks
type BlockResult struct {
	Err        error
	Extensions []graphsync.ExtensionData
}

// ProcessBlockHooks runs block hooks against a request and block data
func (obh *OutgoingBlockHooks) ProcessBlockHooks(p peer.ID, request graphsync.RequestData, blockData graphsync.BlockData) BlockResult {
	bha := &blockHookActions{}
	_ = obh.pubSub.Publish(internalBlockHookEvent{p, request, blockData, bha})
	return bha.result()
}

type blockHookActions struct {
	err        error
	extensions []graphsync.ExtensionData
}

func (bha *blockHookActions) result() BlockResult {
	return BlockResult{bha.err, bha.extensions}
}

func (bha *blockHookActions) SendExtensionData(data graphsync.ExtensionData) {
	bha.extensions = append(bha.extensions, data)
}

func (bha *blockHookActions) TerminateWithError(err error) {
	bha.err = err
}

func (bha *blockHookActions) PauseResponse() {
	bha.err = ErrPaused{}
}
