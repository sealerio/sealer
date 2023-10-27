package linktracker

import (
	"github.com/ipld/go-ipld-prime"

	"github.com/ipfs/go-graphsync"
)

// LinkTracker records links being traversed to determine useful information
// in crafting responses for a peer. Specifically, if any in progress request
// has already sent a block for a given link, don't send it again.
// Second, keep track of whether links are missing blocks so you can determine
// at the end if a complete response has been transmitted.
type LinkTracker struct {
	missingBlocks                     map[graphsync.RequestID]map[ipld.Link]struct{}
	linksWithBlocksTraversedByRequest map[graphsync.RequestID][]ipld.Link
	traversalsWithBlocksInProgress    map[ipld.Link]int
}

// New makes a new link tracker
func New() *LinkTracker {
	return &LinkTracker{
		missingBlocks:                     make(map[graphsync.RequestID]map[ipld.Link]struct{}),
		linksWithBlocksTraversedByRequest: make(map[graphsync.RequestID][]ipld.Link),
		traversalsWithBlocksInProgress:    make(map[ipld.Link]int),
	}
}

// BlockRefCount returns the number of times a present block has been traversed
// by in progress requests
func (lt *LinkTracker) BlockRefCount(link ipld.Link) int {
	return lt.traversalsWithBlocksInProgress[link]
}

// IsKnownMissingLink returns whether the given request recorded the given link as missing
func (lt *LinkTracker) IsKnownMissingLink(requestID graphsync.RequestID, link ipld.Link) bool {
	missingBlocks, ok := lt.missingBlocks[requestID]
	if !ok {
		return false
	}
	_, ok = missingBlocks[link]
	return ok
}

// RecordLinkTraversal records that we traversed a link during a request, and
// whether we had the block when we did it.
func (lt *LinkTracker) RecordLinkTraversal(requestID graphsync.RequestID, link ipld.Link, hasBlock bool) {
	if hasBlock {
		lt.linksWithBlocksTraversedByRequest[requestID] = append(lt.linksWithBlocksTraversedByRequest[requestID], link)
		lt.traversalsWithBlocksInProgress[link]++
	} else {
		missingBlocks, ok := lt.missingBlocks[requestID]
		if !ok {
			missingBlocks = make(map[ipld.Link]struct{})
			lt.missingBlocks[requestID] = missingBlocks
		}
		missingBlocks[link] = struct{}{}
	}
}

// FinishRequest records that we have completed the given request, and returns
// true if all links traversed had blocks present.
func (lt *LinkTracker) FinishRequest(requestID graphsync.RequestID) (hasAllBlocks bool) {
	_, ok := lt.missingBlocks[requestID]
	hasAllBlocks = !ok
	delete(lt.missingBlocks, requestID)
	links, ok := lt.linksWithBlocksTraversedByRequest[requestID]
	if !ok {
		return
	}
	for _, link := range links {
		lt.traversalsWithBlocksInProgress[link]--
		if lt.traversalsWithBlocksInProgress[link] <= 0 {
			delete(lt.traversalsWithBlocksInProgress, link)
		}
	}
	delete(lt.linksWithBlocksTraversedByRequest, requestID)

	return
}

// Empty returns true if the link tracker is empty
func (lt *LinkTracker) Empty() bool {
	return len(lt.missingBlocks) == 0 && len(lt.traversalsWithBlocksInProgress) == 0
}
