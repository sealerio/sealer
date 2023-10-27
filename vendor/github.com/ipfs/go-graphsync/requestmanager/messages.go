package requestmanager

import (
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/ipld/go-ipld-prime"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/requestmanager/executor"
)

type pauseRequestMessage struct {
	id       graphsync.RequestID
	response chan<- error
}

func (prm *pauseRequestMessage) handle(rm *RequestManager) {
	err := rm.pause(prm.id)
	select {
	case <-rm.ctx.Done():
	case prm.response <- err:
	}
}

type unpauseRequestMessage struct {
	id         graphsync.RequestID
	extensions []graphsync.ExtensionData
	response   chan<- error
}

func (urm *unpauseRequestMessage) handle(rm *RequestManager) {
	err := rm.unpause(urm.id, urm.extensions)
	select {
	case <-rm.ctx.Done():
	case urm.response <- err:
	}
}

type processResponseMessage struct {
	p         peer.ID
	responses []gsmsg.GraphSyncResponse
	blks      []blocks.Block
}

func (prm *processResponseMessage) handle(rm *RequestManager) {
	rm.processResponseMessage(prm.p, prm.responses, prm.blks)
}

type cancelRequestMessage struct {
	requestID     graphsync.RequestID
	onTerminated  chan error
	terminalError error
}

func (crm *cancelRequestMessage) handle(rm *RequestManager) {
	rm.cancelRequest(crm.requestID, crm.onTerminated, crm.terminalError)
}

type getRequestTaskMessage struct {
	p                    peer.ID
	task                 *peertask.Task
	requestExecutionChan chan executor.RequestTask
}

func (irm *getRequestTaskMessage) handle(rm *RequestManager) {
	requestExecution := rm.getRequestTask(irm.p, irm.task)
	select {
	case <-rm.ctx.Done():
	case irm.requestExecutionChan <- requestExecution:
	}
}

type releaseRequestTaskMessage struct {
	p    peer.ID
	task *peertask.Task
	err  error
}

func (trm *releaseRequestTaskMessage) handle(rm *RequestManager) {
	rm.releaseRequestTask(trm.p, trm.task, trm.err)
}

type newRequestMessage struct {
	p                     peer.ID
	root                  ipld.Link
	selector              ipld.Node
	extensions            []graphsync.ExtensionData
	inProgressRequestChan chan<- inProgressRequest
}

func (nrm *newRequestMessage) handle(rm *RequestManager) {
	var ipr inProgressRequest

	ipr.request, ipr.incoming, ipr.incomingError = rm.newRequest(nrm.p, nrm.root, nrm.selector, nrm.extensions)
	ipr.requestID = ipr.request.ID()

	select {
	case nrm.inProgressRequestChan <- ipr:
	case <-rm.ctx.Done():
	}
}
