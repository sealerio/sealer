package responseassembler

import (
	"sync"

	"github.com/ipld/go-ipld-prime"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/linktracker"
)

type peerLinkTracker struct {
	linkTrackerLk   sync.RWMutex
	linkTracker     *linktracker.LinkTracker
	altTrackers     map[string]*linktracker.LinkTracker
	dedupKeys       map[graphsync.RequestID]string
	blockSentCount  map[graphsync.RequestID]int64
	skipFirstBlocks map[graphsync.RequestID]int64
}

func newTracker() *peerLinkTracker {
	return &peerLinkTracker{
		linkTracker:     linktracker.New(),
		dedupKeys:       make(map[graphsync.RequestID]string),
		altTrackers:     make(map[string]*linktracker.LinkTracker),
		blockSentCount:  make(map[graphsync.RequestID]int64),
		skipFirstBlocks: make(map[graphsync.RequestID]int64),
	}
}

func (prs *peerLinkTracker) getLinkTracker(requestID graphsync.RequestID) *linktracker.LinkTracker {
	key, ok := prs.dedupKeys[requestID]
	if ok {
		return prs.altTrackers[key]
	}
	return prs.linkTracker
}

// DedupKey indicates that outgoing blocks should be deduplicated in a seperate bucket (only with requests that share
// supplied key string)
func (prs *peerLinkTracker) DedupKey(requestID graphsync.RequestID, key string) {
	prs.linkTrackerLk.Lock()
	defer prs.linkTrackerLk.Unlock()
	prs.dedupKeys[requestID] = key
	_, ok := prs.altTrackers[key]
	if !ok {
		prs.altTrackers[key] = linktracker.New()
	}
}

// IgnoreBlocks indicates that a list of keys should be ignored when sending blocks
func (prs *peerLinkTracker) IgnoreBlocks(requestID graphsync.RequestID, links []ipld.Link) {
	prs.linkTrackerLk.Lock()
	linkTracker := prs.getLinkTracker(requestID)
	for _, link := range links {
		linkTracker.RecordLinkTraversal(requestID, link, true)
	}
	prs.linkTrackerLk.Unlock()
}

func (prs *peerLinkTracker) SkipFirstBlocks(requestID graphsync.RequestID, blocksToSkip int64) {
	prs.linkTrackerLk.Lock()
	prs.skipFirstBlocks[requestID] = blocksToSkip
	prs.linkTrackerLk.Unlock()
}

// FinishTracking clears link tracking data for the request.
func (prs *peerLinkTracker) FinishTracking(requestID graphsync.RequestID) bool {
	prs.linkTrackerLk.Lock()
	defer prs.linkTrackerLk.Unlock()
	linkTracker := prs.getLinkTracker(requestID)
	allBlocks := linkTracker.FinishRequest(requestID)
	key, ok := prs.dedupKeys[requestID]
	if ok {
		delete(prs.dedupKeys, requestID)
		var otherRequestsFound bool
		for _, otherKey := range prs.dedupKeys {
			if otherKey == key {
				otherRequestsFound = true
				break
			}
		}
		if !otherRequestsFound {
			delete(prs.altTrackers, key)
		}
	}
	delete(prs.blockSentCount, requestID)
	delete(prs.skipFirstBlocks, requestID)
	return allBlocks
}

// RecordLinkTraversal records whether a link is found for a request.
func (prs *peerLinkTracker) RecordLinkTraversal(requestID graphsync.RequestID,
	link ipld.Link, hasBlock bool) (bool, int64) {
	prs.linkTrackerLk.Lock()
	defer prs.linkTrackerLk.Unlock()
	prs.blockSentCount[requestID]++
	notSkipped := prs.skipFirstBlocks[requestID] < prs.blockSentCount[requestID]
	linkTracker := prs.getLinkTracker(requestID)
	isUnique := linkTracker.BlockRefCount(link) == 0
	linkTracker.RecordLinkTraversal(requestID, link, hasBlock)
	return hasBlock && notSkipped && isUnique, prs.blockSentCount[requestID]
}
