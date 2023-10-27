package decision

import (
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

type peerLedger struct {
	cids map[cid.Cid]map[peer.ID]struct{}
}

func newPeerLedger() *peerLedger {
	return &peerLedger{cids: make(map[cid.Cid]map[peer.ID]struct{})}
}

func (l *peerLedger) Wants(p peer.ID, k cid.Cid) {
	m, ok := l.cids[k]
	if !ok {
		m = make(map[peer.ID]struct{})
		l.cids[k] = m
	}
	m[p] = struct{}{}
}

func (l *peerLedger) CancelWant(p peer.ID, k cid.Cid) {
	m, ok := l.cids[k]
	if !ok {
		return
	}
	delete(m, p)
	if len(m) == 0 {
		delete(l.cids, k)
	}
}

func (l *peerLedger) Peers(k cid.Cid) []peer.ID {
	m, ok := l.cids[k]
	if !ok {
		return nil
	}
	peers := make([]peer.ID, 0, len(m))
	for p := range m {
		peers = append(peers, p)
	}
	return peers
}
