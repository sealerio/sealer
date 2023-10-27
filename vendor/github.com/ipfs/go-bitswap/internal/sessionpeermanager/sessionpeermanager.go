package sessionpeermanager

import (
	"fmt"
	"sync"

	logging "github.com/ipfs/go-log"

	peer "github.com/libp2p/go-libp2p-core/peer"
)

var log = logging.Logger("bs:sprmgr")

const (
	// Connection Manager tag value for session peers. Indicates to connection
	// manager that it should keep the connection to the peer.
	sessionPeerTagValue = 5
)

// PeerTagger is an interface for tagging peers with metadata
type PeerTagger interface {
	TagPeer(peer.ID, string, int)
	UntagPeer(p peer.ID, tag string)
	Protect(peer.ID, string)
	Unprotect(peer.ID, string) bool
}

// SessionPeerManager keeps track of peers for a session, and takes care of
// ConnectionManager tagging.
type SessionPeerManager struct {
	tagger PeerTagger
	tag    string

	id              uint64
	plk             sync.RWMutex
	peers           map[peer.ID]struct{}
	peersDiscovered bool
}

// New creates a new SessionPeerManager
func New(id uint64, tagger PeerTagger) *SessionPeerManager {
	return &SessionPeerManager{
		id:     id,
		tag:    fmt.Sprint("bs-ses-", id),
		tagger: tagger,
		peers:  make(map[peer.ID]struct{}),
	}
}

// AddPeer adds the peer to the SessionPeerManager.
// Returns true if the peer is a new peer, false if it already existed.
func (spm *SessionPeerManager) AddPeer(p peer.ID) bool {
	spm.plk.Lock()
	defer spm.plk.Unlock()

	// Check if the peer is a new peer
	if _, ok := spm.peers[p]; ok {
		return false
	}

	spm.peers[p] = struct{}{}
	spm.peersDiscovered = true

	// Tag the peer with the ConnectionManager so it doesn't discard the
	// connection
	spm.tagger.TagPeer(p, spm.tag, sessionPeerTagValue)

	log.Debugw("Bitswap: Added peer to session", "session", spm.id, "peer", p, "peerCount", len(spm.peers))
	return true
}

// Protect connection to this peer from being pruned by the connection manager
func (spm *SessionPeerManager) ProtectConnection(p peer.ID) {
	spm.plk.Lock()
	defer spm.plk.Unlock()

	if _, ok := spm.peers[p]; !ok {
		return
	}

	spm.tagger.Protect(p, spm.tag)
}

// RemovePeer removes the peer from the SessionPeerManager.
// Returns true if the peer was removed, false if it did not exist.
func (spm *SessionPeerManager) RemovePeer(p peer.ID) bool {
	spm.plk.Lock()
	defer spm.plk.Unlock()

	if _, ok := spm.peers[p]; !ok {
		return false
	}

	delete(spm.peers, p)
	spm.tagger.UntagPeer(p, spm.tag)
	spm.tagger.Unprotect(p, spm.tag)

	log.Debugw("Bitswap: removed peer from session", "session", spm.id, "peer", p, "peerCount", len(spm.peers))
	return true
}

// PeersDiscovered indicates whether peers have been discovered yet.
// Returns true once a peer has been discovered by the session (even if all
// peers are later removed from the session).
func (spm *SessionPeerManager) PeersDiscovered() bool {
	spm.plk.RLock()
	defer spm.plk.RUnlock()

	return spm.peersDiscovered
}

func (spm *SessionPeerManager) Peers() []peer.ID {
	spm.plk.RLock()
	defer spm.plk.RUnlock()

	peers := make([]peer.ID, 0, len(spm.peers))
	for p := range spm.peers {
		peers = append(peers, p)
	}

	return peers
}

func (spm *SessionPeerManager) HasPeers() bool {
	spm.plk.RLock()
	defer spm.plk.RUnlock()

	return len(spm.peers) > 0
}

func (spm *SessionPeerManager) HasPeer(p peer.ID) bool {
	spm.plk.RLock()
	defer spm.plk.RUnlock()

	_, ok := spm.peers[p]
	return ok
}

// Shutdown untags all the peers
func (spm *SessionPeerManager) Shutdown() {
	spm.plk.Lock()
	defer spm.plk.Unlock()

	// Untag the peers with the ConnectionManager so that it can release
	// connections to those peers
	for p := range spm.peers {
		spm.tagger.UntagPeer(p, spm.tag)
		spm.tagger.Unprotect(p, spm.tag)
	}
}
