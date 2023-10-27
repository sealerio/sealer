package responsemanager

import (
	"context"
	"errors"
	"math"

	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/ipldutil"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/notifications"
	"github.com/ipfs/go-graphsync/responsemanager/hooks"
	"github.com/ipfs/go-graphsync/responsemanager/queryexecutor"
	"github.com/ipfs/go-graphsync/responsemanager/responseassembler"
)

// The code in this file implements the internal thread for the response manager.
// These functions can modify the internal state of the ResponseManager

func (rm *ResponseManager) cleanupInProcessResponses() {
	for _, response := range rm.inProgressResponses {
		response.cancelFn()
	}
}

func (rm *ResponseManager) run() {
	defer rm.cleanupInProcessResponses()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case message := <-rm.messages:
			message.handle(rm)
		}
	}
}

func (rm *ResponseManager) processUpdate(key responseKey, update gsmsg.GraphSyncRequest) {
	response, ok := rm.inProgressResponses[key]
	if !ok {
		log.Warnf("received update for non existent request, peer %s, request ID %d", key.p.Pretty(), key.requestID)
		return
	}
	if response.state != paused {
		response.updates = append(response.updates, update)
		select {
		case response.signals.UpdateSignal <- struct{}{}:
		default:
		}
		return
	} // else this is a paused response, so the update needs to be handled here and not in the executor
	result := rm.updateHooks.ProcessUpdateHooks(key.p, response.request, update)
	_ = rm.responseAssembler.Transaction(key.p, key.requestID, func(rb responseassembler.ResponseBuilder) error {
		for _, extension := range result.Extensions {
			rb.SendExtensionData(extension)
		}
		if result.Err != nil {
			rb.FinishWithError(graphsync.RequestFailedUnknown)
			rb.AddNotifee(notifications.Notifee{Data: graphsync.RequestFailedUnknown, Subscriber: response.subscriber})
		}
		return nil
	})
	if result.Err != nil {
		delete(rm.inProgressResponses, key)
		response.cancelFn()
		return
	}
	if result.Unpause {
		err := rm.unpauseRequest(key.p, key.requestID)
		if err != nil {
			log.Warnf("error unpausing request: %s", err.Error())
		}
	}
}

func (rm *ResponseManager) unpauseRequest(p peer.ID, requestID graphsync.RequestID, extensions ...graphsync.ExtensionData) error {
	key := responseKey{p, requestID}
	inProgressResponse, ok := rm.inProgressResponses[key]
	if !ok {
		return errors.New("could not find request")
	}
	if inProgressResponse.state != paused {
		return errors.New("request is not paused")
	}
	inProgressResponse.state = queued
	if len(extensions) > 0 {
		_ = rm.responseAssembler.Transaction(p, requestID, func(rb responseassembler.ResponseBuilder) error {
			for _, extension := range extensions {
				rb.SendExtensionData(extension)
			}
			return nil
		})
	}
	rm.responseQueue.PushTask(p, peertask.Task{Topic: key, Priority: math.MaxInt32, Work: 1})
	return nil
}

func (rm *ResponseManager) abortRequest(p peer.ID, requestID graphsync.RequestID, err error) error {
	key := responseKey{p, requestID}
	rm.responseQueue.Remove(key, key.p)
	response, ok := rm.inProgressResponses[key]
	if !ok {
		return errors.New("could not find request")
	}

	if response.state != running {
		_ = rm.responseAssembler.Transaction(p, requestID, func(rb responseassembler.ResponseBuilder) error {
			if ipldutil.IsContextCancelErr(err) {
				rm.connManager.Unprotect(p, requestID.Tag())
				rm.cancelledListeners.NotifyCancelledListeners(p, response.request)
				rb.ClearRequest()
			} else if err == queryexecutor.ErrNetworkError {
				rb.ClearRequest()
			} else {
				rb.FinishWithError(graphsync.RequestCancelled)
				rb.AddNotifee(notifications.Notifee{Data: graphsync.RequestCancelled, Subscriber: response.subscriber})
			}
			return nil
		})
		delete(rm.inProgressResponses, key)
		response.cancelFn()
		return nil
	}
	select {
	case response.signals.ErrSignal <- err:
	default:
	}
	return nil
}

func (rm *ResponseManager) processRequests(p peer.ID, requests []gsmsg.GraphSyncRequest) {
	for _, request := range requests {
		key := responseKey{p: p, requestID: request.ID()}
		if request.IsCancel() {
			_ = rm.abortRequest(p, request.ID(), ipldutil.ContextCancelError{})
			continue
		}
		if request.IsUpdate() {
			rm.processUpdate(key, request)
			continue
		}
		rm.connManager.Protect(p, request.ID().Tag())
		rm.requestQueuedHooks.ProcessRequestQueuedHooks(p, request)
		ctx, cancelFn := context.WithCancel(rm.ctx)
		sub := notifications.NewTopicDataSubscriber(&subscriber{
			p:                     key.p,
			request:               request,
			requestCloser:         rm,
			blockSentListeners:    rm.blockSentListeners,
			completedListeners:    rm.completedListeners,
			networkErrorListeners: rm.networkErrorListeners,
			connManager:           rm.connManager,
		})

		rm.inProgressResponses[key] =
			&inProgressResponseStatus{
				ctx:        ctx,
				cancelFn:   cancelFn,
				subscriber: sub,
				request:    request,
				signals: queryexecutor.ResponseSignals{
					PauseSignal:  make(chan struct{}, 1),
					UpdateSignal: make(chan struct{}, 1),
					ErrSignal:    make(chan error, 1),
				},
				state: queued,
			}
		// TODO: Use a better work estimation metric.

		rm.responseQueue.PushTask(p, peertask.Task{Topic: key, Priority: int(request.Priority()), Work: 1})
	}
}

func (rm *ResponseManager) taskDataForKey(key responseKey) queryexecutor.ResponseTask {
	response, hasResponse := rm.inProgressResponses[key]
	if !hasResponse {
		return queryexecutor.ResponseTask{Empty: true}
	}
	if response.loader == nil || response.traverser == nil {
		loader, traverser, isPaused, err := (&queryPreparer{rm.requestHooks, rm.responseAssembler, rm.linkSystem, rm.maxLinksPerRequest}).prepareQuery(response.ctx, key.p, response.request, response.signals, response.subscriber)
		if err != nil {
			response.cancelFn()
			delete(rm.inProgressResponses, key)
			return queryexecutor.ResponseTask{Empty: true}
		}
		response.loader = loader
		response.traverser = traverser
		if isPaused {
			response.state = paused
			return queryexecutor.ResponseTask{Empty: true}
		}
	}
	response.state = running
	return queryexecutor.ResponseTask{
		Ctx:        response.ctx,
		Empty:      false,
		Subscriber: response.subscriber,
		Request:    response.request,
		Loader:     response.loader,
		Traverser:  response.traverser,
		Signals:    response.signals,
	}
}

func (rm *ResponseManager) startTask(task *peertask.Task) queryexecutor.ResponseTask {
	key := task.Topic.(responseKey)
	taskData := rm.taskDataForKey(key)
	if taskData.Empty {
		rm.responseQueue.TaskDone(key.p, task)
	}
	return taskData
}

func (rm *ResponseManager) finishTask(task *peertask.Task, err error) {
	key := task.Topic.(responseKey)
	rm.responseQueue.TaskDone(key.p, task)
	response, ok := rm.inProgressResponses[key]
	if !ok {
		return
	}
	if _, ok := err.(hooks.ErrPaused); ok {
		response.state = paused
		return
	}
	if err != nil {
		log.Infof("response failed: %w", err)
	}
	delete(rm.inProgressResponses, key)
	response.cancelFn()
}

func (rm *ResponseManager) getUpdates(key responseKey) []gsmsg.GraphSyncRequest {
	response, ok := rm.inProgressResponses[key]
	if !ok {
		return nil
	}
	updates := response.updates
	response.updates = nil
	return updates
}

func (rm *ResponseManager) pauseRequest(p peer.ID, requestID graphsync.RequestID) error {
	key := responseKey{p, requestID}
	inProgressResponse, ok := rm.inProgressResponses[key]
	if !ok {
		return errors.New("could not find request")
	}
	if inProgressResponse.state == paused {
		return errors.New("request is already paused")
	}
	select {
	case inProgressResponse.signals.PauseSignal <- struct{}{}:
	default:
	}
	return nil
}
