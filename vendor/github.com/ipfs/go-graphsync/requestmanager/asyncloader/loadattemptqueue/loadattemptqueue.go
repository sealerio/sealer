package loadattemptqueue

import (
	"errors"

	"github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/requestmanager/types"
)

// LoadRequest is a request to load the given link for the given request id,
// with results returned to the given channel
type LoadRequest struct {
	p           peer.ID
	requestID   graphsync.RequestID
	link        ipld.Link
	linkContext ipld.LinkContext
	resultChan  chan types.AsyncLoadResult
}

// NewLoadRequest returns a new LoadRequest for the given request id, link,
// and results channel
func NewLoadRequest(
	p peer.ID,
	requestID graphsync.RequestID,
	link ipld.Link,
	linkContext ipld.LinkContext,
	resultChan chan types.AsyncLoadResult) LoadRequest {
	return LoadRequest{p, requestID, link, linkContext, resultChan}
}

// LoadAttempter attempts to load a link to an array of bytes
// and returns an async load result
type LoadAttempter func(peer.ID, graphsync.RequestID, ipld.Link, ipld.LinkContext) types.AsyncLoadResult

// LoadAttemptQueue attempts to load using the load attempter, and then can
// place requests on a retry queue
type LoadAttemptQueue struct {
	loadAttempter  LoadAttempter
	pausedRequests []LoadRequest
}

// New initializes a new AsyncLoader from loadAttempter function
func New(loadAttempter LoadAttempter) *LoadAttemptQueue {
	return &LoadAttemptQueue{
		loadAttempter: loadAttempter,
	}
}

// AttemptLoad attempts to loads the given load request, and if retry is true
// it saves the loadrequest for retrying later
func (laq *LoadAttemptQueue) AttemptLoad(lr LoadRequest, retry bool) {
	response := laq.loadAttempter(lr.p, lr.requestID, lr.link, lr.linkContext)
	if response.Err != nil || response.Data != nil {
		lr.resultChan <- response
		close(lr.resultChan)
		return
	}
	if !retry {
		laq.terminateWithError("No active request", lr.resultChan)
		return
	}
	laq.pausedRequests = append(laq.pausedRequests, lr)
}

// ClearRequest purges the given request from the queue of load requests
// to retry
func (laq *LoadAttemptQueue) ClearRequest(requestID graphsync.RequestID) {
	pausedRequests := laq.pausedRequests
	laq.pausedRequests = nil
	for _, lr := range pausedRequests {
		if lr.requestID == requestID {
			laq.terminateWithError("No active request", lr.resultChan)
		} else {
			laq.pausedRequests = append(laq.pausedRequests, lr)
		}
	}
}

// RetryLoads attempts loads on all saved load requests that were loaded with
// retry = true
func (laq *LoadAttemptQueue) RetryLoads() {
	// drain buffered
	pausedRequests := laq.pausedRequests
	laq.pausedRequests = nil
	for _, lr := range pausedRequests {
		laq.AttemptLoad(lr, true)
	}
}

func (laq *LoadAttemptQueue) terminateWithError(errMsg string, resultChan chan<- types.AsyncLoadResult) {
	resultChan <- types.AsyncLoadResult{Data: nil, Err: errors.New(errMsg)}
	close(resultChan)
}
