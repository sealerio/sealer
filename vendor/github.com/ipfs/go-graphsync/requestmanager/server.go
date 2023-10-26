package requestmanager

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/ipld/go-ipld-prime/traversal/selector"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/cidset"
	"github.com/ipfs/go-graphsync/dedupkey"
	"github.com/ipfs/go-graphsync/ipldutil"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/requestmanager/executor"
	"github.com/ipfs/go-graphsync/requestmanager/hooks"
)

// The code in this file implements the internal thread for the request manager.
// These functions can modify the internal state of the RequestManager

func (rm *RequestManager) run() {
	// NOTE: Do not open any streams or connections from anywhere in this
	// event loop. Really, just don't do anything likely to block.
	defer rm.cleanupInProcessRequests()

	for {
		select {
		case message := <-rm.messages:
			message.handle(rm)
		case <-rm.ctx.Done():
			return
		}
	}
}

func (rm *RequestManager) cleanupInProcessRequests() {
	for _, requestStatus := range rm.inProgressRequestStatuses {
		requestStatus.cancelFn()
	}
}

func (rm *RequestManager) newRequest(p peer.ID, root ipld.Link, selector ipld.Node, extensions []graphsync.ExtensionData) (gsmsg.GraphSyncRequest, chan graphsync.ResponseProgress, chan error) {
	requestID := rm.nextRequestID
	rm.nextRequestID++

	log.Infow("graphsync request initiated", "request id", requestID, "peer", p, "root", root)

	request, hooksResult, err := rm.validateRequest(requestID, p, root, selector, extensions)
	if err != nil {
		rp, err := rm.singleErrorResponse(err)
		return request, rp, err
	}
	doNotSendCidsData, has := request.Extension(graphsync.ExtensionDoNotSendCIDs)
	var doNotSendCids *cid.Set
	if has {
		doNotSendCids, err = cidset.DecodeCidSet(doNotSendCidsData)
		if err != nil {
			rp, err := rm.singleErrorResponse(err)
			return request, rp, err
		}
	} else {
		doNotSendCids = cid.NewSet()
	}
	ctx, cancel := context.WithCancel(rm.ctx)
	requestStatus := &inProgressRequestStatus{
		ctx:              ctx,
		startTime:        time.Now(),
		cancelFn:         cancel,
		p:                p,
		pauseMessages:    make(chan struct{}, 1),
		doNotSendCids:    doNotSendCids,
		request:          request,
		state:            queued,
		nodeStyleChooser: hooksResult.CustomChooser,
		inProgressChan:   make(chan graphsync.ResponseProgress),
		inProgressErr:    make(chan error),
	}
	requestStatus.lastResponse.Store(gsmsg.NewResponse(request.ID(), graphsync.RequestAcknowledged))
	rm.inProgressRequestStatuses[request.ID()] = requestStatus

	rm.connManager.Protect(p, requestID.Tag())
	rm.requestQueue.PushTask(p, peertask.Task{Topic: requestID, Priority: math.MaxInt32, Work: 1})
	return request, requestStatus.inProgressChan, requestStatus.inProgressErr
}

func (rm *RequestManager) requestTask(requestID graphsync.RequestID) executor.RequestTask {
	ipr, ok := rm.inProgressRequestStatuses[requestID]
	if !ok {
		return executor.RequestTask{Empty: true}
	}
	log.Infow("graphsync request processing begins", "request id", requestID, "peer", ipr.p, "total time", time.Since(ipr.startTime))

	var initialRequest bool
	if ipr.traverser == nil {
		initialRequest = true
		var budget *traversal.Budget
		if rm.maxLinksPerRequest > 0 {
			budget = &traversal.Budget{
				NodeBudget: math.MaxInt64,
				LinkBudget: int64(rm.maxLinksPerRequest),
			}
		}
		// the traverser has its own context because we want to fail on block boundaries, in the executor,
		// and make sure all blocks included up to the termination message
		// are processed and passed in the response channel
		ctx, cancel := context.WithCancel(rm.ctx)
		ipr.traverserCancel = cancel
		ipr.traverser = ipldutil.TraversalBuilder{
			Root:     cidlink.Link{Cid: ipr.request.Root()},
			Selector: ipr.request.Selector(),
			Visitor: func(tp traversal.Progress, node ipld.Node, tr traversal.VisitReason) error {
				select {
				case <-ctx.Done():
				case ipr.inProgressChan <- graphsync.ResponseProgress{
					Node:      node,
					Path:      tp.Path,
					LastBlock: tp.LastBlock,
				}:
				}
				return nil
			},
			Chooser:    ipr.nodeStyleChooser,
			LinkSystem: rm.linkSystem,
			Budget:     budget,
		}.Start(ctx)

		inProgressCount := len(rm.inProgressRequestStatuses)
		rm.outgoingRequestProcessingListeners.NotifyOutgoingRequestProcessingListeners(ipr.p, ipr.request, inProgressCount)
	}

	ipr.state = running
	return executor.RequestTask{
		Ctx:            ipr.ctx,
		Request:        ipr.request,
		LastResponse:   &ipr.lastResponse,
		DoNotSendCids:  ipr.doNotSendCids,
		PauseMessages:  ipr.pauseMessages,
		Traverser:      ipr.traverser,
		P:              ipr.p,
		InProgressErr:  ipr.inProgressErr,
		InitialRequest: initialRequest,
		Empty:          false,
	}
}

func (rm *RequestManager) getRequestTask(p peer.ID, task *peertask.Task) executor.RequestTask {
	requestID := task.Topic.(graphsync.RequestID)
	requestExecution := rm.requestTask(requestID)
	if requestExecution.Empty {
		rm.requestQueue.TaskDone(p, task)
	}
	return requestExecution
}

func (rm *RequestManager) terminateRequest(requestID graphsync.RequestID, ipr *inProgressRequestStatus) {
	if ipr.terminalError != nil {
		select {
		case ipr.inProgressErr <- ipr.terminalError:
		case <-rm.ctx.Done():
		}
	}
	rm.connManager.Unprotect(ipr.p, requestID.Tag())
	delete(rm.inProgressRequestStatuses, requestID)
	ipr.cancelFn()
	rm.asyncLoader.CleanupRequest(ipr.p, requestID)
	if ipr.traverser != nil {
		ipr.traverserCancel()
		ipr.traverser.Shutdown(rm.ctx)
	}
	// make sure context is not closed before closing channels (could cause send
	// on close channel otherwise)
	select {
	case <-rm.ctx.Done():
		return
	default:
	}
	close(ipr.inProgressChan)
	close(ipr.inProgressErr)
	for _, onTerminated := range ipr.onTerminated {
		select {
		case <-rm.ctx.Done():
		case onTerminated <- nil:
		}
	}
}

func (rm *RequestManager) releaseRequestTask(p peer.ID, task *peertask.Task, err error) {
	requestID := task.Topic.(graphsync.RequestID)
	rm.requestQueue.TaskDone(p, task)

	ipr, ok := rm.inProgressRequestStatuses[requestID]
	if !ok {
		return
	}
	if _, ok := err.(hooks.ErrPaused); ok {
		ipr.state = paused
		return
	}
	log.Infow("graphsync request complete", "request id", requestID, "peer", ipr.p, "total time", time.Since(ipr.startTime))
	rm.terminateRequest(requestID, ipr)
}

func (rm *RequestManager) cancelRequest(requestID graphsync.RequestID, onTerminated chan<- error, terminalError error) {
	inProgressRequestStatus, ok := rm.inProgressRequestStatuses[requestID]
	if !ok {
		if onTerminated != nil {
			select {
			case onTerminated <- graphsync.RequestNotFoundErr{}:
			case <-rm.ctx.Done():
			}
		}
		return
	}

	if onTerminated != nil {
		inProgressRequestStatus.onTerminated = append(inProgressRequestStatus.onTerminated, onTerminated)
	}
	rm.SendRequest(inProgressRequestStatus.p, gsmsg.CancelRequest(requestID))
	rm.cancelOnError(requestID, inProgressRequestStatus, terminalError)
}

func (rm *RequestManager) cancelOnError(requestID graphsync.RequestID, ipr *inProgressRequestStatus, terminalError error) {
	if ipr.terminalError == nil {
		ipr.terminalError = terminalError
	}
	if ipr.state != running {
		rm.terminateRequest(requestID, ipr)
	} else {
		ipr.cancelFn()
	}
}

func (rm *RequestManager) processResponseMessage(p peer.ID, responses []gsmsg.GraphSyncResponse, blks []blocks.Block) {
	log.Debugf("beging rocessing message for peer %s", p)
	filteredResponses := rm.processExtensions(responses, p)
	filteredResponses = rm.filterResponsesForPeer(filteredResponses, p)
	rm.updateLastResponses(filteredResponses)
	responseMetadata := metadataForResponses(filteredResponses)
	rm.asyncLoader.ProcessResponse(responseMetadata, blks)
	rm.processTerminations(filteredResponses)
	log.Debugf("end processing message for peer %s", p)
}

func (rm *RequestManager) filterResponsesForPeer(responses []gsmsg.GraphSyncResponse, p peer.ID) []gsmsg.GraphSyncResponse {
	responsesForPeer := make([]gsmsg.GraphSyncResponse, 0, len(responses))
	for _, response := range responses {
		requestStatus, ok := rm.inProgressRequestStatuses[response.RequestID()]
		if !ok || requestStatus.p != p {
			continue
		}
		responsesForPeer = append(responsesForPeer, response)
	}
	return responsesForPeer
}

func (rm *RequestManager) processExtensions(responses []gsmsg.GraphSyncResponse, p peer.ID) []gsmsg.GraphSyncResponse {
	remainingResponses := make([]gsmsg.GraphSyncResponse, 0, len(responses))
	for _, response := range responses {
		success := rm.processExtensionsForResponse(p, response)
		if success {
			remainingResponses = append(remainingResponses, response)
		}
	}
	return remainingResponses
}

func (rm *RequestManager) updateLastResponses(responses []gsmsg.GraphSyncResponse) {
	for _, response := range responses {
		rm.inProgressRequestStatuses[response.RequestID()].lastResponse.Store(response)
	}
}

func (rm *RequestManager) processExtensionsForResponse(p peer.ID, response gsmsg.GraphSyncResponse) bool {
	result := rm.responseHooks.ProcessResponseHooks(p, response)
	if len(result.Extensions) > 0 {
		updateRequest := gsmsg.UpdateRequest(response.RequestID(), result.Extensions...)
		rm.SendRequest(p, updateRequest)
	}
	if result.Err != nil {
		requestStatus, ok := rm.inProgressRequestStatuses[response.RequestID()]
		if !ok {
			return false
		}
		rm.SendRequest(requestStatus.p, gsmsg.CancelRequest(response.RequestID()))
		rm.cancelOnError(response.RequestID(), requestStatus, result.Err)
		return false
	}
	return true
}

func (rm *RequestManager) processTerminations(responses []gsmsg.GraphSyncResponse) {
	for _, response := range responses {
		if response.Status().IsTerminal() {
			if response.Status().IsFailure() {
				rm.cancelOnError(response.RequestID(), rm.inProgressRequestStatuses[response.RequestID()], response.Status().AsError())
			}
			rm.asyncLoader.CompleteResponsesFor(response.RequestID())
		}
	}
}

func (rm *RequestManager) validateRequest(requestID graphsync.RequestID, p peer.ID, root ipld.Link, selectorSpec ipld.Node, extensions []graphsync.ExtensionData) (gsmsg.GraphSyncRequest, hooks.RequestResult, error) {
	_, err := ipldutil.EncodeNode(selectorSpec)
	if err != nil {
		return gsmsg.GraphSyncRequest{}, hooks.RequestResult{}, err
	}
	_, err = selector.ParseSelector(selectorSpec)
	if err != nil {
		return gsmsg.GraphSyncRequest{}, hooks.RequestResult{}, err
	}
	asCidLink, ok := root.(cidlink.Link)
	if !ok {
		return gsmsg.GraphSyncRequest{}, hooks.RequestResult{}, fmt.Errorf("request failed: link has no cid")
	}
	request := gsmsg.NewRequest(requestID, asCidLink.Cid, selectorSpec, defaultPriority, extensions...)
	hooksResult := rm.requestHooks.ProcessRequestHooks(p, request)
	if hooksResult.PersistenceOption != "" {
		dedupData, err := dedupkey.EncodeDedupKey(hooksResult.PersistenceOption)
		if err != nil {
			return gsmsg.GraphSyncRequest{}, hooks.RequestResult{}, err
		}
		request = request.ReplaceExtensions([]graphsync.ExtensionData{
			{
				Name: graphsync.ExtensionDeDupByKey,
				Data: dedupData,
			},
		})
	}
	err = rm.asyncLoader.StartRequest(requestID, hooksResult.PersistenceOption)
	if err != nil {
		return gsmsg.GraphSyncRequest{}, hooks.RequestResult{}, err
	}
	return request, hooksResult, nil
}

func (rm *RequestManager) unpause(id graphsync.RequestID, extensions []graphsync.ExtensionData) error {
	inProgressRequestStatus, ok := rm.inProgressRequestStatuses[id]
	if !ok {
		return graphsync.RequestNotFoundErr{}
	}
	if inProgressRequestStatus.state != paused {
		return errors.New("request is not paused")
	}
	inProgressRequestStatus.state = queued
	inProgressRequestStatus.request = inProgressRequestStatus.request.ReplaceExtensions(extensions)
	rm.requestQueue.PushTask(inProgressRequestStatus.p, peertask.Task{Topic: id, Priority: math.MaxInt32, Work: 1})
	return nil
}

func (rm *RequestManager) pause(id graphsync.RequestID) error {
	inProgressRequestStatus, ok := rm.inProgressRequestStatuses[id]
	if !ok {
		return graphsync.RequestNotFoundErr{}
	}
	if inProgressRequestStatus.state == paused {
		return errors.New("request is already paused")
	}
	select {
	case inProgressRequestStatus.pauseMessages <- struct{}{}:
	default:
	}
	return nil
}
