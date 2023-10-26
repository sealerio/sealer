/*
Package responseassembler assembles responses that are queued for sending in outgoing messages

The response assembler's Transaction method allows a caller to specify response actions that will go into a single
libp2p2 message. The response assembler will also deduplicate blocks that have already been sent over the network in
a previous message
*/
package responseassembler

import (
	"context"

	"github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/notifications"
	"github.com/ipfs/go-graphsync/peermanager"
)

// Transaction is a series of operations that should be send together in a single response
type Transaction func(ResponseBuilder) error

// ResponseBuilder is a limited interface for assembling responses inside a transaction, so that they are included
// in the same message on the protocol
type ResponseBuilder interface {
	// SendResponse adds a response to this transaction.
	SendResponse(
		link ipld.Link,
		data []byte,
	) graphsync.BlockData

	// SendExtensionData adds extension data to the transaction.
	SendExtensionData(graphsync.ExtensionData)

	// ClearRequest removes all tracking for this request.
	ClearRequest()

	// FinishRequest completes the response to a request.
	FinishRequest() graphsync.ResponseStatusCode

	// FinishWithError end the response due to an error
	FinishWithError(status graphsync.ResponseStatusCode)

	// PauseRequest temporarily halts responding to the request
	PauseRequest()

	// AddNotifee adds a notifee to be notified about the response to request.
	AddNotifee(notifications.Notifee)
}

// PeerMessageHandler is an interface that can queue a response for a given peer to go out over the network
// If blkSize > 0, message building may block until enough memory has been freed from the queues to allocate the message.
type PeerMessageHandler interface {
	AllocateAndBuildMessage(p peer.ID, blkSize uint64, buildResponseFn func(*gsmsg.Builder), notifees []notifications.Notifee)
}

// ResponseAssembler manages assembling responses to go out over the network
// in libp2p messages
type ResponseAssembler struct {
	*peermanager.PeerManager
	peerHandler PeerMessageHandler
	ctx         context.Context
}

// New generates a new ResponseAssembler for sending responses
func New(ctx context.Context, peerHandler PeerMessageHandler) *ResponseAssembler {
	return &ResponseAssembler{
		PeerManager: peermanager.New(ctx, func(ctx context.Context, p peer.ID) peermanager.PeerHandler {
			return newTracker()
		}),
		ctx:         ctx,
		peerHandler: peerHandler,
	}
}

// DedupKey indicates that outgoing blocks should be deduplicated in a seperate bucket (only with requests that share
// supplied key string)
func (ra *ResponseAssembler) DedupKey(p peer.ID, requestID graphsync.RequestID, key string) {
	ra.GetProcess(p).(*peerLinkTracker).DedupKey(requestID, key)
}

// IgnoreBlocks indicates that a list of keys should be ignored when sending blocks
func (ra *ResponseAssembler) IgnoreBlocks(p peer.ID, requestID graphsync.RequestID, links []ipld.Link) {
	ra.GetProcess(p).(*peerLinkTracker).IgnoreBlocks(requestID, links)
}

// SkipFirstBlocks tells the assembler for the given request to not send the first N blocks
func (ra *ResponseAssembler) SkipFirstBlocks(p peer.ID, requestID graphsync.RequestID, skipFirstBlocks int64) {
	ra.GetProcess(p).(*peerLinkTracker).SkipFirstBlocks(requestID, skipFirstBlocks)
}

// Transaction builds a response, and queues it for sending in the next outgoing message
func (ra *ResponseAssembler) Transaction(p peer.ID, requestID graphsync.RequestID, transaction Transaction) error {
	rb := &responseBuilder{
		requestID:   requestID,
		linkTracker: ra.GetProcess(p).(*peerLinkTracker),
	}
	err := transaction(rb)
	ra.execute(p, rb.operations, rb.notifees)
	return err
}

func (ra *ResponseAssembler) execute(p peer.ID, operations []responseOperation, notifees []notifications.Notifee) {
	size := uint64(0)
	for _, op := range operations {
		size += op.size()
	}
	ra.peerHandler.AllocateAndBuildMessage(p, size, func(builder *gsmsg.Builder) {
		for _, op := range operations {
			op.build(builder)
		}
	}, notifees)
}
