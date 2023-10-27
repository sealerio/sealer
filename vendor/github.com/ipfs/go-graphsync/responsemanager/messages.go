package responsemanager

import (
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/responsemanager/queryexecutor"
)

type processRequestMessage struct {
	p        peer.ID
	requests []gsmsg.GraphSyncRequest
}

type pauseRequestMessage struct {
	p         peer.ID
	requestID graphsync.RequestID
	response  chan error
}

func (prm *pauseRequestMessage) handle(rm *ResponseManager) {
	err := rm.pauseRequest(prm.p, prm.requestID)
	select {
	case <-rm.ctx.Done():
	case prm.response <- err:
	}
}

type errorRequestMessage struct {
	p         peer.ID
	requestID graphsync.RequestID
	err       error
	response  chan error
}

func (erm *errorRequestMessage) handle(rm *ResponseManager) {
	err := rm.abortRequest(erm.p, erm.requestID, erm.err)
	select {
	case <-rm.ctx.Done():
	case erm.response <- err:
	}
}

type synchronizeMessage struct {
	sync chan error
}

func (sm *synchronizeMessage) handle(rm *ResponseManager) {
	select {
	case <-rm.ctx.Done():
	case sm.sync <- nil:
	}
}

type unpauseRequestMessage struct {
	p          peer.ID
	requestID  graphsync.RequestID
	response   chan error
	extensions []graphsync.ExtensionData
}

func (urm *unpauseRequestMessage) handle(rm *ResponseManager) {
	err := rm.unpauseRequest(urm.p, urm.requestID, urm.extensions...)
	select {
	case <-rm.ctx.Done():
	case urm.response <- err:
	}
}

type responseUpdateRequest struct {
	key        responseKey
	updateChan chan<- []gsmsg.GraphSyncRequest
}

func (rur *responseUpdateRequest) handle(rm *ResponseManager) {
	updates := rm.getUpdates(rur.key)
	select {
	case <-rm.ctx.Done():
	case rur.updateChan <- updates:
	}
}

type finishTaskRequest struct {
	task *peertask.Task
	err  error
}

func (ftr *finishTaskRequest) handle(rm *ResponseManager) {
	rm.finishTask(ftr.task, ftr.err)
}

type startTaskRequest struct {
	task         *peertask.Task
	taskDataChan chan<- queryexecutor.ResponseTask
}

func (str *startTaskRequest) handle(rm *ResponseManager) {
	taskData := rm.startTask(str.task)

	select {
	case <-rm.ctx.Done():
	case str.taskDataChan <- taskData:
	}
}

func (prm *processRequestMessage) handle(rm *ResponseManager) {
	rm.processRequests(prm.p, prm.requests)
}
