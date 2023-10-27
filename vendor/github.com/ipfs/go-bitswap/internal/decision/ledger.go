package decision

import (
	"sync"

	pb "github.com/ipfs/go-bitswap/message/pb"
	wl "github.com/ipfs/go-bitswap/wantlist"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

func newLedger(p peer.ID) *ledger {
	return &ledger{
		wantList: wl.New(),
		Partner:  p,
	}
}

// Keeps the wantlist for the partner. NOT threadsafe!
type ledger struct {
	// Partner is the remote Peer.
	Partner peer.ID

	// wantList is a (bounded, small) set of keys that Partner desires.
	wantList *wl.Wantlist

	lk sync.RWMutex
}

func (l *ledger) Wants(k cid.Cid, priority int32, wantType pb.Message_Wantlist_WantType) {
	log.Debugf("peer %s wants %s", l.Partner, k)
	l.wantList.Add(k, priority, wantType)
}

func (l *ledger) CancelWant(k cid.Cid) bool {
	return l.wantList.Remove(k)
}

func (l *ledger) WantListContains(k cid.Cid) (wl.Entry, bool) {
	return l.wantList.Contains(k)
}

func (l *ledger) Entries() []wl.Entry {
	return l.wantList.Entries()
}
