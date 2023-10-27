package asyncloader

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipld/go-ipld-prime"
	peer "github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/metadata"
	"github.com/ipfs/go-graphsync/requestmanager/asyncloader/loadattemptqueue"
	"github.com/ipfs/go-graphsync/requestmanager/asyncloader/responsecache"
	"github.com/ipfs/go-graphsync/requestmanager/asyncloader/unverifiedblockstore"
	"github.com/ipfs/go-graphsync/requestmanager/types"
)

type alternateQueue struct {
	responseCache    *responsecache.ResponseCache
	loadAttemptQueue *loadattemptqueue.LoadAttemptQueue
}

// AsyncLoader manages loading links asynchronously in as new responses
// come in from the network
type AsyncLoader struct {
	ctx    context.Context
	cancel context.CancelFunc

	// this mutex protects access to the state of the async loader, which covers all data fields below below
	stateLk          sync.Mutex
	activeRequests   map[graphsync.RequestID]struct{}
	requestQueues    map[graphsync.RequestID]string
	alternateQueues  map[string]alternateQueue
	responseCache    *responsecache.ResponseCache
	loadAttemptQueue *loadattemptqueue.LoadAttemptQueue
}

// New initializes a new link loading manager for asynchronous loads from the given context
// and local store loading and storing function
func New(ctx context.Context, linkSystem ipld.LinkSystem) *AsyncLoader {
	responseCache, loadAttemptQueue := setupAttemptQueue(linkSystem)
	ctx, cancel := context.WithCancel(ctx)
	return &AsyncLoader{
		ctx:              ctx,
		cancel:           cancel,
		activeRequests:   make(map[graphsync.RequestID]struct{}),
		requestQueues:    make(map[graphsync.RequestID]string),
		alternateQueues:  make(map[string]alternateQueue),
		responseCache:    responseCache,
		loadAttemptQueue: loadAttemptQueue,
	}
}

// RegisterPersistenceOption registers a new loader/storer option for processing requests
func (al *AsyncLoader) RegisterPersistenceOption(name string, lsys ipld.LinkSystem) error {
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	_, existing := al.alternateQueues[name]
	if existing {
		return errors.New("already registerd a persistence option with this name")
	}
	responseCache, loadAttemptQueue := setupAttemptQueue(lsys)
	al.alternateQueues[name] = alternateQueue{responseCache, loadAttemptQueue}
	return nil
}

// UnregisterPersistenceOption unregisters an existing loader/storer option for processing requests
func (al *AsyncLoader) UnregisterPersistenceOption(name string) error {
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	_, ok := al.alternateQueues[name]
	if !ok {
		return fmt.Errorf("unknown persistence option: %s", name)
	}
	for _, requestQueue := range al.requestQueues {
		if name == requestQueue {
			return errors.New("cannot unregister while requests are in progress")
		}
	}
	delete(al.alternateQueues, name)
	return nil
}

// StartRequest indicates the given request has started and the manager should
// continually attempt to load links for this request as new responses come in
func (al *AsyncLoader) StartRequest(requestID graphsync.RequestID, persistenceOption string) error {
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	if persistenceOption != "" {
		_, ok := al.alternateQueues[persistenceOption]
		if !ok {
			return errors.New("unknown persistence option")
		}
		al.requestQueues[requestID] = persistenceOption
	}
	al.activeRequests[requestID] = struct{}{}
	return nil
}

// ProcessResponse injests new responses and completes asynchronous loads as
// neccesary
func (al *AsyncLoader) ProcessResponse(responses map[graphsync.RequestID]metadata.Metadata,
	blks []blocks.Block) {
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	byQueue := make(map[string][]graphsync.RequestID)
	for requestID := range responses {
		queue := al.requestQueues[requestID]
		byQueue[queue] = append(byQueue[queue], requestID)
	}
	for queue, requestIDs := range byQueue {
		loadAttemptQueue := al.getLoadAttemptQueue(queue)
		responseCache := al.getResponseCache(queue)
		queueResponses := make(map[graphsync.RequestID]metadata.Metadata, len(requestIDs))
		for _, requestID := range requestIDs {
			queueResponses[requestID] = responses[requestID]
		}
		responseCache.ProcessResponse(queueResponses, blks)
		loadAttemptQueue.RetryLoads()
	}
}

// AsyncLoad asynchronously loads the given link for the given request ID. It returns a channel for data and a channel
// for errors -- only one message will be sent over either.
func (al *AsyncLoader) AsyncLoad(p peer.ID, requestID graphsync.RequestID, link ipld.Link, linkContext ipld.LinkContext) <-chan types.AsyncLoadResult {
	resultChan := make(chan types.AsyncLoadResult, 1)
	lr := loadattemptqueue.NewLoadRequest(p, requestID, link, linkContext, resultChan)
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	_, retry := al.activeRequests[requestID]
	loadAttemptQueue := al.getLoadAttemptQueue(al.requestQueues[requestID])
	loadAttemptQueue.AttemptLoad(lr, retry)
	return resultChan
}

// CompleteResponsesFor indicates no further responses will come in for the given
// requestID, so if no responses are in the cache or local store, a link load
// should not retry
func (al *AsyncLoader) CompleteResponsesFor(requestID graphsync.RequestID) {
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	delete(al.activeRequests, requestID)
	loadAttemptQueue := al.getLoadAttemptQueue(al.requestQueues[requestID])
	loadAttemptQueue.ClearRequest(requestID)
}

// CleanupRequest indicates the given request is complete on the client side,
// and no further attempts will be made to load links for this request,
// so any cached response data is invalid can be cleaned
func (al *AsyncLoader) CleanupRequest(p peer.ID, requestID graphsync.RequestID) {
	al.stateLk.Lock()
	defer al.stateLk.Unlock()
	responseCache := al.responseCache
	aq, ok := al.requestQueues[requestID]
	if ok {
		responseCache = al.alternateQueues[aq].responseCache
		delete(al.requestQueues, requestID)
	}
	responseCache.FinishRequest(requestID)
}

func (al *AsyncLoader) getLoadAttemptQueue(queue string) *loadattemptqueue.LoadAttemptQueue {
	if queue == "" {
		return al.loadAttemptQueue
	}
	return al.alternateQueues[queue].loadAttemptQueue
}

func (al *AsyncLoader) getResponseCache(queue string) *responsecache.ResponseCache {
	if queue == "" {
		return al.responseCache
	}
	return al.alternateQueues[queue].responseCache
}

func setupAttemptQueue(lsys ipld.LinkSystem) (*responsecache.ResponseCache, *loadattemptqueue.LoadAttemptQueue) {

	unverifiedBlockStore := unverifiedblockstore.New(lsys.StorageWriteOpener)
	responseCache := responsecache.New(unverifiedBlockStore)
	loadAttemptQueue := loadattemptqueue.New(func(p peer.ID, requestID graphsync.RequestID, link ipld.Link, linkContext ipld.LinkContext) types.AsyncLoadResult {
		// load from response cache
		data, err := responseCache.AttemptLoad(requestID, link, linkContext)
		if err != nil {
			return types.AsyncLoadResult{Err: err, Local: false}
		}
		if data != nil {
			return types.AsyncLoadResult{Data: data, Local: false}
		}
		// fall back to local store
		if stream, err := lsys.StorageReadOpener(linkContext, link); stream != nil && err == nil {
			if localData, err := ioutil.ReadAll(stream); err == nil && localData != nil {
				return types.AsyncLoadResult{Data: localData, Local: true}
			}
		}
		return types.AsyncLoadResult{Local: false}
	})

	return responseCache, loadAttemptQueue
}
