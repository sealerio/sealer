package requestmanager

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hannahhoward/go-pubsub"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/ipldutil"
	"github.com/ipfs/go-graphsync/listeners"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/messagequeue"
	"github.com/ipfs/go-graphsync/metadata"
	"github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/go-graphsync/notifications"
	"github.com/ipfs/go-graphsync/requestmanager/executor"
	"github.com/ipfs/go-graphsync/requestmanager/hooks"
	"github.com/ipfs/go-graphsync/requestmanager/types"
	"github.com/ipfs/go-graphsync/taskqueue"
)

// The code in this file implements the public interface of the request manager.
// Functions in this file operate outside the internal thread and should
// NOT modify the internal state of the RequestManager.

var log = logging.Logger("graphsync")

const (
	// defaultPriority is the default priority for requests sent by graphsync
	defaultPriority = graphsync.Priority(0)
)

type state uint64

const (
	queued state = iota
	running
	paused
)

type inProgressRequestStatus struct {
	ctx              context.Context
	startTime        time.Time
	cancelFn         func()
	p                peer.ID
	terminalError    error
	pauseMessages    chan struct{}
	state            state
	lastResponse     atomic.Value
	onTerminated     []chan<- error
	request          gsmsg.GraphSyncRequest
	doNotSendCids    *cid.Set
	nodeStyleChooser traversal.LinkTargetNodePrototypeChooser
	inProgressChan   chan graphsync.ResponseProgress
	inProgressErr    chan error
	traverser        ipldutil.Traverser
	traverserCancel  context.CancelFunc
}

// PeerHandler is an interface that can send requests to peers
type PeerHandler interface {
	AllocateAndBuildMessage(p peer.ID, blkSize uint64, buildMessageFn func(*gsmsg.Builder), notifees []notifications.Notifee)
}

// AsyncLoader is an interface for loading links asynchronously, returning
// results as new responses are processed
type AsyncLoader interface {
	StartRequest(graphsync.RequestID, string) error
	ProcessResponse(responses map[graphsync.RequestID]metadata.Metadata,
		blks []blocks.Block)
	AsyncLoad(p peer.ID, requestID graphsync.RequestID, link ipld.Link, linkContext ipld.LinkContext) <-chan types.AsyncLoadResult
	CompleteResponsesFor(requestID graphsync.RequestID)
	CleanupRequest(p peer.ID, requestID graphsync.RequestID)
}

// RequestManager tracks outgoing requests and processes incoming reponses
// to them.
type RequestManager struct {
	ctx             context.Context
	cancel          context.CancelFunc
	messages        chan requestManagerMessage
	peerHandler     PeerHandler
	rc              *responseCollector
	asyncLoader     AsyncLoader
	disconnectNotif *pubsub.PubSub
	linkSystem      ipld.LinkSystem
	connManager     network.ConnManager
	// maximum number of links to traverse per request. A value of zero = infinity, or no limit
	maxLinksPerRequest uint64

	// dont touch out side of run loop
	nextRequestID                      graphsync.RequestID
	inProgressRequestStatuses          map[graphsync.RequestID]*inProgressRequestStatus
	requestHooks                       RequestHooks
	responseHooks                      ResponseHooks
	networkErrorListeners              *listeners.NetworkErrorListeners
	outgoingRequestProcessingListeners *listeners.OutgoingRequestProcessingListeners
	requestQueue                       taskqueue.TaskQueue
}

type requestManagerMessage interface {
	handle(rm *RequestManager)
}

// RequestHooks run for new requests
type RequestHooks interface {
	ProcessRequestHooks(p peer.ID, request graphsync.RequestData) hooks.RequestResult
}

// ResponseHooks run for new responses
type ResponseHooks interface {
	ProcessResponseHooks(p peer.ID, response graphsync.ResponseData) hooks.UpdateResult
}

// New generates a new request manager from a context, network, and selectorQuerier
func New(ctx context.Context,
	asyncLoader AsyncLoader,
	linkSystem ipld.LinkSystem,
	requestHooks RequestHooks,
	responseHooks ResponseHooks,
	networkErrorListeners *listeners.NetworkErrorListeners,
	outgoingRequestProcessingListeners *listeners.OutgoingRequestProcessingListeners,
	requestQueue taskqueue.TaskQueue,
	connManager network.ConnManager,
	maxLinksPerRequest uint64,
) *RequestManager {
	ctx, cancel := context.WithCancel(ctx)
	return &RequestManager{
		ctx:                                ctx,
		cancel:                             cancel,
		asyncLoader:                        asyncLoader,
		disconnectNotif:                    pubsub.New(disconnectDispatcher),
		linkSystem:                         linkSystem,
		rc:                                 newResponseCollector(ctx),
		messages:                           make(chan requestManagerMessage, 16),
		inProgressRequestStatuses:          make(map[graphsync.RequestID]*inProgressRequestStatus),
		requestHooks:                       requestHooks,
		responseHooks:                      responseHooks,
		networkErrorListeners:              networkErrorListeners,
		outgoingRequestProcessingListeners: outgoingRequestProcessingListeners,
		requestQueue:                       requestQueue,
		connManager:                        connManager,
		maxLinksPerRequest:                 maxLinksPerRequest,
	}
}

// SetDelegate specifies who will send messages out to the internet.
func (rm *RequestManager) SetDelegate(peerHandler PeerHandler) {
	rm.peerHandler = peerHandler
}

type inProgressRequest struct {
	requestID     graphsync.RequestID
	request       gsmsg.GraphSyncRequest
	incoming      chan graphsync.ResponseProgress
	incomingError chan error
}

// NewRequest initiates a new GraphSync request to the given peer.
func (rm *RequestManager) NewRequest(ctx context.Context,
	p peer.ID,
	root ipld.Link,
	selectorNode ipld.Node,
	extensions ...graphsync.ExtensionData) (<-chan graphsync.ResponseProgress, <-chan error) {
	if _, err := selector.ParseSelector(selectorNode); err != nil {
		return rm.singleErrorResponse(fmt.Errorf("invalid selector spec"))
	}

	inProgressRequestChan := make(chan inProgressRequest)

	rm.send(&newRequestMessage{p, root, selectorNode, extensions, inProgressRequestChan}, ctx.Done())
	var receivedInProgressRequest inProgressRequest
	select {
	case <-rm.ctx.Done():
		return rm.emptyResponse()
	case receivedInProgressRequest = <-inProgressRequestChan:
	}

	// If the connection to the peer is disconnected, fire an error
	unsub := rm.listenForDisconnect(p, func(neterr error) {
		rm.networkErrorListeners.NotifyNetworkErrorListeners(p, receivedInProgressRequest.request, neterr)
	})

	return rm.rc.collectResponses(ctx,
		receivedInProgressRequest.incoming,
		receivedInProgressRequest.incomingError,
		func() {
			rm.cancelRequestAndClose(receivedInProgressRequest.requestID,
				receivedInProgressRequest.incoming,
				receivedInProgressRequest.incomingError)
		},
		// Once the request has completed, stop listening for disconnect events
		unsub,
	)
}

// Dispatch the Disconnect event to subscribers
func disconnectDispatcher(p pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	listener := subscriberFn.(func(peer.ID))
	listener(p.(peer.ID))
	return nil
}

// Listen for the Disconnect event for the given peer
func (rm *RequestManager) listenForDisconnect(p peer.ID, onDisconnect func(neterr error)) func() {
	// Subscribe to Disconnect notifications
	return rm.disconnectNotif.Subscribe(func(evtPeer peer.ID) {
		// If the peer is the one we're interested in, call the listener
		if evtPeer == p {
			onDisconnect(fmt.Errorf("disconnected from peer %s", p))
		}
	})
}

// Disconnected is called when a peer disconnects
func (rm *RequestManager) Disconnected(p peer.ID) {
	// Notify any listeners that a peer has disconnected
	_ = rm.disconnectNotif.Publish(p)
}

func (rm *RequestManager) emptyResponse() (chan graphsync.ResponseProgress, chan error) {
	ch := make(chan graphsync.ResponseProgress)
	close(ch)
	errCh := make(chan error)
	close(errCh)
	return ch, errCh
}

func (rm *RequestManager) singleErrorResponse(err error) (chan graphsync.ResponseProgress, chan error) {
	ch := make(chan graphsync.ResponseProgress)
	close(ch)
	errCh := make(chan error, 1)
	errCh <- err
	close(errCh)
	return ch, errCh
}

func (rm *RequestManager) cancelRequestAndClose(requestID graphsync.RequestID,
	incomingResponses chan graphsync.ResponseProgress,
	incomingErrors chan error) {
	cancelMessageChannel := rm.messages
	for cancelMessageChannel != nil || incomingResponses != nil || incomingErrors != nil {
		select {
		case cancelMessageChannel <- &cancelRequestMessage{requestID, nil, nil}:
			cancelMessageChannel = nil
		// clear out any remaining responses, in case and "incoming reponse"
		// messages get processed before our cancel message
		case _, ok := <-incomingResponses:
			if !ok {
				incomingResponses = nil
			}
		case _, ok := <-incomingErrors:
			if !ok {
				incomingErrors = nil
			}
		case <-rm.ctx.Done():
			return
		}
	}
}

// CancelRequest cancels the given request ID and waits for the request to terminate
func (rm *RequestManager) CancelRequest(ctx context.Context, requestID graphsync.RequestID) error {
	terminated := make(chan error, 1)
	rm.send(&cancelRequestMessage{requestID, terminated, graphsync.RequestClientCancelledErr{}}, ctx.Done())
	select {
	case <-rm.ctx.Done():
		return errors.New("context cancelled")
	case err := <-terminated:
		return err
	}
}

// ProcessResponses ingests the given responses from the network and
// and updates the in progress requests based on those responses.
func (rm *RequestManager) ProcessResponses(p peer.ID, responses []gsmsg.GraphSyncResponse,
	blks []blocks.Block) {
	rm.send(&processResponseMessage{p, responses, blks}, nil)
}

// UnpauseRequest unpauses a request that was paused in a block hook based request ID
// Can also send extensions with unpause
func (rm *RequestManager) UnpauseRequest(requestID graphsync.RequestID, extensions ...graphsync.ExtensionData) error {
	response := make(chan error, 1)
	rm.send(&unpauseRequestMessage{requestID, extensions, response}, nil)
	select {
	case <-rm.ctx.Done():
		return errors.New("context cancelled")
	case err := <-response:
		return err
	}
}

// PauseRequest pauses an in progress request (may take 1 or more blocks to process)
func (rm *RequestManager) PauseRequest(requestID graphsync.RequestID) error {
	response := make(chan error, 1)
	rm.send(&pauseRequestMessage{requestID, response}, nil)
	select {
	case <-rm.ctx.Done():
		return errors.New("context cancelled")
	case err := <-response:
		return err
	}
}

// GetRequestTask gets data for the given task in the request queue
func (rm *RequestManager) GetRequestTask(p peer.ID, task *peertask.Task, requestExecutionChan chan executor.RequestTask) {
	rm.send(&getRequestTaskMessage{p, task, requestExecutionChan}, nil)
}

// ReleaseRequestTask releases a task request the requestQueue
func (rm *RequestManager) ReleaseRequestTask(p peer.ID, task *peertask.Task, err error) {
	rm.send(&releaseRequestTaskMessage{p, task, err}, nil)
}

// SendRequest sends a request to the message queue
func (rm *RequestManager) SendRequest(p peer.ID, request gsmsg.GraphSyncRequest) {
	sub := notifications.NewTopicDataSubscriber(&reqSubscriber{p, request, rm.networkErrorListeners})
	failNotifee := notifications.Notifee{Data: requestNetworkError, Subscriber: sub}
	rm.peerHandler.AllocateAndBuildMessage(p, 0, func(builder *gsmsg.Builder) {
		builder.AddRequest(request)
	}, []notifications.Notifee{failNotifee})
}

// Startup starts processing for the WantManager.
func (rm *RequestManager) Startup() {
	go rm.run()
}

// Shutdown ends processing for the want manager.
func (rm *RequestManager) Shutdown() {
	rm.cancel()
}

func (rm *RequestManager) send(message requestManagerMessage, done <-chan struct{}) {
	select {
	case <-rm.ctx.Done():
	case <-done:
	case rm.messages <- message:
	}
}

type reqSubscriber struct {
	p                     peer.ID
	request               gsmsg.GraphSyncRequest
	networkErrorListeners *listeners.NetworkErrorListeners
}

func (r *reqSubscriber) OnNext(topic notifications.Topic, event notifications.Event) {
	mqEvt, isMQEvt := event.(messagequeue.Event)
	if !isMQEvt || mqEvt.Name != messagequeue.Error {
		return
	}

	r.networkErrorListeners.NotifyNetworkErrorListeners(r.p, r.request, mqEvt.Err)
	//r.re.networkError <- mqEvt.Err
	//r.re.terminateRequest()
}

func (r reqSubscriber) OnClose(topic notifications.Topic) {
}

const requestNetworkError = "request_network_error"
