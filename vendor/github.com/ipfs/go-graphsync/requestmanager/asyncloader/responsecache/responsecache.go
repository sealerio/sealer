package responsecache

import (
	"sync"

	blocks "github.com/ipfs/go-block-format"
	logging "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"

	"github.com/ipfs/go-graphsync"
	"github.com/ipfs/go-graphsync/linktracker"
	"github.com/ipfs/go-graphsync/metadata"
)

var log = logging.Logger("graphsync")

// UnverifiedBlockStore is an interface for storing blocks
// as they come in and removing them as they are verified
type UnverifiedBlockStore interface {
	PruneBlocks(func(ipld.Link, uint64) bool)
	PruneBlock(ipld.Link)
	VerifyBlock(ipld.Link, ipld.LinkContext) ([]byte, error)
	AddUnverifiedBlock(ipld.Link, []byte)
}

// ResponseCache maintains a store of unverified blocks and response
// data about links for loading, and prunes blocks as needed.
type ResponseCache struct {
	responseCacheLk sync.RWMutex

	linkTracker          *linktracker.LinkTracker
	unverifiedBlockStore UnverifiedBlockStore
}

// New initializes a new ResponseCache using the given unverified block store.
func New(unverifiedBlockStore UnverifiedBlockStore) *ResponseCache {
	return &ResponseCache{
		linkTracker:          linktracker.New(),
		unverifiedBlockStore: unverifiedBlockStore,
	}
}

// FinishRequest indicate there is no more need to track blocks tied to this
// response. It returns the total number of bytes in blocks that were being
// tracked but are no longer in memory
func (rc *ResponseCache) FinishRequest(requestID graphsync.RequestID) {
	rc.responseCacheLk.Lock()
	rc.linkTracker.FinishRequest(requestID)

	rc.unverifiedBlockStore.PruneBlocks(func(link ipld.Link, amt uint64) bool {
		return rc.linkTracker.BlockRefCount(link) == 0
	})
	rc.responseCacheLk.Unlock()
}

// AttemptLoad attempts to laod the given block from the cache
func (rc *ResponseCache) AttemptLoad(requestID graphsync.RequestID, link ipld.Link, linkContext ipld.LinkContext) ([]byte, error) {
	rc.responseCacheLk.Lock()
	defer rc.responseCacheLk.Unlock()
	if rc.linkTracker.IsKnownMissingLink(requestID, link) {
		return nil, graphsync.RemoteMissingBlockErr{Link: link}
	}
	data, _ := rc.unverifiedBlockStore.VerifyBlock(link, linkContext)
	return data, nil
}

// ProcessResponse processes incoming response data, adding unverified blocks,
// and tracking link metadata from a remote peer
func (rc *ResponseCache) ProcessResponse(responses map[graphsync.RequestID]metadata.Metadata,
	blks []blocks.Block) {
	rc.responseCacheLk.Lock()

	for _, block := range blks {
		log.Debugf("Received block from network: %s", block.Cid().String())
		rc.unverifiedBlockStore.AddUnverifiedBlock(cidlink.Link{Cid: block.Cid()}, block.RawData())
	}

	for requestID, md := range responses {
		for _, item := range md {
			log.Debugf("Traverse link %s on request ID %d", item.Link.String(), requestID)
			rc.linkTracker.RecordLinkTraversal(requestID, cidlink.Link{Cid: item.Link}, item.BlockPresent)
		}
	}

	// prune unused blocks right away
	for _, block := range blks {
		if rc.linkTracker.BlockRefCount(cidlink.Link{Cid: block.Cid()}) == 0 {
			rc.unverifiedBlockStore.PruneBlock(cidlink.Link{Cid: block.Cid()})
		}
	}

	rc.responseCacheLk.Unlock()
}
