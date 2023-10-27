package unverifiedblockstore

import (
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	ipld "github.com/ipld/go-ipld-prime"
)

var log = logging.Logger("gs-unverifiedbs")

type settableWriter interface {
	SetBytes([]byte) error
}

// UnverifiedBlockStore holds an in memory cache of receied blocks from the network
// that have not been verified to be part of a traversal
type UnverifiedBlockStore struct {
	inMemoryBlocks map[ipld.Link][]byte
	storer         ipld.BlockWriteOpener
	dataSize       uint64
}

// New initializes a new unverified store with the given storer function for writing
// to permaneant storage if the block is verified
func New(storer ipld.BlockWriteOpener) *UnverifiedBlockStore {
	return &UnverifiedBlockStore{
		inMemoryBlocks: make(map[ipld.Link][]byte),
		storer:         storer,
	}
}

// AddUnverifiedBlock adds a new unverified block to the in memory cache as it
// comes in as part of a traversal.
func (ubs *UnverifiedBlockStore) AddUnverifiedBlock(lnk ipld.Link, data []byte) {
	ubs.inMemoryBlocks[lnk] = data
	ubs.dataSize = ubs.dataSize + uint64(len(data))
	log.Debugw("added in-memory block", "total_queued_bytes", ubs.dataSize)
}

// PruneBlocks removes blocks from the unverified store without committing them,
// if the passed in function returns true for the given link
func (ubs *UnverifiedBlockStore) PruneBlocks(shouldPrune func(ipld.Link, uint64) bool) {
	for link, data := range ubs.inMemoryBlocks {
		if shouldPrune(link, uint64(len(data))) {
			delete(ubs.inMemoryBlocks, link)
			ubs.dataSize = ubs.dataSize - uint64(len(data))
		}
	}
	log.Debugw("finished pruning in-memory blocks", "total_queued_bytes", ubs.dataSize)
}

// PruneBlock deletes an individual block from the store
func (ubs *UnverifiedBlockStore) PruneBlock(link ipld.Link) {
	delete(ubs.inMemoryBlocks, link)
	ubs.dataSize = ubs.dataSize - uint64(len(ubs.inMemoryBlocks[link]))
	log.Debugw("pruned in-memory block", "total_queued_bytes", ubs.dataSize)
}

// VerifyBlock verifies the data for the given link as being part of a traversal,
// removes it from the unverified store, and writes it to permaneant storage.
func (ubs *UnverifiedBlockStore) VerifyBlock(lnk ipld.Link, linkContext ipld.LinkContext) ([]byte, error) {
	data, ok := ubs.inMemoryBlocks[lnk]
	if !ok {
		return nil, fmt.Errorf("block not found")
	}
	delete(ubs.inMemoryBlocks, lnk)
	ubs.dataSize = ubs.dataSize - uint64(len(data))
	log.Debugw("verified block", "total_queued_bytes", ubs.dataSize)

	buffer, committer, err := ubs.storer(linkContext)
	if err != nil {
		return nil, err
	}
	if settable, ok := buffer.(settableWriter); ok {
		err = settable.SetBytes(data)
	} else {
		_, err = buffer.Write(data)
	}
	if err != nil {
		return nil, err
	}
	err = committer(lnk)
	if err != nil {
		return nil, err
	}
	return data, nil
}
