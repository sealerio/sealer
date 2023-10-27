package responsemanager

import (
	"context"
	"errors"

	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-peertaskqueue/peertask"
	ipld "github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/ipldutil"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/go-graphsync/notifications"
	"github.com/ipfs/go-graphsync/responsemanager/hooks"
	"github.com/ipfs/go-graphsync/responsemanager/queryexecutor"
	"github.com/ipfs/go-graphsync/responsemanager/responseassembler"
	"github.com/ipfs/go-graphsync/taskqueue"
)

// The code in this file implements the public interface of the response manager.
// Functions in this file operate outside the internal thread and should
// NOT modify the internal state of the ResponseManager.

var log = logging.Logger("graphsync")

type state uint64

const (
	queued state = iota
	running
	paused
)

type inProgressResponseStatus struct {
	ctx        context.Context
	cancelFn   func()
	request    gsmsg.GraphSyncRequest
	loader     ipld.BlockReadOpener
	traverser  ipldutil.Traverser
	signals    queryexecutor.ResponseSignals
	updates    []gsmsg.GraphSyncRequest
	state      state
	subscriber *notifications.TopicDataSubscriber
}

type responseKey struct {
	p         peer.ID
	requestID graphsync.RequestID
}

// RequestHooks is an interface for processing request hooks
type RequestHooks interface {
	ProcessRequestHooks(p peer.ID, request graphsync.RequestData) hooks.RequestResult
}

// RequestQueuedHooks is an interface for processing request queued hooks
type RequestQueuedHooks interface {
	ProcessRequestQueuedHooks(p peer.ID, request graphsync.RequestData)
}

// UpdateHooks is an interface for processing update hooks
type UpdateHooks interface {
	ProcessUpdateHooks(p peer.ID, request graphsync.RequestData, update graphsync.RequestData) hooks.UpdateResult
}

// CompletedListeners is an interface for notifying listeners that responses are complete
type CompletedListeners interface {
	NotifyCompletedListeners(p peer.ID, request graphsync.RequestData, status graphsync.ResponseStatusCode)
}

// CancelledListeners is an interface for notifying listeners that requestor cancelled
type CancelledListeners interface {
	NotifyCancelledListeners(p peer.ID, request graphsync.RequestData)
}

// BlockSentListeners is an interface for notifying listeners that of a block send occuring over the wire
type BlockSentListeners interface {
	NotifyBlockSentListeners(p peer.ID, request graphsync.RequestData, block graphsync.BlockData)
}

// NetworkErrorListeners is an interface for notifying listeners that an error occurred sending a data on the wire
type NetworkErrorListeners interface {
	NotifyNetworkErrorListeners(p peer.ID, request graphsync.RequestData, err error)
}

// ResponseAssembler is an interface that returns sender interfaces for peer responses.
type ResponseAssembler interface {
	DedupKey(p peer.ID, requestID graphsync.RequestID, key string)
	IgnoreBlocks(p peer.ID, requestID graphsync.RequestID, links []ipld.Link)
	SkipFirstBlocks(p peer.ID, requestID graphsync.RequestID, skipCount int64)
	Transaction(p peer.ID, requestID graphsync.RequestID, transaction responseassembler.Transaction) error
}

type responseManagerMessage interface {
	handle(rm *ResponseManager)
}

// ResponseManager handles incoming requests from the network, initiates selector
// traversals, and transmits responses
type ResponseManager struct {
	ctx                   context.Context
	cancelFn              context.CancelFunc
	responseAssembler     ResponseAssembler
	requestHooks          RequestHooks
	linkSystem            ipld.LinkSystem
	requestQueuedHooks    RequestQueuedHooks
	updateHooks           UpdateHooks
	cancelledListeners    CancelledListeners
	completedListeners    CompletedListeners
	blockSentListeners    BlockSentListeners
	networkErrorListeners NetworkErrorListeners
	messages              chan responseManagerMessage
	inProgressResponses   map[responseKey]*inProgressResponseStatus
	maxInProcessRequests  uint64
	connManager           network.ConnManager
	// maximum number of links to traverse per request. A value of zero = infinity, or no limit
	maxLinksPerRequest uint64
	responseQueue      taskqueue.TaskQueue
}

// New creates a new response manager for responding to requests
func New(ctx context.Context,
	linkSystem ipld.LinkSystem,
	responseAssembler ResponseAssembler,
	requestQueuedHooks RequestQueuedHooks,
	requestHooks RequestHooks,
	updateHooks UpdateHooks,
	completedListeners CompletedListeners,
	cancelledListeners CancelledListeners,
	blockSentListeners BlockSentListeners,
	networkErrorListeners NetworkErrorListeners,
	maxInProcessRequests uint64,
	connManager network.ConnManager,
	maxLinksPerRequest uint64,
	responseQueue taskqueue.TaskQueue,
) *ResponseManager {
	ctx, cancelFn := context.WithCancel(ctx)
	messages := make(chan responseManagerMessage, 16)
	rm := &ResponseManager{
		ctx:                   ctx,
		cancelFn:              cancelFn,
		requestHooks:          requestHooks,
		linkSystem:            linkSystem,
		responseAssembler:     responseAssembler,
		requestQueuedHooks:    requestQueuedHooks,
		updateHooks:           updateHooks,
		cancelledListeners:    cancelledListeners,
		completedListeners:    completedListeners,
		blockSentListeners:    blockSentListeners,
		networkErrorListeners: networkErrorListeners,
		messages:              messages,
		inProgressResponses:   make(map[responseKey]*inProgressResponseStatus),
		maxInProcessRequests:  maxInProcessRequests,
		connManager:           connManager,
		maxLinksPerRequest:    maxLinksPerRequest,
		responseQueue:         responseQueue,
	}
	return rm
}

// ProcessRequests processes incoming requests for the given peer
func (rm *ResponseManager) ProcessRequests(ctx context.Context, p peer.ID, requests []gsmsg.GraphSyncRequest) {
	rm.send(&processRequestMessage{p, requests}, ctx.Done())
}

// UnpauseResponse unpauses a response that was previously paused
func (rm *ResponseManager) UnpauseResponse(p peer.ID, requestID graphsync.RequestID, extensions ...graphsync.ExtensionData) error {
	response := make(chan error, 1)
	rm.send(&unpauseRequestMessage{p, requestID, response, extensions}, nil)
	select {
	case <-rm.ctx.Done():
		return errors.New("context cancelled")
	case err := <-response:
		return err
	}
}

// PauseResponse pauses an in progress response (may take 1 or more blocks to process)
func (rm *ResponseManager) PauseResponse(p peer.ID, requestID graphsync.RequestID) error {
	response := make(chan error, 1)
	rm.send(&pauseRequestMessage{p, requestID, response}, nil)
	select {
	case <-rm.ctx.Done():
		return errors.New("context cancelled")
	case err := <-response:
		return err
	}
}

// CancelResponse cancels an in progress response
func (rm *ResponseManager) CancelResponse(p peer.ID, requestID graphsync.RequestID) error {
	response := make(chan error, 1)
	rm.send(&errorRequestMessage{p, requestID, queryexecutor.ErrCancelledByCommand, response}, nil)
	select {
	case <-rm.ctx.Done():
		return errors.New("context cancelled")
	case err := <-response:
		return err
	}
}

// this is a test utility method to force all messages to get processed
func (rm *ResponseManager) synchronize() {
	sync := make(chan error)
	rm.send(&synchronizeMessage{sync}, nil)
	select {
	case <-rm.ctx.Done():
	case <-sync:
	}
}

// StartTask starts the given task from the peer task queue
func (rm *ResponseManager) StartTask(task *peertask.Task, responseTaskChan chan<- queryexecutor.ResponseTask) {
	rm.send(&startTaskRequest{task, responseTaskChan}, nil)
}

// GetUpdates is called to read pending updates for a task and clear them
func (rm *ResponseManager) GetUpdates(p peer.ID, requestID graphsync.RequestID, updatesChan chan<- []gsmsg.GraphSyncRequest) {
	rm.send(&responseUpdateRequest{responseKey{p, requestID}, updatesChan}, nil)
}

// FinishTask marks a task from the task queue as done
func (rm *ResponseManager) FinishTask(task *peertask.Task, err error) {
	rm.send(&finishTaskRequest{task, err}, nil)
}

// CloseWithNetworkError closes a request due to a network error
func (rm *ResponseManager) CloseWithNetworkError(p peer.ID, requestID graphsync.RequestID) {
	rm.send(&errorRequestMessage{p, requestID, queryexecutor.ErrNetworkError, make(chan error, 1)}, nil)
}

func (rm *ResponseManager) send(message responseManagerMessage, done <-chan struct{}) {
	select {
	case <-rm.ctx.Done():
	case <-done:
	case rm.messages <- message:
	}
}

// Startup starts processing for the WantManager.
func (rm *ResponseManager) Startup() {
	go rm.run()
}

// Shutdown ends processing for the want manager.
func (rm *ResponseManager) Shutdown() {
	rm.cancelFn()
}
