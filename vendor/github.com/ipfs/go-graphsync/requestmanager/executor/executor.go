package executor

import (
	"bytes"
	"context"
	"sync/atomic"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-peertaskqueue/peertask"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/traversal"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/cidset"
	"github.com/ipfs/go-graphsync/ipldutil"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/requestmanager/hooks"
	"github.com/ipfs/go-graphsync/requestmanager/types"
)

var log = logging.Logger("gs_request_executor")

// Manager is an interface the Executor uses to interact with the request manager
type Manager interface {
	SendRequest(peer.ID, gsmsg.GraphSyncRequest)
	GetRequestTask(peer.ID, *peertask.Task, chan RequestTask)
	ReleaseRequestTask(peer.ID, *peertask.Task, error)
}

// BlockHooks run for each block loaded
type BlockHooks interface {
	ProcessBlockHooks(p peer.ID, response graphsync.ResponseData, block graphsync.BlockData) hooks.UpdateResult
}

// AsyncLoadFn is a function which given a request id and an ipld.Link, returns
// a channel which will eventually return data for the link or an err
type AsyncLoadFn func(peer.ID, graphsync.RequestID, ipld.Link, ipld.LinkContext) <-chan types.AsyncLoadResult

// Executor handles actually executing graphsync requests and verifying them.
// It has control of requests when they are in the "running" state, while
// the manager is in charge when requests are queued or paused
type Executor struct {
	manager    Manager
	blockHooks BlockHooks
	loader     AsyncLoadFn
}

// NewExecutor returns a new executor
func NewExecutor(
	manager Manager,
	blockHooks BlockHooks,
	loader AsyncLoadFn) *Executor {
	return &Executor{
		manager:    manager,
		blockHooks: blockHooks,
		loader:     loader,
	}
}

func (e *Executor) ExecuteTask(ctx context.Context, pid peer.ID, task *peertask.Task) bool {
	requestTaskChan := make(chan RequestTask)
	var requestTask RequestTask
	e.manager.GetRequestTask(pid, task, requestTaskChan)
	select {
	case requestTask = <-requestTaskChan:
	case <-ctx.Done():
		return true
	}
	if requestTask.Empty {
		log.Info("Empty task on peer request stack")
		return false
	}
	log.Debugw("beginning request execution", "id", requestTask.Request.ID(), "peer", pid.String(), "root_cid", requestTask.Request.Root().String())
	err := e.traverse(requestTask)
	if err != nil && !ipldutil.IsContextCancelErr(err) {
		e.manager.SendRequest(requestTask.P, gsmsg.CancelRequest(requestTask.Request.ID()))
		if !isPausedErr(err) {
			select {
			case <-requestTask.Ctx.Done():
			case requestTask.InProgressErr <- err:
			}
		}
	}
	e.manager.ReleaseRequestTask(pid, task, err)
	log.Debugw("finishing response execution", "id", requestTask.Request.ID(), "peer", pid.String(), "root_cid", requestTask.Request.Root().String())
	return false
}

// RequestTask are parameters for a single request execution
type RequestTask struct {
	Ctx            context.Context
	Request        gsmsg.GraphSyncRequest
	LastResponse   *atomic.Value
	DoNotSendCids  *cid.Set
	PauseMessages  <-chan struct{}
	Traverser      ipldutil.Traverser
	P              peer.ID
	InProgressErr  chan error
	Empty          bool
	InitialRequest bool
}

func (e *Executor) traverse(rt RequestTask) error {
	onlyOnce := &onlyOnce{e, rt, false}
	// for initial request, start remote right away
	if rt.InitialRequest {
		if err := onlyOnce.startRemoteRequest(); err != nil {
			return err
		}
	}
	for {
		// check if traversal is complete
		isComplete, err := rt.Traverser.IsComplete()
		if isComplete {
			return err
		}
		// get current link request
		lnk, linkContext := rt.Traverser.CurrentRequest()
		// attempt to load
		log.Debugf("will load link=%s", lnk)
		resultChan := e.loader(rt.P, rt.Request.ID(), lnk, linkContext)
		var result types.AsyncLoadResult
		// check for immediate result
		select {
		case result = <-resultChan:
		default:
			// if no immediate result
			// initiate remote request if not already sent (we want to fill out the doNotSendCids on a resume)
			if err := onlyOnce.startRemoteRequest(); err != nil {
				return err
			}
			// wait for block result
			select {
			case <-rt.Ctx.Done():
				return ipldutil.ContextCancelError{}
			case result = <-resultChan:
			}
		}
		log.Debugf("successfully loaded link=%s, nBlocksRead=%d", lnk, rt.Traverser.NBlocksTraversed())
		// advance the traversal based on results
		err = e.advanceTraversal(rt, result)
		if err != nil {
			return err
		}

		// check for interrupts and run block hooks
		err = e.processResult(rt, lnk, result)
		if err != nil {
			return err
		}
	}
}

func (e *Executor) processBlockHooks(p peer.ID, response graphsync.ResponseData, block graphsync.BlockData) error {
	result := e.blockHooks.ProcessBlockHooks(p, response, block)
	if len(result.Extensions) > 0 {
		updateRequest := gsmsg.UpdateRequest(response.RequestID(), result.Extensions...)
		e.manager.SendRequest(p, updateRequest)
	}
	return result.Err
}

func (e *Executor) onNewBlock(rt RequestTask, block graphsync.BlockData) error {
	rt.DoNotSendCids.Add(block.Link().(cidlink.Link).Cid)
	response := rt.LastResponse.Load().(gsmsg.GraphSyncResponse)
	return e.processBlockHooks(rt.P, response, block)
}

func (e *Executor) advanceTraversal(rt RequestTask, result types.AsyncLoadResult) error {
	if result.Err != nil {
		// before processing result check for context cancel to avoid sending an additional error
		select {
		case <-rt.Ctx.Done():
			return ipldutil.ContextCancelError{}
		default:
		}
		select {
		case <-rt.Ctx.Done():
			return ipldutil.ContextCancelError{}
		case rt.InProgressErr <- result.Err:
			rt.Traverser.Error(traversal.SkipMe{})
			return nil
		}
	}
	return rt.Traverser.Advance(bytes.NewBuffer(result.Data))
}

func (e *Executor) processResult(rt RequestTask, link ipld.Link, result types.AsyncLoadResult) error {
	err := e.onNewBlock(rt, &blockData{link, result.Local, uint64(len(result.Data)), int64(rt.Traverser.NBlocksTraversed())})
	select {
	case <-rt.PauseMessages:
		if err == nil {
			err = hooks.ErrPaused{}
		}
	default:
	}
	return err
}

func (e *Executor) startRemoteRequest(rt RequestTask) error {
	request := rt.Request
	if rt.DoNotSendCids.Len() > 0 {
		cidsData, err := cidset.EncodeCidSet(rt.DoNotSendCids)
		if err != nil {
			return err
		}
		request = rt.Request.ReplaceExtensions([]graphsync.ExtensionData{{Name: graphsync.ExtensionDoNotSendCIDs, Data: cidsData}})
	}
	log.Debugw("starting remote request", "id", rt.Request.ID(), "peer", rt.P.String(), "root_cid", rt.Request.Root().String())
	e.manager.SendRequest(rt.P, request)
	return nil
}

func isPausedErr(err error) bool {
	_, isPaused := err.(hooks.ErrPaused)
	return isPaused
}

type onlyOnce struct {
	e           *Executor
	rt          RequestTask
	requestSent bool
}

func (so *onlyOnce) startRemoteRequest() error {
	if so.requestSent {
		return nil
	}
	so.requestSent = true
	return so.e.startRemoteRequest(so.rt)
}

type blockData struct {
	link  ipld.Link
	local bool
	size  uint64
	index int64
}

// Link is the link/cid for the block
func (bd *blockData) Link() ipld.Link {
	return bd.link
}

// BlockSize specifies the size of the block
func (bd *blockData) BlockSize() uint64 {
	return bd.size
}

// BlockSize specifies the amount of data actually transmitted over the network
func (bd *blockData) BlockSizeOnWire() uint64 {
	if bd.local {
		return 0
	}
	return bd.size
}

func (bd *blockData) Index() int64 {
	return bd.index
}
