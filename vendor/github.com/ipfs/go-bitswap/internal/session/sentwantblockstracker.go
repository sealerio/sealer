package session

import (
	cid "github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// sentWantBlocksTracker keeps track of which peers we've sent a want-block to
type sentWantBlocksTracker struct {
	sentWantBlocks map[peer.ID]map[cid.Cid]struct{}
}

func newSentWantBlocksTracker() *sentWantBlocksTracker {
	return &sentWantBlocksTracker{
		sentWantBlocks: make(map[peer.ID]map[cid.Cid]struct{}),
	}
}

func (s *sentWantBlocksTracker) addSentWantBlocksTo(p peer.ID, ks []cid.Cid) {
	cids, ok := s.sentWantBlocks[p]
	if !ok {
		cids = make(map[cid.Cid]struct{}, len(ks))
		s.sentWantBlocks[p] = cids
	}
	for _, c := range ks {
		cids[c] = struct{}{}
	}
}

func (s *sentWantBlocksTracker) haveSentWantBlockTo(p peer.ID, c cid.Cid) bool {
	_, ok := s.sentWantBlocks[p][c]
	return ok
}
