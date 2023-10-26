package blockpresencemanager

import (
	"sync"

	cid "github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

// BlockPresenceManager keeps track of which peers have indicated that they
// have or explicitly don't have a block
type BlockPresenceManager struct {
	sync.RWMutex
	presence map[cid.Cid]map[peer.ID]bool
}

func New() *BlockPresenceManager {
	return &BlockPresenceManager{
		presence: make(map[cid.Cid]map[peer.ID]bool),
	}
}

// ReceiveFrom is called when a peer sends us information about which blocks
// it has and does not have
func (bpm *BlockPresenceManager) ReceiveFrom(p peer.ID, haves []cid.Cid, dontHaves []cid.Cid) {
	bpm.Lock()
	defer bpm.Unlock()

	for _, c := range haves {
		bpm.updateBlockPresence(p, c, true)
	}
	for _, c := range dontHaves {
		bpm.updateBlockPresence(p, c, false)
	}
}

func (bpm *BlockPresenceManager) updateBlockPresence(p peer.ID, c cid.Cid, present bool) {
	_, ok := bpm.presence[c]
	if !ok {
		bpm.presence[c] = make(map[peer.ID]bool)
	}

	// Make sure not to change HAVE to DONT_HAVE
	has, pok := bpm.presence[c][p]
	if pok && has {
		return
	}
	bpm.presence[c][p] = present
}

// PeerHasBlock indicates whether the given peer has sent a HAVE for the given
// cid
func (bpm *BlockPresenceManager) PeerHasBlock(p peer.ID, c cid.Cid) bool {
	bpm.RLock()
	defer bpm.RUnlock()

	return bpm.presence[c][p]
}

// PeerDoesNotHaveBlock indicates whether the given peer has sent a DONT_HAVE
// for the given cid
func (bpm *BlockPresenceManager) PeerDoesNotHaveBlock(p peer.ID, c cid.Cid) bool {
	bpm.RLock()
	defer bpm.RUnlock()

	have, known := bpm.presence[c][p]
	return known && !have
}

// Filters the keys such that all the given peers have received a DONT_HAVE
// for a key.
// This allows us to know if we've exhausted all possibilities of finding
// the key with the peers we know about.
func (bpm *BlockPresenceManager) AllPeersDoNotHaveBlock(peers []peer.ID, ks []cid.Cid) []cid.Cid {
	bpm.RLock()
	defer bpm.RUnlock()

	var res []cid.Cid
	for _, c := range ks {
		if bpm.allDontHave(peers, c) {
			res = append(res, c)
		}
	}
	return res
}

func (bpm *BlockPresenceManager) allDontHave(peers []peer.ID, c cid.Cid) bool {
	// Check if we know anything about the cid's block presence
	ps, cok := bpm.presence[c]
	if !cok {
		return false
	}

	// Check if we explicitly know that all the given peers do not have the cid
	for _, p := range peers {
		if has, pok := ps[p]; !pok || has {
			return false
		}
	}
	return true
}

// RemoveKeys cleans up the given keys from the block presence map
func (bpm *BlockPresenceManager) RemoveKeys(ks []cid.Cid) {
	bpm.Lock()
	defer bpm.Unlock()

	for _, c := range ks {
		delete(bpm.presence, c)
	}
}

// HasKey indicates whether the BlockPresenceManager is tracking the given key
// (used by the tests)
func (bpm *BlockPresenceManager) HasKey(c cid.Cid) bool {
	bpm.Lock()
	defer bpm.Unlock()

	_, ok := bpm.presence[c]
	return ok
}
